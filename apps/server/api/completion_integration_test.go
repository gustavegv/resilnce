//go:build integration

package api

import (
	"encoding/json"
	"fmt"
	"github.com/gustavegv/resilnce/apps/server/application"
	"github.com/gustavegv/resilnce/apps/server/internal/testdb"
	"github.com/gustavegv/resilnce/apps/server/migrations"
	"github.com/gustavegv/resilnce/apps/server/training"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func completionIntent(w training.Workout, index, op int) map[string]any {
	a := w.Exercises[index]
	return map[string]any{"operationId": fmt.Sprintf("a3300000-0000-4000-8000-%012d", op), "workoutId": w.ID, "attemptId": a.ID, "exerciseId": a.Exercise.ID, "variantId": a.VariantID, "expectedWorkoutRevision": w.Revision, "expectedVariantRevision": a.CurrentVariant.Revision}
}
func sendCompletion(t *testing.T, h http.Handler, c map[string]any) (int, string) {
	t.Helper()
	raw, _ := json.Marshal(c)
	p := request(t, h, "POST", "/api/training/workouts/complete", string(raw), "https://app.example", nil)
	return p.Code, p.Body.String()
}
func completed(t *testing.T, h http.Handler, c map[string]any) training.WorkoutResult {
	t.Helper()
	status, body := sendCompletion(t, h, c)
	var result training.WorkoutResult
	if e := json.Unmarshal([]byte(body), &result); status != 200 || e != nil {
		t.Fatalf("complete: %d %s", status, body)
	}
	return result
}
func completionRecord(t *testing.T, h http.Handler, w training.Workout, ai, si, op int, state string, reps int) training.Workout {
	t.Helper()
	a := w.Exercises[ai]
	c := map[string]any{"operationId": fmt.Sprintf("a3310000-0000-4000-8000-%012d", op), "workoutId": w.ID, "attemptId": a.ID, "setId": a.Sets[si].ID, "expectedWorkoutRevision": w.Revision, "state": state}
	if state == "performed" {
		c["recordedReps"] = reps
	}
	raw, _ := json.Marshal(c)
	p := request(t, h, "POST", "/api/training/workouts/sets/record", string(raw), "https://app.example", nil)
	var result training.WorkoutResult
	if e := json.Unmarshal(p.Body.Bytes(), &result); p.Code != 200 || e != nil {
		t.Fatalf("record: %d %s", p.Code, p.Body.String())
	}
	return result.Workout
}
func TestCompletionProgressesActualWorkAndEndsOnlyLastOutstandingAttempt(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if e := migrations.Apply(ctx, pool); e != nil {
		t.Fatal(e)
	}
	h, e := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if e != nil {
		t.Fatal(e)
	}
	w := configuredWorkout(t, h)
	// Last listed is not last outstanding. Explicit zero is actual performance.
	w = completionRecord(t, h, w, 1, 0, 1, "performed", 0)
	first := completed(t, h, completionIntent(w, 1, 1))
	w = first.Workout
	if w.Status != "active" || w.EndedAt != nil || w.Exercises[0].State != "draft" || w.Exercises[1].State != "completed" || w.Exercises[1].CompletedAt == nil || *w.Exercises[1].Sets[0].RecordedReps != 0 || w.Exercises[1].CurrentVariant.TargetSets[0].TargetReps != 0 {
		t.Fatalf("last listed: %+v", w)
	}
	w = completionRecord(t, h, w, 0, 0, 2, "performed", 13)
	w = completionRecord(t, h, w, 0, 1, 3, "performed", 11)
	before := w.Exercises[0]
	final := completed(t, h, completionIntent(w, 0, 2))
	w = final.Workout
	if w.Status != "ended" || w.EndedAt == nil || w.Exercises[0].State != "completed" {
		t.Fatalf("atomic end: %+v", w)
	}
	a := w.Exercises[0]
	for i, set := range a.Sets {
		if set.ID != before.Sets[i].ID || *set.RecordedWeight != "40.50" || *set.RecordedReps != *before.Sets[i].RecordedReps || set.TargetWeight != "40.50" || !set.RecordedAt.Equal(*before.Sets[i].RecordedAt) || a.CurrentVariant.TargetSets[i].TargetWeight != "43.00" || a.CurrentVariant.TargetSets[i].TargetReps != 7 {
			t.Fatalf("actual vs future: %+v", a)
		}
	}
	p := request(t, h, "GET", "/api/training/workouts/active", "", "", nil)
	if p.Body.String() != "{\"workout\":null}\n" {
		t.Fatal(p.Body.String())
	}
}

func completionRead(t *testing.T, h http.Handler, id string) training.Workout {
	t.Helper()
	p := request(t, h, "GET", "/api/training/workouts/"+id, "", "", nil)
	var result training.WorkoutResult
	if e := json.Unmarshal(p.Body.Bytes(), &result); p.Code != 200 || e != nil {
		t.Fatalf("read: %d %s", p.Code, p.Body.String())
	}
	return result.Workout
}

func TestCompletionMixedTargetsAndLateOverrideRemainIndependentOfActuals(t *testing.T) {
	for _, late := range []bool{false, true} {
		t.Run(fmt.Sprintf("late override %v", late), func(t *testing.T) {
			pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
			if e := migrations.Apply(ctx, pool); e != nil {
				t.Fatal(e)
			}
			h, e := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
			if e != nil {
				t.Fatal(e)
			}
			w := configuredWorkout(t, h)
			w = configurationSaved(t, h, configurationIntent(w, 1, map[string]any{"prescribedSets": 3, "repThreshold": 11})).Workout
			a := w.Exercises[0]
			for i := 1; i < 3; i++ {
				if _, e = pool.Exec(ctx, `UPDATE public.variant_target_set SET target_weight=50 WHERE id=$1`, *a.Sets[i].TargetSetID); e != nil {
					t.Fatal(e)
				}
				if _, e = pool.Exec(ctx, `UPDATE public.workout_set SET target_weight=50 WHERE id=$1`, a.Sets[i].ID); e != nil {
					t.Fatal(e)
				}
			}
			for i, reps := range []int{12, 11, 10} {
				w = completionRecord(t, h, w, 0, i, i+1, "performed", reps)
			}
			if late {
				w = configurationSaved(t, h, configurationIntent(w, 2, map[string]any{"applyWeightToAllSets": "45.00"})).Workout
			}
			before := w.Exercises[0]
			w = completed(t, h, completionIntent(w, 0, 1)).Workout
			now := w.Exercises[0]
			wantWeights := []string{"43.00", "52.50", "52.50"}
			wantReps := []int{6, 6, 6}
			if late {
				wantWeights = []string{"45.00", "45.00", "45.00"}
				wantReps = []int{12, 11, 10}
			}
			for i, target := range now.CurrentVariant.TargetSets {
				if target.TargetWeight != wantWeights[i] || target.TargetReps != wantReps[i] || *now.Sets[i].RecordedWeight != *before.Sets[i].RecordedWeight || *now.Sets[i].RecordedReps != *before.Sets[i].RecordedReps {
					t.Fatalf("current and actual: %+v", now)
				}
			}
		})
	}
}

func TestCompletionReplaysBeforeConflictsAndRepeatedCompletionCannotProgressLaterWorkout(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if e := migrations.Apply(ctx, pool); e != nil {
		t.Fatal(e)
	}
	h, e := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if e != nil {
		t.Fatal(e)
	}
	w := configuredWorkout(t, h)
	w = completionRecord(t, h, w, 0, 0, 1, "performed", 12)
	w = completionRecord(t, h, w, 0, 1, 2, "performed", 12)
	intent := completionIntent(w, 0, 1)
	accepted := completed(t, h, intent)
	w = accepted.Workout
	actual := w.Exercises[0]
	w = completionRecord(t, h, w, 1, 0, 3, "skipped", 0)
	w = completed(t, h, completionIntent(w, 1, 2)).Workout
	start := `{"operationId":"a3320000-0000-4000-8000-000000000001","sessionId":"` + w.SourceSession.ID + `","expectedSessionRevision":"1"}`
	p := request(t, h, "POST", "/api/training/workouts/start", start, "https://app.example", nil)
	var later training.WorkoutResult
	if e = json.Unmarshal(p.Body.Bytes(), &later); p.Code != 200 || e != nil {
		t.Fatal(p.Body.String())
	}
	later.Workout = completionRecord(t, h, later.Workout, 0, 0, 4, "performed", 12)
	later.Workout = completionRecord(t, h, later.Workout, 0, 1, 5, "performed", 12)
	later = completed(t, h, completionIntent(later.Workout, 0, 3))
	if later.Workout.Exercises[0].CurrentVariant.TargetSets[0].TargetWeight != "45.50" {
		t.Fatal("later workout did not progress")
	}
	replay := completed(t, h, intent)
	if replay.Operation.AcceptedRevision != accepted.Operation.AcceptedRevision || replay.Workout.Revision != w.Revision || !replay.Workout.Exercises[0].CompletedAt.Equal(*actual.CompletedAt) || replay.Workout.Exercises[0].CurrentVariant.TargetSets[0].TargetWeight != "45.50" {
		t.Fatalf("exact replay regressed: %+v", replay)
	}
	intent["operationId"] = "a3300000-0000-4000-8000-000000000004"
	duplicate := completed(t, h, intent)
	if !reflect.DeepEqual(duplicate.Workout, replay.Workout) {
		t.Fatal("new ID completion changed frozen work/current targets")
	}
	changed := completionIntent(w, 0, 1)
	changed["expectedWorkoutRevision"] = w.Revision
	if status, body := sendCompletion(t, h, changed); status != 409 || !strings.Contains(body, "operation-mismatch") {
		t.Fatalf("changed replay: %d %s", status, body)
	}
	late := map[string]any{"operationId": "a3310000-0000-4000-8000-000000000006", "workoutId": w.ID, "attemptId": actual.ID, "setId": actual.Sets[0].ID, "expectedWorkoutRevision": w.Revision, "state": "performed", "recordedReps": 99}
	raw, _ := json.Marshal(late)
	p = request(t, h, "POST", "/api/training/workouts/sets/record", string(raw), "https://app.example", nil)
	if p.Code != 409 || !strings.Contains(p.Body.String(), "resource-finalized") {
		t.Fatalf("late draft write: %d %s", p.Code, p.Body.String())
	}
	active := request(t, h, "GET", "/api/training/workouts/active", "", "", nil)
	if !strings.Contains(active.Body.String(), later.Workout.ID) {
		t.Fatal("old completion affected later active workout")
	}
}

func TestCompletionGuardsPendingOwnershipAndReviewedRevisions(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if e := migrations.Apply(ctx, pool); e != nil {
		t.Fatal(e)
	}
	h, e := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if e != nil {
		t.Fatal(e)
	}
	w := configuredWorkout(t, h)
	before := w
	if status, body := sendCompletion(t, h, completionIntent(w, 0, 1)); status != 422 || !strings.Contains(body, "pending") {
		t.Fatalf("pending: %d %s", status, body)
	}
	if got := completionRead(t, h, w.ID); !reflect.DeepEqual(got, before) {
		t.Fatal("pending completion invented actuals or changed targets")
	}
	w = completionRecord(t, h, w, 0, 0, 1, "skipped", 0)
	w = completionRecord(t, h, w, 0, 1, 2, "performed", 0)
	for i, tc := range []struct {
		field  string
		value  any
		status int
		code   string
	}{
		{"attemptId", w.Exercises[1].ID, 404, "not-found"},
		{"exerciseId", w.Exercises[1].Exercise.ID, 404, "not-found"},
		{"variantId", w.Exercises[1].VariantID, 404, "not-found"},
		{"workoutId", "a3330000-0000-4000-8000-000000000001", 404, "not-found"},
		{"expectedWorkoutRevision", "1", 409, "stale-revision"},
		{"expectedVariantRevision", "9007199254740993", 409, "stale-revision"},
		{"expectedVariantRevision", nil, 422, "invalid-field"},
		{"notes", "not supported", 400, "malformed-request"},
	} {
		c := completionIntent(w, 0, i+2)
		c[tc.field] = tc.value
		status, body := sendCompletion(t, h, c)
		if status != tc.status || !strings.Contains(body, tc.code) {
			t.Fatalf("%s: %d %s", tc.field, status, body)
		}
	}
	// A separately authenticated owner cannot inspect the completion or its parents.
	if _, e = pool.Exec(ctx, `INSERT INTO public."User"(mail,name) VALUES('other@example.com','Other')`); e != nil {
		t.Fatal(e)
	}
	other, e := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{Model: application.ExerciseCentric}, Resolve: func(*http.Request) (application.Identity, error) {
		return application.Identity{Email: "other@example.com"}, nil
	}})
	if e != nil {
		t.Fatal(e)
	}
	if status, body := sendCompletion(t, other, completionIntent(w, 0, 20)); status != 404 {
		t.Fatalf("other owner: %d %s", status, body)
	}
	// Captured configuration and current shared configuration must agree as well.
	if _, e = pool.Exec(ctx, `UPDATE public.exercise_variant SET revision=revision+1 WHERE id=$1`, w.Exercises[0].VariantID); e != nil {
		t.Fatal(e)
	}
	current := completionRead(t, h, w.ID)
	if status, body := sendCompletion(t, h, completionIntent(current, 0, 21)); status != 409 || !strings.Contains(body, "stale-revision") {
		t.Fatalf("captured config: %d %s", status, body)
	}
	if _, e = pool.Exec(ctx, `UPDATE public.exercise_variant SET revision=revision-1 WHERE id=$1`, w.Exercises[0].VariantID); e != nil {
		t.Fatal(e)
	}
	w = completed(t, h, completionIntent(w, 0, 22)).Workout
	a := w.Exercises[0]
	if *a.Sets[1].RecordedReps != 0 || a.Sets[0].RecordedReps != nil || a.CurrentVariant.TargetSets[0].TargetReps != 7 || a.CurrentVariant.TargetSets[1].TargetReps != 0 || a.CurrentVariant.TargetSets[0].TargetWeight != "40.50" {
		t.Fatalf("zero/skip: %+v", a)
	}
}

