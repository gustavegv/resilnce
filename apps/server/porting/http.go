package porting

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gustavegv/resilnce/apps/server/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// The sole internal-tool membership policy, enforced on every API request.
func Allowed(email string) bool { return strings.EqualFold(email, "gustavengvall@yahoo.se") }

type HTTP struct {
	Service      *Service
	Authenticate func(*http.Request) (string, error)
	Origin       string
}

func (h HTTP) Register(mux *http.ServeMux) {
	mux.Handle("/internal/porting/", h)
}

func (h HTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	actor, err := h.Authenticate(r)
	if err != nil {
		http.Error(w, "Sign in again with a verified email.", http.StatusUnauthorized)
		return
	}
	if !Allowed(actor) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if r.Method != "GET" {
		origin, parseErr := url.Parse(h.Origin)
		if parseErr != nil || origin.Host == "" || r.Header.Get("Origin") != origin.Scheme+"://"+origin.Host {
			http.Error(w, "Invalid mutation origin", http.StatusForbidden)
			return
		}
		mediaType, _, mediaErr := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if mediaErr != nil || mediaType != "application/json" {
			http.Error(w, "Expected application/json", http.StatusUnsupportedMediaType)
			return
		}
	}
	path := strings.TrimPrefix(r.URL.Path, "/internal/porting/")
	var result any
	responseStatus := http.StatusOK
	if r.Method == "POST" {
		responseStatus = http.StatusAccepted
	}
	switch {
	case path == "schema" && r.Method == "GET":
		result, err = migrations.Inspect(r.Context(), h.Service.Pool)
	case path == "schema/apply" && r.Method == "POST":
		var body struct{}
		if !decode(w, r, &body) {
			return
		}
		// Schema setup must work before the job tables exist. Bound this small,
		// atomic DDL transaction within the HTTP server's 60-second write timeout.
		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()
		err = migrations.Apply(ctx, h.Service.Pool)
		if err == nil {
			result, err = migrations.Inspect(ctx, h.Service.Pool)
		}
		responseStatus = http.StatusOK
	case path == "overview" && r.Method == "GET":
		inventory := []Inventory{}
		recent := []Run{}
		var schema migrations.Status
		schema, err = migrations.Inspect(r.Context(), h.Service.Pool)
		if err == nil && schema.Ready {
			inventory, err = h.Service.Inventory(r.Context())
		}
		if err == nil && schema.Ready {
			recent, err = h.Service.RecentRuns(r.Context())
		}
		result = map[string]any{"schema": schema, "applyEnabled": h.Service.AllowApply, "inventory": inventory, "runs": recent}
	case (path == "audits" || path == "runs") && r.Method == "POST":
		if !h.requireSchema(w, r) {
			return
		}
		var body struct {
			Owner   string  `json:"owner"`
			Options Options `json:"options"`
		}
		if !decode(w, r, &body) {
			return
		}
		if err = body.Options.Validate(); err != nil {
			http.Error(w, "Invalid import options: "+err.Error(), 400)
			return
		}
		mode := "apply"
		if path == "audits" {
			mode = "audit"
		}
		var id string
		id, err = h.Service.StartRun(r.Context(), actor, mode, body.Owner, body.Options)
		result = map[string]string{"id": id}
	case strings.HasPrefix(path, "runs/"):
		if !h.requireSchema(w, r) {
			return
		}
		parts := strings.Split(strings.TrimPrefix(path, "runs/"), "/")
		if !validID(parts[0]) {
			http.Error(w, "Invalid run ID", 400)
			return
		}
		if len(parts) == 1 && r.Method == "GET" {
			result, err = h.Service.GetRun(r.Context(), parts[0])
		} else if len(parts) == 2 && parts[1] == "resume" && r.Method == "POST" {
			var body struct{}
			if !decode(w, r, &body) {
				return
			}
			err = h.Service.ResumeRun(r.Context(), parts[0])
			result = map[string]string{"id": parts[0]}
		} else {
			http.Error(w, "Unsupported method or path", 405)
			return
		}
	default:
		http.Error(w, "Unsupported method or path", 405)
		return
	}
	if err != nil {
		var layoutError *migrations.LayoutError
		var databaseError *pgconn.PgError
		if errors.As(err, &layoutError) {
			http.Error(w, layoutError.Error(), http.StatusConflict)
		} else if path == "schema/apply" && errors.As(err, &databaseError) && databaseError.Code == "42501" {
			http.Error(w, "The backend database role needs permission to create the new schema and tables. Update its database permissions, then retry.", http.StatusConflict)
		} else if errors.Is(err, context.DeadlineExceeded) {
			http.Error(w, "Database operation timed out. Refresh the layout status before retrying.", http.StatusGatewayTimeout)
		} else if errors.Is(err, ErrApplyDisabled) || errors.Is(err, ErrNotResumable) {
			http.Error(w, err.Error(), 409)
		} else if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Run not found", 404)
		} else {
			log.Printf("porting API: %v", err)
			http.Error(w, "Porting request failed; inspect server logs.", 500)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(responseStatus)
	json.NewEncoder(w).Encode(result)
}

func (h HTTP) requireSchema(w http.ResponseWriter, r *http.Request) bool {
	schema, err := migrations.Inspect(r.Context(), h.Service.Pool)
	if err != nil {
		log.Printf("porting schema check: %v", err)
		http.Error(w, "Database layout check failed; inspect server logs.", 500)
		return false
	}
	if !schema.Ready {
		http.Error(w, "Database layout is not ready. Check the layout and apply pending migrations in the porting tool.", 409)
		return false
	}
	return true
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		http.Error(w, "Invalid request body", 400)
		return false
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "Expected one JSON object", 400)
		return false
	}
	return true
}

