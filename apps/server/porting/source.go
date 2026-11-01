// Package porting owns the legacy transformation used by both the CLI and API.
package porting

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
)

const Version = "exercise-centric-v1"

type Row struct {
	Table       string          `json:"table"`
	Key         json.RawMessage `json:"key"`
	Owner       string          `json:"owner"`
	Raw         json.RawMessage `json:"data"`
	Data        Source          `json:"-"`
	DecodeError string          `json:"-"`
}

// Pointers preserve legacy NULLs, including NULL array elements.
type Source struct {
	Mail           *string        `json:"mail"`
	Name           *string        `json:"ex_name"`
	SessionName    *string        `json:"ses_name"`
	NormalizedName string         `json:"_normalized_name"`
	ExID           int            `json:"ex_id"`
	SesID          int            `json:"ses_id"`
	Increment      *json.Number   `json:"auto_inc"`
	Threshold      *int           `json:"rep_threshold"`
	SetCount       *int           `json:"set_count"`
	Reps           []*int         `json:"rep_per_set"`
	Weights        []*json.Number `json:"weight_per_set"`
	Rest           *int           `json:"rest_seconds"`
	Order          *int           `json:"order"`
	ActiveSession  *string        `json:"active_session"`
	SessionLastRan *string        `json:"date_last_ran"`
	SaveDate       string         `json:"save_date"`
}

func sourceRow(table string, raw json.RawMessage, owner string) (Row, error) {
	r := Row{Table: table, Raw: raw, Owner: owner}
	// A multidimensional array or PostgreSQL NaN must block its owner, not abort
	// the inventory and conceal otherwise valid accounts.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return r, err
	}
	arrays := map[string]any{"rep_per_set": &r.Data.Reps, "weight_per_set": &r.Data.Weights, "auto_inc": &r.Data.Increment}
	for name, target := range arrays {
		if value, ok := fields[name]; ok {
			if err := json.Unmarshal(value, target); err != nil {
				r.DecodeError = "Source contains unsupported arrays or non-finite numeric values."
			}
			delete(fields, name)
		}
	}
	core, _ := json.Marshal(fields)
	if err := json.Unmarshal(core, &r.Data); err != nil {
		return r, err
	}
	var key []any
	switch table {
	case "User":
		key = []any{r.Data.Mail}
	case "Exercise":
		key = []any{r.Data.ExID}
	case "Session":
		key = []any{r.Data.SesID}
	case "SesExercise":
		key = []any{r.Data.SesID, r.Data.ExID}
	case "History":
		key = []any{r.Data.ExID, r.Data.SaveDate}
	case "Set":
		key = []any{r.Data.SesID, r.Data.ExID, r.Data.SetCount}
	default:
		return r, fmt.Errorf("unknown source table %s", table)
	}
	r.Key, _ = json.Marshal(key)
	return r, nil
}

func (r Row) identity() string { return r.Table + ":" + string(r.Key) }

// Snapshot includes every source row, including NULL owners and inconsistent
// links. Never use an inner join here: it would hide unresolved records.
func snapshot(ctx context.Context, tx pgx.Tx) ([]Row, error) {
	queries := []struct{ table, sql string }{
		{"User", `SELECT to_jsonb(t), coalesce(t.mail,'') FROM public."User" t ORDER BY mail`},
		{"Exercise", `SELECT to_jsonb(t)||jsonb_build_object('_normalized_name',lower(btrim(t.ex_name))), coalesce(t.mail,'') FROM public."Exercise" t ORDER BY ex_id`},
		{"Session", `SELECT to_jsonb(t), coalesce(t.mail,'') FROM public."Session" t ORDER BY ses_id`},
		{"SesExercise", `SELECT to_jsonb(t), coalesce(e.mail,s.mail,'') FROM public."SesExercise" t LEFT JOIN public."Exercise" e USING(ex_id) LEFT JOIN public."Session" s USING(ses_id) ORDER BY t.ses_id,t.ex_id`},
		{"History", `SELECT to_jsonb(t), coalesce(e.mail,'') FROM public."History" t LEFT JOIN public."Exercise" e USING(ex_id) ORDER BY t.ex_id,t.save_date`},
		{"Set", `SELECT to_jsonb(t), coalesce(e.mail,s.mail,'') FROM public."Set" t LEFT JOIN public."Exercise" e USING(ex_id) LEFT JOIN public."Session" s USING(ses_id) ORDER BY t.ses_id,t.ex_id,t.set_count`},
	}
	var out []Row
	for _, q := range queries {
		rows, err := tx.Query(ctx, q.sql)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var raw json.RawMessage
			var owner string
			if err = rows.Scan(&raw, &owner); err != nil {
				rows.Close()
				return nil, err
			}
			r, err := sourceRow(q.table, raw, owner)
			if err != nil {
				rows.Close()
				return nil, err
			}
			out = append(out, r)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func owners(rows []Row, scope string) []string {
	set := map[string]bool{}
	for _, r := range rows {
		if scope == "" || r.Owner == scope {
			set[r.Owner] = true
		}
	}
	out := make([]string, 0, len(set))
	for owner := range set {
		out = append(out, owner)
	}
	sort.Strings(out)
	return out
}
