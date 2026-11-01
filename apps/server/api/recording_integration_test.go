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

func TestExplicitSetRecordingRestoresActualZeroAndStableIdentity(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if err != nil {
		t.Fatal(err)
	}
	p := request(t, h, "POST", "/api/training/sessions/create", manualPlan, "https://app.example", nil)
	var plan training.SessionResult
	if err := json.Unmarshal(p.Body.Bytes(), &plan); p.Code != 200 || err != nil {
		t.Fatal(p.Body.String())
	}
	p = request(t, h, "POST", "/api/training/workouts/start", `{"operationId":"91000000-0000-4000-8000-000000000001","sessionId":"`+plan.Session.ID+`","expectedSessionRevision":"1"}`, "https://app.example", nil)
	var started training.WorkoutResult
	if err := json.Unmarshal(p.Body.Bytes(), &started); p.Code != 200 || err != nil {
		t.Fatal(p.Body.String())
	}
	w := started.Workout
	a := w.Exercises[0]
	command := `{"operationId":"91000000-0000-4000-8000-000000000002","workoutId":"` + w.ID + `","attemptId":"` + a.ID + `","setId":"` + a.Sets[0].ID + `","expectedWorkoutRevision":"1","state":"performed","recordedReps":0}`
	p = request(t, h, "POST", "/api/training/workouts/sets/record", command, "https://app.example", nil)
	var saved training.WorkoutResult
	if err := json.Unmarshal(p.Body.Bytes(), &saved); p.Code != http.StatusOK || err != nil {
		t.Fatalf("record: %d %s", p.Code, p.Body.String())
	}
	set := saved.Workout.Exercises[0].Sets[0]
	if saved.Workout.Revision != "2" || saved.Operation.AcceptedRevision != "2" || set.ID != a.Sets[0].ID || set.State != "performed" || set.RecordedReps == nil || *set.RecordedReps != 0 || set.RecordedWeight == nil || *set.RecordedWeight != "40.50" || set.RecordedAt == nil {
		t.Fatalf("actual zero: %s", p.Body.String())
	}
	if saved.Workout.Exercises[0].Sets[1].State != "pending" || saved.Workout.Exercises[1].State != "draft" {
		t.Fatal("recording affected other sets or completed attempt")
	}
	for _, path := range []string{"/api/training/workouts/" + w.ID, "/api/training/workouts/active"} {
		p = request(t, h, "GET", path, "", "", nil)
		var resumed struct{ Workout training.Workout }
		if err := json.Unmarshal(p.Body.Bytes(), &resumed); p.Code != 200 || err != nil || resumed.Workout.Exercises[0].Sets[0].RecordedReps == nil || *resumed.Workout.Exercises[0].Sets[0].RecordedReps != 0 {
			t.Fatalf("restore: %d %s", p.Code, p.Body.String())
		}
	}
}