func TestCompletionBoundsAndFailureRollbackAreAtomicThroughEndingAndReceipt(t *testing.T) {
	for _, stage := range []string{"bounds", "attempt", "ending", "receipt"} {
		t.Run(stage, func(t *testing.T) {
			pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
			if e := migrations.Apply(ctx, pool); e != nil {
				t.Fatal(e)
			}
			h, e := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
			if e != nil {
				t.Fatal(e)
			}
			w := configuredWorkout(t, h)
			// Leave the first attempt as the last outstanding one; the failing completion
			// would otherwise update targets, freeze actuals and automatically end it.
			w = completionRecord(t, h, w, 1, 0, 1, "skipped", 0)
			w = completed(t, h, completionIntent(w, 1, 1)).Workout
			w = completionRecord(t, h, w, 0, 0, 2, "performed", 12)
			w = completionRecord(t, h, w, 0, 1, 3, "performed", 12)
			if stage == "bounds" {
				w = configurationSaved(t, h, configurationIntent(w, 1, map[string]any{"applyWeightToAllSets": "9998.00"})).Workout
				// This configuration preceded the performance for this scenario; remove the
				// deliberately late marker via the public reset/record transitions.
				for i := range 2 {
					w = completionRecord(t, h, w, 0, i, 4+i*2, "pending", 0)
					w = completionRecord(t, h, w, 0, i, 5+i*2, "performed", 12)
				}
			} else {
				table, event := "workout_exercise", "UPDATE"
				if stage == "ending" {
					table = "workout"
				}
				if stage == "receipt" {
					table = "mutation_receipt"
					event = "INSERT"
				}
				sql := fmt.Sprintf(`CREATE FUNCTION public.fail_completion() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced completion failure'; END $$; CREATE TRIGGER fail_completion BEFORE %s ON public.%s FOR EACH ROW EXECUTE FUNCTION public.fail_completion()`, event, table)
				if _, e = pool.Exec(ctx, sql); e != nil {
					t.Fatal(e)
				}
			}
			before := w
			intent := completionIntent(w, 0, 2)
			status, body := sendCompletion(t, h, intent)
			want := 500
			if stage == "bounds" {
				want = 422
			}
			if status != want {
				t.Fatalf("failure: %d %s", status, body)
			}
			if stage == "bounds" && !strings.Contains(body, "Adjust") {
				t.Fatalf("bounds is not actionable: %s", body)
			}
			if after := completionRead(t, h, w.ID); !reflect.DeepEqual(after, before) {
				t.Fatalf("partial completion escaped rollback:\nbefore %+v\nafter %+v", before, after)
			}
			active := request(t, h, "GET", "/api/training/workouts/active", "", "", nil)
			if !strings.Contains(active.Body.String(), w.ID) {
				t.Fatal("failed final completion stranded active workout")
			}
			if stage == "bounds" {
				w = configurationSaved(t, h, configurationIntent(w, 2, map[string]any{"autoIncrement": "0.25"})).Workout
				intent = completionIntent(w, 0, 3)
			} else {
				if _, e = pool.Exec(ctx, `DROP FUNCTION public.fail_completion() CASCADE`); e != nil {
					t.Fatal(e)
				}
			}
			result := completed(t, h, intent)
			if result.Workout.Status != "ended" || result.Workout.Exercises[0].State != "completed" || result.Workout.Exercises[0].CurrentVariant.TargetSets[0].TargetWeight == before.Exercises[0].CurrentVariant.TargetSets[0].TargetWeight {
				t.Fatal("retry did not commit once after rollback")
			}
		})
	}
}