func validID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, c := range id {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

type Inventory struct {
	Owner  string `json:"owner"`
	Table  string `json:"table"`
	Source int    `json:"source"`
	Mapped int    `json:"mapped"`
}

func (s *Service) Inventory(ctx context.Context) ([]Inventory, error) {
	rows, err := s.Pool.Query(ctx, `WITH source AS (
 SELECT coalesce(mail,'') owner,'User' AS tab,count(*) n FROM public."User" GROUP BY mail UNION ALL
 SELECT coalesce(mail,''),'Exercise',count(*) FROM public."Exercise" GROUP BY mail UNION ALL
 SELECT coalesce(mail,''),'Session',count(*) FROM public."Session" GROUP BY mail UNION ALL
 SELECT coalesce(e.mail,s.mail,''),'SesExercise',count(*) FROM public."SesExercise" t LEFT JOIN public."Exercise" e USING(ex_id) LEFT JOIN public."Session" s USING(ses_id) GROUP BY coalesce(e.mail,s.mail,'') UNION ALL
 SELECT coalesce(e.mail,''),'History',count(*) FROM public."History" t LEFT JOIN public."Exercise" e USING(ex_id) GROUP BY e.mail UNION ALL
 SELECT coalesce(e.mail,s.mail,''),'Set',count(*) FROM public."Set" t LEFT JOIN public."Exercise" e USING(ex_id) LEFT JOIN public."Session" s USING(ses_id) GROUP BY coalesce(e.mail,s.mail,'')
), mapped AS (SELECT owner_key,source_table,count(*) n FROM internal.porting_map WHERE migration_version=$1 AND (destinations->>'deletedUnused') IS DISTINCT FROM 'true' GROUP BY owner_key,source_table)
 SELECT s.owner,s.tab,s.n,coalesce(m.n,0) FROM source s LEFT JOIN mapped m ON m.owner_key=s.owner AND m.source_table=s.tab ORDER BY s.owner,s.tab`, Version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Inventory{}
	for rows.Next() {
		var i Inventory
		if err := rows.Scan(&i.Owner, &i.Table, &i.Source, &i.Mapped); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