func TestSetCorrectionResetSkipAndRetryPreserveExactRecordedPerformance(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if err != nil {
		t.Fatal(err)
	}
	p := request(t, h, "POST", "/api/training/sessions/create", manualPlan, "https://app.example", nil)
	var plan training.SessionResult
	if err := json.Unmarshal(p.Body.Bytes(), &plan); p.Code != 200 || err != nil {
		t.Fatal(p.Body.String())
	}
	// Establish unequal applicable workout targets before any recording. Public
	// configuration reconciliation is delivered in the following ticket.
	if _, err := pool.Exec(ctx, `UPDATE public.variant_target_set SET target_weight=50.25 WHERE id=$1`, plan.Session.Placements[0].Variant.TargetSets[1].ID); err != nil {
		t.Fatal(err)
	}
	p = request(t, h, "POST", "/api/training/workouts/start", `{"operationId":"92000000-0000-4000-8000-000000000001","sessionId":"`+plan.Session.ID+`","expectedSessionRevision":"1"}`, "https://app.example", nil)
	var start training.WorkoutResult
	if err := json.Unmarshal(p.Body.Bytes(), &start); p.Code != 200 || err != nil {
		t.Fatal(p.Body.String())
	}
	w := start.Workout
	command := func(op, revision, setID, state string, reps any) string {
		c := map[string]any{"operationId": op, "workoutId": w.ID, "attemptId": w.Exercises[0].ID, "setId": setID, "expectedWorkoutRevision": revision, "state": state}
		if reps != nil {
			c["recordedReps"] = reps
		}
		body, _ := json.Marshal(c)
		return string(body)
	}
	send := func(body string) training.WorkoutResult {
		t.Helper()
		p := request(t, h, "POST", "/api/training/workouts/sets/record", body, "https://app.example", nil)
		var out training.WorkoutResult
		if err := json.Unmarshal(p.Body.Bytes(), &out); p.Code != 200 || err != nil {
			t.Fatalf("record: %d %s", p.Code, p.Body.String())
		}
		return out
	}
	firstBody := command("92000000-0000-4000-8000-000000000002", "1", w.Exercises[0].Sets[0].ID, "performed", 15)
	first := send(firstBody)
	firstSet := first.Workout.Exercises[0].Sets[0]
	second := send(command("92000000-0000-4000-8000-000000000003", "2", w.Exercises[0].Sets[1].ID, "performed", 8))
	if *second.Workout.Exercises[0].Sets[1].RecordedWeight != "50.25" || *second.Workout.Exercises[0].Sets[0].RecordedWeight != "40.50" {
		t.Fatal("individual weights were equalized")
	}
	replay := send(firstBody)
	if replay.Operation != first.Operation || replay.Workout.Revision != "3" || *replay.Workout.Exercises[0].Sets[1].RecordedReps != 8 {
		t.Fatal("lost-response replay lost acknowledgment/current state")
	}
	// Simulate the later shared-config operation's retained-set reconciliation.
	if _, err := pool.Exec(ctx, `UPDATE public.workout_set SET target_weight=45,late_weight_override=true WHERE id=$1`, firstSet.ID); err != nil {
		t.Fatal(err)
	}
	corrected := send(command("92000000-0000-4000-8000-000000000004", "3", firstSet.ID, "performed", 100))
	set := corrected.Workout.Exercises[0].Sets[0]
	if *set.RecordedReps != 100 || *set.RecordedWeight != "40.50" || !set.RecordedAt.Equal(*firstSet.RecordedAt) || set.TargetWeight != "45.00" || !set.LateWeightOverride {
		t.Fatalf("correction rewrote captured performance: %+v", set)
	}
	skipped := send(command("92000000-0000-4000-8000-000000000005", "4", set.ID, "skipped", nil))
	set = skipped.Workout.Exercises[0].Sets[0]
	if set.State != "skipped" || set.RecordedReps != nil || set.RecordedWeight != nil || set.RecordedAt != nil || set.TargetWeight != "45.00" {
		t.Fatalf("skip: %+v", set)
	}
	reset := send(command("92000000-0000-4000-8000-000000000006", "5", set.ID, "pending", nil))
	set = reset.Workout.Exercises[0].Sets[0]
	if set.State != "pending" || set.RecordedReps != nil || set.RecordedWeight != nil || set.RecordedAt != nil {
		t.Fatalf("reset: %+v", set)
	}
	reperformed := send(command("92000000-0000-4000-8000-000000000007", "6", set.ID, "performed", 0))
	set = reperformed.Workout.Exercises[0].Sets[0]
	if set.State != "performed" || *set.RecordedReps != 0 || *set.RecordedWeight != "45.00" || !set.RecordedAt.After(*firstSet.RecordedAt) || set.LateWeightOverride {
		t.Fatalf("explicit fresh perform: %+v", set)
	}
	p = request(t, h, "GET", "/api/training/sessions/"+plan.Session.ID, "", "", nil)
	var current training.SessionResult
	if err := json.Unmarshal(p.Body.Bytes(), &current); p.Code != 200 || err != nil {
		t.Fatal(p.Body.String())
	}
	if current.Session.Placements[0].Variant.Revision != "1" || current.Session.Placements[0].Variant.TargetSets[0].TargetReps != 12 || current.Session.Placements[0].Variant.TargetSets[0].TargetWeight != "40.50" {
		t.Fatal("recording progressed or overwrote future targets")
	}
}

