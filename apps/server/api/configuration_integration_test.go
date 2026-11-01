//go:build integration

package api

import (
	"encoding/json"
	"fmt"
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

func configurationIntent(w training.Workout, op int, patch map[string]any) map[string]any {
	a := w.Exercises[0]
	return map[string]any{"operationId": fmt.Sprintf("a2000000-0000-4000-8000-%012d", op), "workoutId": w.ID, "attemptId": a.ID, "exerciseId": a.Exercise.ID, "variantId": a.VariantID, "expectedWorkoutRevision": w.Revision, "expectedExerciseRevision": a.Exercise.Revision, "expectedVariantRevision": a.CurrentVariant.Revision, "patch": patch}
}
func sendConfiguration(t *testing.T, h http.Handler, c map[string]any) (int, string) {
	t.Helper()
	b, _ := json.Marshal(c)
	p := request(t, h, "POST", "/api/training/workouts/configuration", string(b), "https://app.example", nil)
	return p.Code, p.Body.String()
}
func configuredWorkout(t *testing.T, h http.Handler) training.Workout {
	t.Helper()
	p := request(t, h, "POST", "/api/training/sessions/create", strings.Replace(manualPlan, `"muscleGroupCodes":[]`, `"muscleGroupCodes":["chest"]`, 1), "https://app.example", nil)
	var plan training.SessionResult
	if err := json.Unmarshal(p.Body.Bytes(), &plan); p.Code != 200 || err != nil {
		t.Fatal(p.Body.String())
	}
	p = request(t, h, "POST", "/api/training/workouts/start", `{"operationId":"a1000000-0000-4000-8000-000000000001","sessionId":"`+plan.Session.ID+`","expectedSessionRevision":"1"}`, "https://app.example", nil)
	var out training.WorkoutResult
	if err := json.Unmarshal(p.Body.Bytes(), &out); p.Code != 200 || err != nil {
		t.Fatal(p.Body.String())
	}
	return out.Workout
}
func configurationSaved(t *testing.T, h http.Handler, c map[string]any) training.ConfigurationResult {
	t.Helper()
	status, body := sendConfiguration(t, h, c)
	var out training.ConfigurationResult
	if err := json.Unmarshal([]byte(body), &out); status != 200 || err != nil {
		t.Fatalf("configuration: %d %s", status, body)
	}
	return out
}
func TestSharedConfigurationRenamesAndThresholdApplyAtomically(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if err != nil {
		t.Fatal(err)
	}
	w := configuredWorkout(t, h)
	a := w.Exercises[0]
	saved := configurationSaved(t, h, configurationIntent(w, 1, map[string]any{"exerciseName": " Barbell press ", "variantName": " Heavy day ", "repThreshold": 10}))
	now := saved.Workout.Exercises[0]
	if saved.Exercise.Name != "Barbell press" || saved.Variant.Name != "Heavy day" || saved.Variant.Revision != "2" || now.Exercise.Name != "Barbell press" || now.CurrentVariant.Name != "Heavy day" || now.RepThreshold != 10 || now.CurrentVariant.RepThreshold != 10 || now.ConfigRevision != "2" || now.Exercise.Revision != "2" || saved.Workout.Revision != "2" {
		t.Fatalf("configuration not applied: %+v", now)
	}
	if now.Sets[0].ID != a.Sets[0].ID || now.Sets[0].TargetSetID == nil || *now.Sets[0].TargetSetID != *a.Sets[0].TargetSetID || now.Sets[0].TargetReps != 12 || now.Sets[0].TargetWeight != "40.50" || now.Sets[0].RecordedReps != nil {
		t.Fatal("retained target/set changed")
	}
	p := request(t, h, "GET", "/api/training/sessions/"+w.SourceSession.ID, "", "", nil)
	if !strings.Contains(p.Body.String(), `"name":"Barbell press"`) || !strings.Contains(p.Body.String(), `"repThreshold":10`) {
		t.Fatalf("shared session not current: %s", p.Body.String())
	}
}

func TestSharedConfigurationPreservesMixedWeightsAndActualsUntilExplicitWeightAction(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if err != nil {
		t.Fatal(err)
	}
	w := configuredWorkout(t, h)
	a := w.Exercises[0]
	// Independent targets and deferred data are fixture-only setup, not observed SQL.
	_, err = pool.Exec(ctx, `UPDATE public.variant_target_set SET target_weight=50.25,target_reps=8 WHERE id=$1;`, *a.Sets[1].TargetSetID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE public.workout_set SET target_weight=50.25,target_reps=8 WHERE id=$1`, a.Sets[1].ID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE public.exercise_variant SET progression_mode='off',last_rpe=9,last_rpe_at=now() WHERE id=$1`, a.VariantID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE public.workout_exercise SET progression_mode='off',notes='Keep me',recorded_rpe=7 WHERE id=$1`, a.ID); err != nil {
		t.Fatal(err)
	}
	p := request(t, h, "POST", "/api/training/workouts/sets/record", `{"operationId":"a3000000-0000-4000-8000-000000000001","workoutId":"`+w.ID+`","attemptId":"`+a.ID+`","setId":"`+a.Sets[0].ID+`","expectedWorkoutRevision":"1","state":"performed","recordedReps":0}`, "https://app.example", nil)
	var recorded training.WorkoutResult
	if err := json.Unmarshal(p.Body.Bytes(), &recorded); p.Code != 200 || err != nil {
		t.Fatal(p.Body.String())
	}
	first := recorded.Workout.Exercises[0].Sets[0]
	grown := configurationSaved(t, h, configurationIntent(recorded.Workout, 2, map[string]any{"prescribedSets": 3, "repThreshold": 10, "autoIncrement": "1.25", "restSeconds": 90}))
	now := grown.Workout.Exercises[0]
	if len(now.Sets) != 3 || len(now.CurrentVariant.TargetSets) != 3 || now.Sets[0].ID != a.Sets[0].ID || now.Sets[1].ID != a.Sets[1].ID || now.Sets[1].TargetWeight != "50.25" || now.Sets[1].TargetReps != 8 || now.Sets[2].TargetWeight != "50.25" || now.Sets[2].TargetReps != 10 || now.Sets[2].State != "pending" {
		t.Fatalf("growth flattened or reset retained work: %+v", now)
	}
	if len(now.Exercise.MuscleGroupCodes) != 1 || now.Exercise.MuscleGroupCodes[0] != "chest" || *now.Sets[0].RecordedReps != 0 || *now.Sets[0].RecordedWeight != "40.50" || !now.Sets[0].RecordedAt.Equal(*first.RecordedAt) || now.Sets[0].LateWeightOverride || now.ProgressionMode != "off" || now.CurrentVariant.ProgressionMode != "off" || *now.Notes != "Keep me" || *now.RecordedRPE != 7 || now.AutoIncrement != "1.25" || now.RestSeconds != 90 {
		t.Fatalf("growth lost actual/deferred settings: %+v", now)
	}
	equalized := configurationSaved(t, h, configurationIntent(grown.Workout, 3, map[string]any{"applyWeightToAllSets": "45.00"}))
	now = equalized.Workout.Exercises[0]
	for _, s := range now.Sets {
		if s.TargetWeight != "45.00" {
			t.Fatal("explicit equalization failed")
		}
	}
	if !now.Sets[0].LateWeightOverride || now.Sets[1].LateWeightOverride || *now.Sets[0].RecordedWeight != "40.50" || !now.Sets[0].RecordedAt.Equal(*first.RecordedAt) {
		t.Fatal("late override incorrectly changed actuals or marked pending rows")
	}
	shrunk := configurationSaved(t, h, configurationIntent(equalized.Workout, 4, map[string]any{"prescribedSets": 1, "variantName": "Work"}))
	now = shrunk.Workout.Exercises[0]
	if len(now.Sets) != 1 || len(now.CurrentVariant.TargetSets) != 1 || now.Sets[0].ID != first.ID || *now.Sets[0].TargetSetID != *first.TargetSetID || !now.Sets[0].LateWeightOverride || *now.Sets[0].RecordedReps != 0 {
		t.Fatalf("shrink lost retained identity/marker: %+v", now)
	}
}

func TestSharedConfigurationShrinkDiscardsDraftWorkButRetainsCompletedProvenanceAndSharedTargets(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if err != nil {
		t.Fatal(err)
	}
	old := configuredWorkout(t, h)
	// Completed history is fixture setup until #33. Include actual zero, notes,
	// settings and original target provenance, then begin a fresh draft publicly.
	if _, err = pool.Exec(ctx, `UPDATE public.workout_set SET state='performed',recorded_reps=0,recorded_weight=target_weight,recorded_at=now() WHERE workout_exercise_id=$1`, old.Exercises[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE public.workout_exercise SET state='completed',completed_at=now(),notes='History',recorded_rpe=8 WHERE id=$1`, old.Exercises[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE public.workout SET status='ended',ended_at=now() WHERE id=$1`, old.ID); err != nil {
		t.Fatal(err)
	}
	p := request(t, h, "GET", "/api/training/workouts/"+old.ID, "", "", nil)
	var history struct{ Workout training.Workout }
	if err := json.Unmarshal(p.Body.Bytes(), &history); err != nil {
		t.Fatal(err)
	}
	p = request(t, h, "POST", "/api/training/workouts/start", `{"operationId":"a4000000-0000-4000-8000-000000000001","sessionId":"`+old.SourceSession.ID+`","expectedSessionRevision":"1"}`, "https://app.example", nil)
	var fresh training.WorkoutResult
	if err := json.Unmarshal(p.Body.Bytes(), &fresh); p.Code != 200 || err != nil {
		t.Fatal(p.Body.String())
	}
	w := fresh.Workout
	// A second session uses the same shared variant, with no copied settings.
	a := w.Exercises[0]
	p = request(t, h, "POST", "/api/training/sessions/create", `{"operationId":"a4000000-0000-4000-8000-000000000002","name":"Also press","items":[{"kind":"reuseVariant","exerciseId":"`+a.Exercise.ID+`","variantId":"`+a.VariantID+`","expectedVariantRevision":"1"}]}`, "https://app.example", nil)
	var shared training.SessionResult
	if err := json.Unmarshal(p.Body.Bytes(), &shared); p.Code != 200 || err != nil {
		t.Fatal(p.Body.String())
	}
	p = request(t, h, "POST", "/api/training/workouts/sets/record", `{"operationId":"a4000000-0000-4000-8000-000000000003","workoutId":"`+w.ID+`","attemptId":"`+a.ID+`","setId":"`+a.Sets[1].ID+`","expectedWorkoutRevision":"1","state":"performed","recordedReps":11}`, "https://app.example", nil)
	var recorded training.WorkoutResult
	if err := json.Unmarshal(p.Body.Bytes(), &recorded); p.Code != 200 || err != nil {
		t.Fatal(p.Body.String())
	}
	saved := configurationSaved(t, h, configurationIntent(recorded.Workout, 5, map[string]any{"prescribedSets": 1, "exerciseName": "New name", "applyWeightToAllSets": "42.00"}))
	if len(saved.Workout.Exercises[0].Sets) != 1 {
		t.Fatal("removed draft work retained")
	}
	p = request(t, h, "GET", "/api/training/workouts/"+old.ID, "", "", nil)
	var after struct{ Workout training.Workout }
	if err := json.Unmarshal(p.Body.Bytes(), &after); p.Code != 200 || err != nil {
		t.Fatal(p.Body.String())
	}
	before, now := history.Workout.Exercises[0], after.Workout.Exercises[0]
	if len(now.Sets) != 2 || now.Sets[1].TargetSetID != nil || now.Sets[1].Kind != "prescribed" || now.Sets[1].ID != before.Sets[1].ID || *now.Sets[1].RecordedReps != 0 || now.Sets[1].TargetWeight != "40.50" || *now.Sets[1].RecordedWeight != "40.50" || !now.Sets[1].RecordedAt.Equal(*before.Sets[1].RecordedAt) || !now.CompletedAt.Equal(*before.CompletedAt) || now.RepThreshold != 12 || now.ConfigRevision != "1" || *now.Notes != "History" || *now.RecordedRPE != 8 || now.Exercise.Name != "New name" {
		t.Fatalf("history reconciled: %+v", now)
	}
	p = request(t, h, "GET", "/api/training/sessions/"+shared.Session.ID, "", "", nil)
	var current training.SessionResult
	if err := json.Unmarshal(p.Body.Bytes(), &current); p.Code != 200 || err != nil {
		t.Fatal(p.Body.String())
	}
	if current.Session.Placements[0].Variant.PrescribedSets != 1 || current.Session.Placements[0].Variant.TargetSets[0].TargetWeight != "42.00" || current.Session.Placements[0].Exercise.Name != "New name" {
		t.Fatal("another session kept old shared settings")
	}
	// Abandonment cannot undo this independent configuration edit (quit is #34).
	if _, err = pool.Exec(ctx, `DELETE FROM public.workout_exercise WHERE workout_id=$1`, w.ID); err != nil {
		t.Fatal(err)
	}
	p = request(t, h, "GET", "/api/training/sessions/"+shared.Session.ID, "", "", nil)
	if !strings.Contains(p.Body.String(), `"targetWeight":"42.00"`) {
		t.Fatal("configuration depended on draft persistence")
	}
}

func TestSharedConfigurationGuardsNamesRevisionsOwnershipReplayAndRollback(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if err != nil {
		t.Fatal(err)
	}
	w := configuredWorkout(t, h)
	a := w.Exercises[0]
	// A reserved archived sibling name remains a conflict.
	fixtureTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer fixtureTx.Rollback(ctx)
	if _, err = fixtureTx.Exec(ctx, `INSERT INTO public.exercise_variant(id,owner_mail,exercise_id,name,progression_mode,rep_threshold,auto_increment,rest_seconds,archived_at) SELECT 'a5000000-0000-4000-8000-000000000001',owner_mail,exercise_id,'Reserved','all_sets',12,2.5,0,now() FROM public.exercise_variant WHERE id=$1`, a.VariantID); err != nil {
		t.Fatal(err)
	}
	if _, err = fixtureTx.Exec(ctx, `INSERT INTO public.variant_target_set(id,owner_mail,variant_id,position,target_reps,target_weight) SELECT 'a5000000-0000-4000-8000-000000000002',owner_mail,'a5000000-0000-4000-8000-000000000001',1,12,40 FROM public.exercise_variant WHERE id=$1`, a.VariantID); err != nil {
		t.Fatal(err)
	}
	if err = fixtureTx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		field string
		value any
		want  int
		code  string
	}{
		{"expectedWorkoutRevision", "2", 409, "stale-revision"}, {"expectedExerciseRevision", "2", 409, "stale-revision"}, {"expectedVariantRevision", "2", 409, "stale-revision"},
		{"exerciseId", w.Exercises[1].Exercise.ID, 404, "not-found"}, {"variantId", w.Exercises[1].VariantID, 404, "not-found"}, {"attemptId", w.Exercises[1].ID, 404, "not-found"},
		{"patch", map[string]any{"variantName": " reserved "}, 409, "name-conflict"}, {"patch", map[string]any{"exerciseName": w.Exercises[1].Exercise.Name}, 409, "name-conflict"},
		{"patch", map[string]any{"initialWeight": "10"}, 400, "malformed-request"}, {"patch", map[string]any{"progressionMode": "off"}, 400, "malformed-request"}, {"patch", map[string]any{"notes": "changed"}, 400, "malformed-request"},
		{"patch", map[string]any{"applyWeightToAllSets": "10.001"}, 422, "invalid-field"}, {"patch", map[string]any{"prescribedSets": 0}, 422, "invalid-field"}, {"patch", map[string]any{"repThreshold": nil}, 422, "invalid-field"},
	} {
		c := configurationIntent(w, 6, map[string]any{"exerciseName": "Safe"})
		c[tc.field] = tc.value
		status, body := sendConfiguration(t, h, c)
		if status != tc.want || !strings.Contains(body, tc.code) {
			t.Fatalf("%s=%v: %d %s", tc.field, tc.value, status, body)
		}
	}
	if _, err = pool.Exec(ctx, `INSERT INTO public."User"(mail,name) VALUES('other-config@example.com','Other')`); err != nil {
		t.Fatal(err)
	}
	other, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{Model: application.ExerciseCentric}, Resolve: func(*http.Request) (application.Identity, error) {
		return application.Identity{Email: "other-config@example.com"}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	foreignStatus, foreignBody := sendConfiguration(t, other, configurationIntent(w, 7, map[string]any{"exerciseName": "Cross owner"}))
	if foreignStatus != 404 || !strings.Contains(foreignBody, "not-found") {
		t.Fatalf("ownership: %d %s", foreignStatus, foreignBody)
	}

	c := configurationIntent(w, 7, map[string]any{"exerciseName": "Saved", "prescribedSets": 1})
	if _, err = pool.Exec(ctx, `CREATE FUNCTION public.fail_config_revision() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test config failure'; END $$; CREATE TRIGGER fail_config_revision BEFORE UPDATE ON public.workout FOR EACH ROW EXECUTE FUNCTION public.fail_config_revision()`); err != nil {
		t.Fatal(err)
	}
	status, body := sendConfiguration(t, h, c)
	if status != 500 || strings.Contains(body, "test config failure") {
		t.Fatalf("rollback: %d %s", status, body)
	}
	p := request(t, h, "GET", "/api/training/workouts/"+w.ID, "", "", nil)
	var unchanged struct{ Workout training.Workout }
	if err := json.Unmarshal(p.Body.Bytes(), &unchanged); p.Code != 200 || err != nil {
		t.Fatal(p.Body.String())
	}
	if unchanged.Workout.Revision != "1" || unchanged.Workout.Exercises[0].Exercise.Name != a.Exercise.Name || len(unchanged.Workout.Exercises[0].Sets) != 2 || unchanged.Workout.Exercises[0].CurrentVariant.Revision != "1" {
		t.Fatal("failed transaction partially committed")
	}
	if _, err = pool.Exec(ctx, `DROP TRIGGER fail_config_revision ON public.workout; DROP FUNCTION public.fail_config_revision()`); err != nil {
		t.Fatal(err)
	}
	saved := configurationSaved(t, h, c)
	c["patch"] = map[string]any{"exerciseName": "Different", "prescribedSets": 1}
	status, body = sendConfiguration(t, h, c)
	if status != 409 || !strings.Contains(body, "operation-mismatch") {
		t.Fatal(body)
	}
	c["patch"] = map[string]any{"exerciseName": "Saved", "prescribedSets": 1}
	if _, err = pool.Exec(ctx, `UPDATE public.workout SET status='ended',ended_at=now(),revision=3 WHERE id=$1`, w.ID); err != nil {
		t.Fatal(err)
	}
	replay := configurationSaved(t, h, c)
	if replay.Operation != saved.Operation || replay.Workout.Revision != "3" || replay.Workout.Status != "ended" {
		t.Fatal("receipt was not resolved before finality")
	}
	c["operationId"] = "a2000000-0000-4000-8000-000000000008"
	c["expectedWorkoutRevision"] = "3"
	c["expectedExerciseRevision"] = "2"
	c["expectedVariantRevision"] = "2"
	status, body = sendConfiguration(t, h, c)
	if status != 409 || !strings.Contains(body, "resource-finalized") {
		t.Fatalf("late mutation: %d %s", status, body)
	}
}

func TestSharedConfigurationSerializesRecordingAndIdenticalRetries(t *testing.T) {
	for _, sameIntent := range []bool{true, false} {
		t.Run(fmt.Sprintf("same intent %v", sameIntent), func(t *testing.T) {
			pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
			if err := migrations.Apply(ctx, pool); err != nil {
				t.Fatal(err)
			}
			h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
			if err != nil {
				t.Fatal(err)
			}
			w := configuredWorkout(t, h)
			a := w.Exercises[0]
			b, _ := json.Marshal(configurationIntent(w, 10, map[string]any{"prescribedSets": 3, "applyWeightToAllSets": "45.50"}))
			bodies := []string{string(b), string(b)}
			paths := []string{"/api/training/workouts/configuration", "/api/training/workouts/configuration"}
			if !sameIntent {
				paths[1] = "/api/training/workouts/sets/record"
				bodies[1] = `{"operationId":"a6000000-0000-4000-8000-000000000001","workoutId":"` + w.ID + `","attemptId":"` + a.ID + `","setId":"` + a.Sets[0].ID + `","expectedWorkoutRevision":"1","state":"performed","recordedReps":0}`
			}
			var replies [2]*httptest.ResponseRecorder
			var wg sync.WaitGroup
			for i := range replies {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					replies[i] = request(t, h, "POST", paths[i], bodies[i], "https://app.example", nil)
				}(i)
			}
			wg.Wait()
			accepted, stale := 0, 0
			var ack training.Acknowledgment
			for _, p := range replies {
				switch p.Code {
				case 200:
					var out training.WorkoutResult
					if err := json.Unmarshal(p.Body.Bytes(), &out); err != nil || out.Workout.Revision != "2" {
						t.Fatal(p.Body.String())
					}
					if accepted > 0 && ack != out.Operation {
						t.Fatal("identical retry acknowledged another mutation")
					}
					ack = out.Operation
					accepted++
				case 409:
					if !strings.Contains(p.Body.String(), "stale-revision") {
						t.Fatal(p.Body.String())
					}
					stale++
				default:
					t.Fatalf("concurrent command: %d %s", p.Code, p.Body.String())
				}
			}
			if sameIntent && (accepted != 2 || stale != 0) || !sameIntent && (accepted != 1 || stale != 1) {
				t.Fatalf("accepted %d stale %d", accepted, stale)
			}
		})
	}
}
