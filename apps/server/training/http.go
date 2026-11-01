package training

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gustavegv/resilnce/apps/server/application"
	"github.com/gustavegv/resilnce/apps/server/predefined"
)

type HTTP struct {
	Service  *Service
	Boundary application.Boundary
}

func (h HTTP) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/training/stats", h.Boundary.Handler("GET", application.ExerciseCentric, false, h.stats))

	mux.HandleFunc("/api/training/workouts/quit", h.Boundary.Handler("POST", application.ExerciseCentric, true, h.quitWorkout))
	mux.HandleFunc("/api/training/workouts/complete", h.Boundary.Handler("POST", application.ExerciseCentric, true, h.completeAttempt))
	mux.HandleFunc("/api/training/workouts/configuration", h.Boundary.Handler("POST", application.ExerciseCentric, true, h.updateConfiguration))
	mux.HandleFunc("/api/training/workouts/sets/record", h.Boundary.Handler("POST", application.ExerciseCentric, true, h.recordSet))
	mux.HandleFunc("/api/training/catalog", h.Boundary.Handler("GET", application.ExerciseCentric, false, func(w http.ResponseWriter, r *http.Request) {
		respond(w, struct {
			MuscleGroups []predefined.MuscleGroup `json:"muscleGroups"`
		}{predefined.MuscleGroups()}, nil)
	}))
	mux.HandleFunc("/api/training/workouts/start", h.Boundary.Handler("POST", application.ExerciseCentric, true, h.startWorkout))
	mux.HandleFunc("/api/training/workouts/active", h.Boundary.Handler("GET", application.ExerciseCentric, false, h.activeWorkout))
	mux.HandleFunc("/api/training/workouts/", h.Boundary.Handler("GET", application.ExerciseCentric, false, h.getWorkout))
	mux.HandleFunc("/api/training/exercises", h.Boundary.Handler("GET", application.ExerciseCentric, false, h.searchExercises))
	mux.HandleFunc("/api/training/exercises/", h.Boundary.Handler("GET", application.ExerciseCentric, false, h.getExercise))
	mux.HandleFunc("/api/training/sessions", h.Boundary.Handler("GET", application.ExerciseCentric, false, h.list))
	mux.HandleFunc("/api/training/sessions/", h.sessionRoute)
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

func (h HTTP) searchExercises(w http.ResponseWriter, r *http.Request) {
	v, err := h.Service.SearchExercises(r.Context(), owner(r), r.URL.Query().Get("q"))
	respond(w, struct {
		Exercises []ExerciseSelection `json:"exercises"`
	}{v}, err)
}
func (h HTTP) getExercise(w http.ResponseWriter, r *http.Request) {
	v, err := h.Service.GetExercise(r.Context(), owner(r), r.URL.Path[len("/api/training/exercises/"):])
	respond(w, v, err)
}

func (h HTTP) sessionRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/training/sessions/")
	parts := strings.Split(path, "/")
	if len(parts) == 1 {
		h.Boundary.Handler("GET", application.ExerciseCentric, false, h.get)(w, r)
		return
	}
	if len(parts) == 2 && (parts[1] == "update" || parts[1] == "delete") {
		h.Boundary.Handler("POST", application.ExerciseCentric, true, func(w http.ResponseWriter, r *http.Request) {
			var value any
			if parts[1] == "update" {
				value = &UpdateSession{}
			} else {
				value = &DeleteSession{}
			}
			d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256*1024))
			d.DisallowUnknownFields()
			if err := d.Decode(&value); err != nil {
				var domain *Error
				if errors.As(err, &domain) {
					respond(w, nil, err)
					return
				}
				respond(w, nil, &Error{400, "malformed-request", "Send a valid session command", nil})
				return
			}
			if value == nil {
				respond(w, nil, &Error{400, "malformed-request", "Send a session command object", nil})
				return
			}
			if err := d.Decode(new(any)); err != io.EOF {
				respond(w, nil, &Error{400, "malformed-request", "Send one session command", nil})
				return
			}
			if c, ok := value.(*UpdateSession); ok {
				v, err := h.Service.UpdateSession(r.Context(), owner(r), parts[0], *c)
				respond(w, v, err)
			} else {
				v, err := h.Service.DeleteSession(r.Context(), owner(r), parts[0], *value.(*DeleteSession))
				respond(w, v, err)
			}
		})(w, r)
		return
	}
	http.NotFound(w, r)
}

func (h HTTP) stats(w http.ResponseWriter, r *http.Request) {
	v, err := h.Service.Stats(r.Context(), owner(r))
	respond(w, struct {
		Tracks []StatsTrack `json:"tracks"`
	}{v}, err)
}
