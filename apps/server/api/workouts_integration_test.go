//go:build integration

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gustavegv/resilnce/apps/server/application"
	"github.com/gustavegv/resilnce/apps/server/internal/testdb"
	"github.com/gustavegv/resilnce/apps/server/migrations"
	"github.com/gustavegv/resilnce/apps/server/training"
)

func TestWorkoutStartsSnapshotAndResumesByWorkoutIdentity(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if err != nil {
		t.Fatal(err)
	}
	var plan training.SessionResult
	w := request(t, h, "POST", "/api/training/sessions/create", manualPlan, "https://app.example", nil)
	if err := json.Unmarshal(w.Body.Bytes(), &plan); w.Code != http.StatusOK || err != nil {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	empty := request(t, h, "GET", "/api/training/workouts/active", "", "", nil)
	if empty.Code != 200 || empty.Body.String() != "{\"workout\":null}\n" {
		t.Fatalf("empty active: %d %s", empty.Code, empty.Body.String())
	}
	body := `{"operationId":"90000000-0000-4000-8000-000000000001","sessionId":"` + plan.Session.ID + `","expectedSessionRevision":"1"}`
	w = request(t, h, "POST", "/api/training/workouts/start", body, "https://app.example", nil)
	var result struct {
		Operation training.Acknowledgment
		Workout   struct {
			ID, Status, Revision string
			SourceSession        struct{ ID, Name string }
			Exercises            []struct {
				ID, State, ConfigRevision string
				Position                  int
				Sets                      []struct {
					ID                  string
					TargetSetID         *string
					State, TargetWeight string
					TargetReps          int
					RecordedReps        *int
					RecordedWeight      *string
					RecordedAt          *string
				}
			}
		}
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); w.Code != 200 || err != nil {
		t.Fatalf("start: %d %s", w.Code, w.Body.String())
	}
	a := result.Workout
	if a.ID == plan.Session.ID || len(a.ID) != 36 || a.Status != "active" || a.Revision != "1" || a.SourceSession.ID != plan.Session.ID || a.SourceSession.Name != "Upper body" || len(a.Exercises) != 2 {
		t.Fatalf("workout: %s", w.Body.String())
	}
	first := a.Exercises[0]
	if len(first.ID) != 36 || first.Position != 1 || first.State != "draft" || first.ConfigRevision != "1" || len(first.Sets) != 2 {
		t.Fatalf("attempt: %s", w.Body.String())
	}
	for i, set := range first.Sets {
		if len(set.ID) != 36 || set.TargetSetID == nil || *set.TargetSetID != plan.Session.Placements[0].Variant.TargetSets[i].ID || set.State != "pending" || set.TargetWeight != "40.50" || set.TargetReps != 12 || set.RecordedReps != nil || set.RecordedWeight != nil || set.RecordedAt != nil {
			t.Fatalf("pending target/actuals: %+v", set)
		}
	}
	for _, path := range []string{"/api/training/workouts/active", "/api/training/workouts/" + a.ID} {
		read := request(t, h, "GET", path, "", "", nil)
		var actual map[string]json.RawMessage
		if err := json.Unmarshal(read.Body.Bytes(), &actual); read.Code != 200 || err != nil {
			t.Fatalf("resume: %d %s", read.Code, read.Body.String())
		}
		var initial map[string]json.RawMessage
		json.Unmarshal(w.Body.Bytes(), &initial)
		if string(actual["workout"]) != string(initial["workout"]) {
			t.Fatalf("resume changed workout: %s", read.Body.String())
		}
	}
}

func TestConcurrentWorkoutStartsResolveOneSourceAndReplay(t *testing.T) {
	for _, scenario := range []struct {
		name                    string
		samePlan, sameOperation bool
	}{{"same plan", true, false}, {"different plans", false, false}, {"identical retry", true, true}} {
		t.Run(scenario.name, func(t *testing.T) {
			pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
			if err := migrations.Apply(ctx, pool); err != nil {
				t.Fatal(err)
			}
			h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
			if err != nil {
				t.Fatal(err)
			}
			var p1, p2 training.SessionResult
			w := request(t, h, "POST", "/api/training/sessions/create", manualPlan, "https://app.example", nil)
			json.Unmarshal(w.Body.Bytes(), &p1)
			second := strings.NewReplacer("10000000-0000-4000-8000-000000000001", "10000000-0000-4000-8000-000000000002", "Upper body", "Lower body", "Bench press", "Squat", "Row", "Deadlift").Replace(manualPlan)
			w = request(t, h, "POST", "/api/training/sessions/create", second, "https://app.example", nil)
			if w.Code != 200 {
				t.Fatal(w.Body.String())
			}
			json.Unmarshal(w.Body.Bytes(), &p2)
			if scenario.samePlan {
				p2 = p1
			}
			commands := []string{`{"operationId":"90000000-0000-4000-8000-000000000001","sessionId":"` + p1.Session.ID + `","expectedSessionRevision":"1"}`, `{"operationId":"90000000-0000-4000-8000-000000000002","sessionId":"` + p2.Session.ID + `","expectedSessionRevision":"1"}`}
			if scenario.sameOperation {
				commands[1] = commands[0]
			}
			var responses [2]*httptest.ResponseRecorder
			var wg sync.WaitGroup
			for i := range responses {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					responses[i] = request(t, h, "POST", "/api/training/workouts/start", commands[i], "https://app.example", nil)
				}(i)
			}
			wg.Wait()
			var results [2]training.WorkoutResult
			for i, w := range responses {
				if err := json.Unmarshal(w.Body.Bytes(), &results[i]); err != nil || w.Code != 200 {
					t.Fatalf("concurrent start: %d %s", w.Code, w.Body.String())
				}
			}
			if results[0].Workout.ID != results[1].Workout.ID || results[0].Workout.SourceSession.ID != results[1].Workout.SourceSession.ID {
				t.Fatal("simultaneous workouts or hidden source")
			}
			for i, body := range commands {
				w := request(t, h, "POST", "/api/training/workouts/start", body, "https://app.example", nil)
				var replay training.WorkoutResult
				json.Unmarshal(w.Body.Bytes(), &replay)
				if w.Code != 200 || replay.Workout.ID != results[i].Workout.ID || replay.Operation != results[i].Operation {
					t.Fatalf("replay: %d %s", w.Code, w.Body.String())
				}
			}
			changed := strings.Replace(commands[0], `"expectedSessionRevision":"1"`, `"expectedSessionRevision":"2"`, 1)
			w = request(t, h, "POST", "/api/training/workouts/start", changed, "https://app.example", nil)
			if w.Code != 409 || !strings.Contains(w.Body.String(), "operation-mismatch") {
				t.Fatalf("mismatch: %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestWorkoutStartGuardsAndSnapshotIndependence(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO public."User"(mail,name) VALUES('owner@example.com','Owner'),('other@example.com','Other')`); err != nil {
		t.Fatal(err)
	}
	handler := func(email string) http.Handler {
		h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{Model: application.ExerciseCentric}, Resolve: func(*http.Request) (application.Identity, error) { return application.Identity{Email: email}, nil }})
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	h, other := handler("owner@example.com"), handler("other@example.com")
	w := request(t, h, "POST", "/api/training/sessions/create", manualPlan, "https://app.example", nil)
	var plan training.SessionResult
	json.Unmarshal(w.Body.Bytes(), &plan)
	body := `{"operationId":"90000000-0000-4000-8000-000000000001","sessionId":"` + plan.Session.ID + `","expectedSessionRevision":"1"}`
	w = request(t, other, "POST", "/api/training/workouts/start", body, "https://app.example", nil)
	if w.Code != 404 {
		t.Fatalf("cross owner start: %d %s", w.Code, w.Body.String())
	}
	stale := strings.Replace(body, `"expectedSessionRevision":"1"`, `"expectedSessionRevision":"2"`, 1)
	w = request(t, h, "POST", "/api/training/workouts/start", stale, "https://app.example", nil)
	if w.Code != 409 {
		t.Fatalf("stale: %d %s", w.Code, w.Body.String())
	}
	if _, err := pool.Exec(ctx, `UPDATE public.exercise_variant SET archived_at=now() WHERE id=$1`, plan.Session.Placements[0].VariantID); err != nil {
		t.Fatal(err)
	}
	w = request(t, h, "POST", "/api/training/workouts/start", body, "https://app.example", nil)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "resource-unavailable") {
		t.Fatalf("archive: %d %s", w.Code, w.Body.String())
	}
	if active := request(t, h, "GET", "/api/training/workouts/active", "", "", nil); active.Body.String() != "{\"workout\":null}\n" {
		t.Fatal("rejected start left partial workout")
	}
	if _, err := pool.Exec(ctx, `UPDATE public.exercise_variant SET archived_at=NULL,revision=7 WHERE id=$1`, plan.Session.Placements[0].VariantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE public.variant_target_set SET target_weight=50.25,target_reps=9 WHERE id=$1`, plan.Session.Placements[0].Variant.TargetSets[1].ID); err != nil {
		t.Fatal(err)
	}
	w = request(t, h, "POST", "/api/training/workouts/start", body, "https://app.example", nil)
	var started training.WorkoutResult
	json.Unmarshal(w.Body.Bytes(), &started)
	if w.Code != 200 {
		t.Fatalf("retry after rejection: %d %s", w.Code, w.Body.String())
	}
	if started.Workout.Exercises[0].Sets[1].TargetWeight != "50.25" || started.Workout.Exercises[0].Sets[1].TargetReps != 9 || started.Workout.Exercises[0].ConfigRevision != "7" || started.Workout.Exercises[1].RestSeconds != 60 {
		t.Fatalf("current targets/settings: %s", w.Body.String())
	}
	if _, err := pool.Exec(ctx, `DELETE FROM public.session_variant WHERE session_id=$1 AND position=1`, plan.Session.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE public.variant_target_set SET target_weight=70 WHERE id=$1`, plan.Session.Placements[0].Variant.TargetSets[1].ID); err != nil {
		t.Fatal(err)
	}
	w = request(t, h, "GET", "/api/training/workouts/active", "", "", nil)
	var resumed struct{ Workout training.Workout }
	json.Unmarshal(w.Body.Bytes(), &resumed)
	if len(resumed.Workout.Exercises) != 2 || resumed.Workout.Exercises[0].Sets[1].TargetWeight != "50.25" || resumed.Workout.Exercises[0].CurrentVariant.TargetSets[1].TargetWeight != "70.00" {
		t.Fatalf("snapshot changed: %s", w.Body.String())
	}
	w = request(t, other, "GET", "/api/training/workouts/"+started.Workout.ID, "", "", nil)
	if w.Code != 404 {
		t.Fatalf("cross owner read: %d %s", w.Code, w.Body.String())
	}
	w = request(t, other, "GET", "/api/training/workouts/active?mail=owner@example.com", "", "", nil)
	if w.Body.String() != "{\"workout\":null}\n" {
		t.Fatal("active owner override")
	}
	if _, err := pool.Exec(ctx, `UPDATE public.workout SET status='ended',ended_at=now(),revision=2 WHERE id=$1`, started.Workout.ID); err != nil {
		t.Fatal(err)
	}
	w = request(t, h, "POST", "/api/training/workouts/start", body, "https://app.example", nil)
	var replay training.WorkoutResult
	json.Unmarshal(w.Body.Bytes(), &replay)
	if w.Code != 200 || replay.Workout.ID != started.Workout.ID || replay.Workout.Status != "ended" || replay.Operation != started.Operation {
		t.Fatalf("ended replay: %d %s", w.Code, w.Body.String())
	}
	laterBody := strings.Replace(body, "90000000-0000-4000-8000-000000000001", "90000000-0000-4000-8000-000000000003", 1)
	w = request(t, h, "POST", "/api/training/workouts/start", laterBody, "https://app.example", nil)
	var later training.WorkoutResult
	json.Unmarshal(w.Body.Bytes(), &later)
	if w.Code != 200 || later.Workout.ID == started.Workout.ID {
		t.Fatalf("later workout: %d %s", w.Code, w.Body.String())
	}
	w = request(t, h, "POST", "/api/training/workouts/start", body, "https://app.example", nil)
	json.Unmarshal(w.Body.Bytes(), &replay)
	if w.Code != 200 || replay.Workout.ID != started.Workout.ID || replay.Workout.Status != "ended" || replay.Operation != started.Operation {
		t.Fatalf("old start targeted new workout: %d %s", w.Code, w.Body.String())
	}
	if _, err := pool.Exec(ctx, `UPDATE public.workout SET status='ended',ended_at=now(),revision=2 WHERE id=$1`, later.Workout.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM public.session_variant WHERE session_id=$1`, plan.Session.ID); err != nil {
		t.Fatal(err)
	}
	newBody := strings.Replace(body, "90000000-0000-4000-8000-000000000001", "90000000-0000-4000-8000-000000000002", 1)
	w = request(t, h, "POST", "/api/training/workouts/start", newBody, "https://app.example", nil)
	if w.Code != 422 {
		t.Fatalf("empty session: %d %s", w.Code, w.Body.String())
	}
}

func TestWorkoutRestSnapshotMigrationUpgradesExistingAttempt(t *testing.T) {
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
	json.Unmarshal(w.Body.Bytes(), &plan)
	body := `{"operationId":"90000000-0000-4000-8000-000000000001","sessionId":"` + plan.Session.ID + `","expectedSessionRevision":"1"}`
	w = request(t, h, "POST", "/api/training/workouts/start", body, "https://app.example", nil)
	if w.Code != 200 {
		t.Fatalf("start: %d %s", w.Code, w.Body.String())
	}
	// Restore this disposable fixture to pre-006 shape/ledger, keeping attempts.
	// Applied migration checksums are never rewritten.
	if _, err := pool.Exec(ctx, `ALTER TABLE public.workout_exercise DROP COLUMN rest_seconds; DELETE FROM internal.schema_migration WHERE version='006_workout_rest.sql'`); err != nil {
		t.Fatal(err)
	}
	status, err := migrations.Inspect(ctx, pool)
	if err != nil || status.Ready || !status.CanApply {
		t.Fatalf("upgrade readiness: %+v %v", status, err)
	}
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("idempotent apply: %v", err)
	}
	w = request(t, h, "GET", "/api/training/workouts/active", "", "", nil)
	var resumed struct{ Workout training.Workout }
	json.Unmarshal(w.Body.Bytes(), &resumed)
	if w.Code != 200 || len(resumed.Workout.Exercises) != 2 || resumed.Workout.Exercises[1].RestSeconds != 60 || resumed.Workout.Exercises[1].Sets[0].RecordedReps != nil {
		t.Fatalf("rest upgrade/performance: %d %s", w.Code, w.Body.String())
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE public.workout_exercise DROP CONSTRAINT workout_exercise_rest_nonnegative`); err != nil {
		t.Fatal(err)
	}
	status, err = migrations.Inspect(ctx, pool)
	if err != nil || status.Ready || len(status.Issues) == 0 {
		t.Fatalf("missing constraint accepted: %+v %v", status, err)
	}
}