func TestCompletionSerializesIdenticalDuplicateAndCompetingCommands(t *testing.T) {
	for _, race := range []string{"identical", "new operation", "configuration", "recording", "other attempt"} {
		t.Run(race, func(t *testing.T) {
			pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
			if e := migrations.Apply(ctx, pool); e != nil {
				t.Fatal(e)
			}
			h, e := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
			if e != nil {
				t.Fatal(e)
			}
			w := configuredWorkout(t, h)
			w = completionRecord(t, h, w, 0, 0, 1, "performed", 12)
			w = completionRecord(t, h, w, 0, 1, 2, "performed", 12)
			w = completionRecord(t, h, w, 1, 0, 3, "skipped", 0)
			first := completionIntent(w, 0, 1)
			second := completionIntent(w, 0, 1)
			path := "/api/training/workouts/complete"
			switch race {
			case "new operation":
				second["operationId"] = "a3300000-0000-4000-8000-000000000002"
			case "configuration":
				second = configurationIntent(w, 1, map[string]any{"repThreshold": 11})
				path = "/api/training/workouts/configuration"
			case "recording":
				second = map[string]any{"operationId": "a3310000-0000-4000-8000-000000000004", "workoutId": w.ID, "attemptId": w.Exercises[0].ID, "setId": w.Exercises[0].Sets[0].ID, "expectedWorkoutRevision": w.Revision, "state": "performed", "recordedReps": 0}
				path = "/api/training/workouts/sets/record"
			case "other attempt":
				second = completionIntent(w, 1, 2)
			}
			server := httptest.NewServer(h)
			defer server.Close()
			intents := []map[string]any{first, second}
			paths := []string{"/api/training/workouts/complete", path}
			statuses := [2]int{}
			bodies := [2]string{}
			start := make(chan struct{})
			var wg sync.WaitGroup
			for i := range 2 {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					<-start
					raw, _ := json.Marshal(intents[i])
					req, e := http.NewRequest("POST", server.URL+paths[i], strings.NewReader(string(raw)))
					if e != nil {
						t.Error(e)
						return
					}
					req.Header.Set("Origin", "https://app.example")
					r, e := server.Client().Do(req)
					if e != nil {
						t.Error(e)
						return
					}
					defer r.Body.Close()
					b, e := io.ReadAll(r.Body)
					if e != nil {
						t.Error(e)
						return
					}
					statuses[i] = r.StatusCode
					bodies[i] = string(b)
				}(i)
			}
			close(start)
			wg.Wait()
			duplicate := race == "identical" || race == "new operation"
			if duplicate {
				if statuses[0] != 200 || statuses[1] != 200 {
					t.Fatalf("safe duplicate: %v %v", statuses, bodies)
				}
			} else if !((statuses[0] == 200 && statuses[1] == 409) || (statuses[0] == 409 && statuses[1] == 200)) {
				t.Fatalf("serialized competing intents: %v %v", statuses, bodies)
			}
			current := completionRead(t, h, w.ID)
			if duplicate {
				if current.Revision != "5" || current.Exercises[0].CurrentVariant.TargetSets[0].TargetWeight != "43.00" {
					t.Fatalf("progressed twice: %+v", current)
				}
			}
			if race == "other attempt" {
				if current.Status != "active" {
					t.Fatal("ended while an earlier draft remained")
				}
				for i, a := range current.Exercises {
					if a.State == "draft" {
						current = completed(t, h, completionIntent(current, i, 3)).Workout
						break
					}
				}
				if current.Status != "ended" {
					t.Fatal("reviewed retry left workout stranded")
				}
			}
		})
	}
}
