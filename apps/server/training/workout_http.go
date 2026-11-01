package training

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gustavegv/resilnce/apps/server/application"
)

func (h HTTP) activeWorkout(w http.ResponseWriter, r *http.Request) {
	v, err := h.Service.ActiveWorkout(r.Context(), owner(r))
	respond(w, struct {
		Workout *Workout `json:"workout"`
	}{v}, err)
}
func (h HTTP) getWorkout(w http.ResponseWriter, r *http.Request) {
	v, err := h.Service.GetWorkout(r.Context(), owner(r), r.URL.Path[len("/api/training/workouts/"):])
	respond(w, struct {
		Workout Workout `json:"workout"`
	}{v}, err)
}
func (h HTTP) startWorkout(w http.ResponseWriter, r *http.Request) {
	var c *StartWorkout
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil || c == nil {
		application.WriteError(w, 400, "malformed-request", "Send a workout start command object", nil)
		return
	}
	if err := d.Decode(new(any)); err != io.EOF {
		application.WriteError(w, 400, "malformed-request", "Send one workout start command", nil)
		return
	}
	v, err := h.Service.StartWorkout(r.Context(), owner(r), *c)
	respond(w, v, err)
}

func (h HTTP) recordSet(w http.ResponseWriter, r *http.Request) {
	var c *RecordSet
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil || c == nil {
		application.WriteError(w, 400, "malformed-request", "Send one explicit set recording command object", nil)
		return
	}
	if err := d.Decode(new(any)); err != io.EOF {
		application.WriteError(w, 400, "malformed-request", "Send one set recording command", nil)
		return
	}
	v, err := h.Service.RecordSet(r.Context(), owner(r), *c)
	respond(w, v, err)
}

func (h HTTP) updateConfiguration(w http.ResponseWriter, r *http.Request) {
	var c *UpdateConfiguration
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil || c == nil {
		application.WriteError(w, 400, "malformed-request", "Send one configuration edit object", nil)
		return
	}
	if err := d.Decode(new(any)); err != io.EOF {
		application.WriteError(w, 400, "malformed-request", "Send one configuration edit", nil)
		return
	}
	v, err := h.Service.UpdateConfiguration(r.Context(), owner(r), *c)
	respond(w, v, err)
}