func TestSetCommandsRejectStaleMixedIdentityAndChangedReplayWithoutOverwritingWork(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO public."User"(mail,name) VALUES('owner@example.com','Owner'),('other@example.com','Other')`); err != nil {
		t.Fatal(err)
	}
	handler := func(email string) http.Handler {
		t.Helper()
		h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{Model: application.ExerciseCentric}, Resolve: func(*http.Request) (application.Identity, error) { return application.Identity{Email: email}, nil }})
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	h, other := handler("owner@example.com"), handler("other@example.com")
	start := func(h http.Handler) training.Workout {
		t.Helper()
		p := request(t, h, "POST", "/api/training/sessions/create", manualPlan, "https://app.example", nil)
		var plan training.SessionResult
		if err := json.Unmarshal(p.Body.Bytes(), &plan); p.Code != 200 || err != nil {
			t.Fatal(p.Body.String())
		}
		p = request(t, h, "POST", "/api/training/workouts/start", `{"operationId":"93000000-0000-4000-8000-000000000001","sessionId":"`+plan.Session.ID+`","expectedSessionRevision":"1"}`, "https://app.example", nil)
		var started training.WorkoutResult
		if err := json.Unmarshal(p.Body.Bytes(), &started); p.Code != 200 || err != nil {
			t.Fatal(p.Body.String())
		}
		return started.Workout
	}
	w, foreign := start(h), start(other)
	c := map[string]any{"operationId": "93000000-0000-4000-8000-000000000002", "workoutId": w.ID, "attemptId": w.Exercises[0].ID, "setId": w.Exercises[0].Sets[0].ID, "expectedWorkoutRevision": "1", "state": "performed", "recordedReps": 7}
	send := func(h http.Handler, c map[string]any) (int, string) {
		t.Helper()
		body, _ := json.Marshal(c)
		p := request(t, h, "POST", "/api/training/workouts/sets/record", string(body), "https://app.example", nil)
		return p.Code, p.Body.String()
	}
	status, body := send(h, c)
	var accepted training.WorkoutResult
	if err := json.Unmarshal([]byte(body), &accepted); status != 200 || err != nil {
		t.Fatalf("initial: %d %s", status, body)
	}
	changed := func(fields map[string]any) map[string]any {
		copy := map[string]any{}
		for k, v := range c {
			copy[k] = v
		}
		for k, v := range fields {
			copy[k] = v
		}
		return copy
	}
	cases := []struct {
		name   string
		h      http.Handler
		fields map[string]any
		status int
		code   string
	}{
		{"changed reps under accepted operation", h, map[string]any{"recordedReps": 8}, 409, "operation-mismatch"},
		{"changed set under accepted operation", h, map[string]any{"setId": w.Exercises[0].Sets[1].ID}, 409, "operation-mismatch"},
		{"stale device", h, map[string]any{"operationId": "93000000-0000-4000-8000-000000000003", "recordedReps": 18}, 409, "stale-revision"},
		{"cross owner", other, map[string]any{}, 404, "not-found"},
		{"foreign workout", h, map[string]any{"operationId": "93000000-0000-4000-8000-000000000004", "workoutId": foreign.ID}, 404, "not-found"},
		{"foreign attempt", h, map[string]any{"operationId": "93000000-0000-4000-8000-000000000004", "attemptId": foreign.Exercises[0].ID}, 404, "not-found"},
		{"wrong attempt parent", h, map[string]any{"operationId": "93000000-0000-4000-8000-000000000004", "attemptId": w.Exercises[1].ID}, 404, "not-found"},
		{"wrong set parent", h, map[string]any{"operationId": "93000000-0000-4000-8000-000000000004", "setId": w.Exercises[1].Sets[0].ID}, 404, "not-found"},
		{"foreign set", h, map[string]any{"operationId": "93000000-0000-4000-8000-000000000004", "setId": foreign.Exercises[0].Sets[0].ID}, 404, "not-found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := send(tc.h, changed(tc.fields))
			if status != tc.status || !strings.Contains(body, tc.code) {
				t.Fatalf("guard: %d %s", status, body)
			}
		})
	}
	// Finality is fixture setup until #33 exposes completion. Exact receipt replay
	// must still work, while a new late draft command is rejected.
	if _, err := pool.Exec(ctx, `UPDATE public.workout_exercise SET state='completed',completed_at=now() WHERE id=$1`, w.Exercises[0].ID); err != nil {
		t.Fatal(err)
	}
	status, body = send(h, changed(map[string]any{"operationId": "93000000-0000-4000-8000-000000000005", "expectedWorkoutRevision": "2"}))
	if status != 409 || !strings.Contains(body, "resource-finalized") {
		t.Fatalf("finality: %d %s", status, body)
	}
	if _, err := pool.Exec(ctx, `UPDATE public.workout SET status='ended',ended_at=now(),revision=3 WHERE id=$1`, w.ID); err != nil {
		t.Fatal(err)
	}
	status, body = send(h, c)
	var replay training.WorkoutResult
	if err := json.Unmarshal([]byte(body), &replay); status != 200 || err != nil || replay.Operation != accepted.Operation || replay.Workout.Status != "ended" || replay.Workout.Revision != "3" || *replay.Workout.Exercises[0].Sets[0].RecordedReps != 7 {
		t.Fatalf("receipt before stale/final: %d %s", status, body)
	}
}

func TestConcurrentSetCommandsAndFailedTransactionRemainRetrySafe(t *testing.T) {
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
			p := request(t, h, "POST", "/api/training/sessions/create", manualPlan, "https://app.example", nil)
			var plan training.SessionResult
			if err := json.Unmarshal(p.Body.Bytes(), &plan); p.Code != 200 || err != nil {
				t.Fatal(p.Body.String())
			}
			p = request(t, h, "POST", "/api/training/workouts/start", `{"operationId":"94000000-0000-4000-8000-000000000001","sessionId":"`+plan.Session.ID+`","expectedSessionRevision":"1"}`, "https://app.example", nil)
			var start training.WorkoutResult
			if err := json.Unmarshal(p.Body.Bytes(), &start); p.Code != 200 || err != nil {
				t.Fatal(p.Body.String())
			}
			w := start.Workout
			body := `{"operationId":"94000000-0000-4000-8000-000000000002","workoutId":"` + w.ID + `","attemptId":"` + w.Exercises[0].ID + `","setId":"` + w.Exercises[0].Sets[0].ID + `","expectedWorkoutRevision":"1","state":"performed","recordedReps":6}`
			// Fail after the set update, before revision/receipt can commit. Observe the
			// rollback and recovery solely through the public mutation/read boundaries.
			if _, err := pool.Exec(ctx, `CREATE FUNCTION public.fail_record_revision() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test storage failure'; END $$; CREATE TRIGGER fail_record_revision BEFORE UPDATE ON public.workout FOR EACH ROW EXECUTE FUNCTION public.fail_record_revision()`); err != nil {
				t.Fatal(err)
			}
			p = request(t, h, "POST", "/api/training/workouts/sets/record", body, "https://app.example", nil)
			if p.Code != 500 || strings.Contains(p.Body.String(), "test storage failure") {
				t.Fatalf("storage failure: %d %s", p.Code, p.Body.String())
			}
			p = request(t, h, "GET", "/api/training/workouts/"+w.ID, "", "", nil)
			var unchanged struct{ Workout training.Workout }
			if err := json.Unmarshal(p.Body.Bytes(), &unchanged); p.Code != 200 || err != nil || unchanged.Workout.Revision != "1" || unchanged.Workout.Exercises[0].Sets[0].State != "pending" {
				t.Fatalf("partial failed write: %s", p.Body.String())
			}
			if _, err := pool.Exec(ctx, `DROP TRIGGER fail_record_revision ON public.workout; DROP FUNCTION public.fail_record_revision()`); err != nil {
				t.Fatal(err)
			}
			bodies := []string{body, body}
			if !sameIntent {
				bodies[1] = strings.NewReplacer("94000000-0000-4000-8000-000000000002", "94000000-0000-4000-8000-000000000003", `"recordedReps":6`, `"recordedReps":9`).Replace(body)
			}
			var replies [2]*httptest.ResponseRecorder
			var wg sync.WaitGroup
			for i := range replies {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					replies[i] = request(t, h, "POST", "/api/training/workouts/sets/record", bodies[i], "https://app.example", nil)
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
						t.Fatalf("concurrent save: %s", p.Body.String())
					}
					if accepted > 0 && ack != out.Operation {
						t.Fatal("identical concurrent retry got different acknowledgment")
					}
					ack = out.Operation
					accepted++
				case 409:
					if !strings.Contains(p.Body.String(), "stale-revision") {
						t.Fatal(p.Body.String())
					}
					stale++
				default:
					t.Fatalf("concurrent: %d %s", p.Code, p.Body.String())
				}
			}
			if sameIntent && (accepted != 2 || stale != 0) || !sameIntent && (accepted != 1 || stale != 1) {
				t.Fatalf("accepted=%d stale=%d", accepted, stale)
			}
		})
	}
}

