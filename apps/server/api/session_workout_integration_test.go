//go:build integration

package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/gustavegv/resilnce/apps/server/application"
	"github.com/gustavegv/resilnce/apps/server/internal/testdb"
	"github.com/gustavegv/resilnce/apps/server/migrations"
	"github.com/gustavegv/resilnce/apps/server/training"
)

func TestSessionMaintenanceProtectsActiveSourceAndPreservesWorkoutSnapshot(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if err != nil {
		t.Fatal(err)
	}
	w := request(t, h, "POST", "/api/training/sessions/create", manualPlan, "https://app.example", nil)
	var plan training.SessionResult
	if err = json.Unmarshal(w.Body.Bytes(), &plan); err != nil || w.Code != 200 {
		t.Fatal(w.Code, w.Body.String(), err)
	}
	path := "/api/training/sessions/" + plan.Session.ID
	start := fmt.Sprintf(`{"operationId":"10000000-0000-4000-8000-000000000030","sessionId":%q,"expectedSessionRevision":"1"}`, plan.Session.ID)
	w = request(t, h, "POST", "/api/training/workouts/start", start, "https://app.example", nil)
	var started training.WorkoutResult
	if err = json.Unmarshal(w.Body.Bytes(), &started); err != nil || w.Code != 200 {
		t.Fatal(w.Code, w.Body.String(), err)
	}
	readPlan := func() training.Session {
		t.Helper()
		w := request(t, h, "GET", path, "", "", nil)
		var body struct{ Session training.Session }
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || w.Code != 200 {
			t.Fatal(w.Code, w.Body.String(), err)
		}
		return body.Session
	}
	if got := readPlan(); got.LastPerformedAt != nil || got.LastPerformedStatus != "never" {
		t.Fatal("start invented recency", got)
	}
	update := fmt.Sprintf(`{"operationId":"10000000-0000-4000-8000-000000000031","expectedRevision":"1","name":"Renamed source","items":[{"kind":"retainPlacement","placementId":%q},{"kind":"createExercise","name":"Squat","configuration":{"prescribedSets":1,"initialWeight":"60"}}]}`, plan.Session.Placements[1].ID)
	w = request(t, h, "POST", path+"/update", update, "https://app.example", nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var active struct{ Workout training.Workout }
	w = request(t, h, "GET", "/api/training/workouts/active", "", "", nil)
	if err = json.Unmarshal(w.Body.Bytes(), &active); err != nil || w.Code != 200 {
		t.Fatal(w.Code, w.Body.String(), err)
	}
	if active.Workout.ID != started.Workout.ID || active.Workout.SourceSession.Name != "Renamed source" || len(active.Workout.Exercises) != 2 {
		t.Fatal("source/snapshot", w.Body.String())
	}
	for i, a := range active.Workout.Exercises {
		original := started.Workout.Exercises[i]
		if a.ID != original.ID || a.VariantID != original.VariantID || a.Position != original.Position || a.Sets[0].ID != original.Sets[0].ID || a.Sets[0].TargetWeight != original.Sets[0].TargetWeight {
			t.Fatal("plan edit changed captured composition", w.Body.String())
		}
	}
	deletion := `{"operationId":"10000000-0000-4000-8000-000000000032","expectedRevision":"2"}`
	w = request(t, h, "POST", path+"/delete", deletion, "https://app.example", nil)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "active-source-session") {
		t.Fatal("active source deleted", w.Code, w.Body.String())
	}
	// The completion/end seams belong to later slices. Establish an ended workout
	// with durable performance as fixture, then observe preservation through HTTP.
	if _, err = pool.Exec(ctx, `UPDATE public.workout_set SET state='performed',recorded_reps=0,recorded_weight=target_weight,recorded_at=now() WHERE workout_exercise_id=$1`, started.Workout.Exercises[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE public.workout_exercise SET state='completed',completed_at=now() WHERE id=$1`, started.Workout.Exercises[0].ID); err != nil {
		t.Fatal(err)
	}
	if got := readPlan(); got.LastPerformedAt == nil || got.LastPerformedStatus != "dated" {
		t.Fatal("completed attempt missing recency", got)
	}
	if _, err = pool.Exec(ctx, `UPDATE public.workout SET status='ended',ended_at=now() WHERE id=$1`, started.Workout.ID); err != nil {
		t.Fatal(err)
	}
	w = request(t, h, "POST", path+"/delete", deletion, "https://app.example", nil)
	if w.Code != 200 {
		t.Fatal("ended source deletion", w.Code, w.Body.String())
	}
	w = request(t, h, "GET", "/api/training/workouts/"+started.Workout.ID, "", "", nil)
	if err = json.Unmarshal(w.Body.Bytes(), &active); err != nil || w.Code != 200 {
		t.Fatal(w.Code, w.Body.String(), err)
	}
	if active.Workout.SourceSession != nil || len(active.Workout.Exercises) != 2 || active.Workout.Exercises[0].State != "completed" || active.Workout.Exercises[0].Sets[0].RecordedReps == nil || *active.Workout.Exercises[0].Sets[0].RecordedReps != 0 || *active.Workout.Exercises[0].Sets[0].RecordedWeight != "40.50" {
		t.Fatal("deletion lost workout/performance or retained source", w.Body.String())
	}
	w = request(t, h, "GET", "/api/training/exercises/"+plan.Session.Placements[0].Exercise.ID, "", "", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), plan.Session.Placements[0].Variant.TargetSets[0].ID) {
		t.Fatal("deletion lost original removed variant targets", w.Code, w.Body.String())
	}
	// A different, now-empty plan cannot start even though an ended workout exists.
	var empty training.SessionResult
	create := strings.Replace(strings.Replace(manualPlan, "000000000001", "000000000033", 1), "Bench press", "Other bench", 1)
	create = strings.Replace(create, `"name":"Row"`, `"name":"Other row"`, 1)
	w = request(t, h, "POST", "/api/training/sessions/create", create, "https://app.example", nil)
	if err = json.Unmarshal(w.Body.Bytes(), &empty); err != nil || w.Code != 200 {
		t.Fatal(w.Code, w.Body.String(), err)
	}
	w = request(t, h, "POST", "/api/training/sessions/"+empty.Session.ID+"/update", `{"operationId":"10000000-0000-4000-8000-000000000034","expectedRevision":"1","name":"Empty","items":[]}`, "https://app.example", nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	start = fmt.Sprintf(`{"operationId":"10000000-0000-4000-8000-000000000035","sessionId":%q,"expectedSessionRevision":"2"}`, empty.Session.ID)
	w = request(t, h, "POST", "/api/training/workouts/start", start, "https://app.example", nil)
	if w.Code != 422 {
		t.Fatal("empty plan started", w.Code, w.Body.String())
	}
}
