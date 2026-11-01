package training

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gustavegv/resilnce/apps/server/application"
)

type HTTP struct {
	Service  *Service
	Boundary application.Boundary
}

func (h HTTP) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/training/sessions", h.Boundary.Handler("GET", application.ExerciseCentric, false, h.list))
	mux.HandleFunc("/api/training/sessions/", h.Boundary.Handler("GET", application.ExerciseCentric, false, h.get))
	mux.HandleFunc("/api/training/sessions/create", h.Boundary.Handler("POST", application.ExerciseCentric, true, h.create))
}
func respond(w http.ResponseWriter, value any, err error) {
	if err != nil {
		var domain *Error
		if errors.As(err, &domain) {
			application.WriteError(w, domain.Status, domain.Code, domain.Message, domain.Details)
		} else {
			application.WriteError(w, 500, "internal-error", "Unable to save or load training data. Retry the request", nil)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(value)
}
func owner(r *http.Request) string { v, _ := application.FromContext(r.Context()); return v.Email }
func (h HTTP) list(w http.ResponseWriter, r *http.Request) {
	v, err := h.Service.ListSessions(r.Context(), owner(r))
	respond(w, struct {
		Sessions []Session `json:"sessions"`
	}{v}, err)
}
func (h HTTP) get(w http.ResponseWriter, r *http.Request) {
	v, err := h.Service.GetSession(r.Context(), owner(r), r.URL.Path[len("/api/training/sessions/"):])
	respond(w, struct {
		Session Session `json:"session"`
	}{v}, err)
}
func (h HTTP) create(w http.ResponseWriter, r *http.Request) {
	var c *CreateSession
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256*1024))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		var domain *Error
		if errors.As(err, &domain) {
			respond(w, nil, err)
			return
		}
		application.WriteError(w, 400, "malformed-request", "Send a valid session command", nil)
		return
	}
	if c == nil {
		application.WriteError(w, 400, "malformed-request", "Send a session command object", nil)
		return
	}
	if err := d.Decode(new(any)); err != io.EOF {
		application.WriteError(w, 400, "malformed-request", "Send one session command", nil)
		return
	}
	v, err := h.Service.CreateSession(r.Context(), owner(r), *c)
	respond(w, v, err)
}