func TestSetCommandValidationAndPayloadPresence(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if err != nil {
		t.Fatal(err)
	}
	p := request(t, h, "POST", "/api/training/sessions/create", manualPlan, "https://app.example", nil)
	var plan training.SessionResult
	if err := json.Unmarshal(p.Body.Bytes(), &plan); p.Code != 200 || err != nil {
		t.Fatal(p.Body.String())
	}
	p = request(t, h, "POST", "/api/training/workouts/start", `{"operationId":"95000000-0000-4000-8000-000000000001","sessionId":"`+plan.Session.ID+`","expectedSessionRevision":"1"}`, "https://app.example", nil)
	var start training.WorkoutResult
	if err := json.Unmarshal(p.Body.Bytes(), &start); p.Code != 200 || err != nil {
		t.Fatal(p.Body.String())
	}
	w := start.Workout
	base := map[string]any{"operationId": "95000000-0000-4000-8000-000000000002", "workoutId": w.ID, "attemptId": w.Exercises[0].ID, "setId": w.Exercises[0].Sets[0].ID, "expectedWorkoutRevision": "1", "state": "performed", "recordedReps": 7}
	for _, tc := range []struct {
		name  string
		field string
		value any
		want  int
	}{
		{"negative reps", "recordedReps", -1, 422}, {"storage overflow", "recordedReps", 2147483648, 422}, {"null reps", "recordedReps", nil, 422}, {"fractional reps", "recordedReps", 1.5, 400}, {"text reps", "recordedReps", "7", 400}, {"unknown state", "state", "complete", 422}, {"skip with actuals", "state", "skipped", 422}, {"pending with actuals", "state", "pending", 422}, {"weight override", "recordedWeight", "50", 400}, {"deferred RPE", "recordedRpe", 5, 400}, {"deferred notes", "notes", "test", 400}, {"owner override", "owner", "other@example.com", 400}, {"numeric revision", "expectedWorkoutRevision", 1, 400}, {"noncanonical revision", "expectedWorkoutRevision", "01", 422}, {"invalid nested ID", "setId", "not-a-uuid", 422},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := map[string]any{}
			for k, v := range base {
				c[k] = v
			}
			c[tc.field] = tc.value
			body, _ := json.Marshal(c)
			p := request(t, h, "POST", "/api/training/workouts/sets/record", string(body), "https://app.example", nil)
			if p.Code != tc.want {
				t.Fatalf("validation: %d %s", p.Code, p.Body.String())
			}
		})
	}
	for _, body := range []string{"null", "[]", "{} {}"} {
		p := request(t, h, "POST", "/api/training/workouts/sets/record", body, "https://app.example", nil)
		if p.Code != 400 {
			t.Fatalf("shape: %d %s", p.Code, p.Body.String())
		}
	}
	delete(base, "recordedReps")
	base["state"] = "pending"
	body, _ := json.Marshal(base)
	p = request(t, h, "POST", "/api/training/workouts/sets/record", string(body), "https://app.example", nil)
	if p.Code != 200 {
		t.Fatalf("pending without actuals: %d %s", p.Code, p.Body.String())
	}
	base["recordedReps"] = nil
	body, _ = json.Marshal(base)
	p = request(t, h, "POST", "/api/training/workouts/sets/record", string(body), "https://app.example", nil)
	if p.Code != 409 || !strings.Contains(p.Body.String(), "operation-mismatch") {
		t.Fatalf("changed field presence: %d %s", p.Code, p.Body.String())
	}
}
