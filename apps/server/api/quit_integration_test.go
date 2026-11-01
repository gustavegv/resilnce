//go:build integration

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/gustavegv/resilnce/apps/server/application"
	"github.com/gustavegv/resilnce/apps/server/internal/testdb"
	"github.com/gustavegv/resilnce/apps/server/migrations"
	"github.com/gustavegv/resilnce/apps/server/training"
)

func quitIntent(w training.Workout, op int) map[string]any {
	return map[string]any{"operationId": fmt.Sprintf("a3400000-0000-4000-8000-%012d", op), "workoutId": w.ID, "expectedWorkoutRevision": w.Revision}
}
func sendQuit(t *testing.T, h http.Handler, c map[string]any) (int, string) {
	t.Helper()
	raw, _ := json.Marshal(c)
	p := request(t, h, "POST", "/api/training/workouts/quit", string(raw), "https://app.example", nil)
	return p.Code, p.Body.String()
}
func quitSaved(t *testing.T, h http.Handler, c map[string]any) training.WorkoutResult {
	t.Helper()
	status, body := sendQuit(t, h, c)
	var out training.WorkoutResult
	if e := json.Unmarshal([]byte(body), &out); status != 200 || e != nil {
		t.Fatalf("quit: %d %s", status, body)
	}
	return out
}
func TestQuitRetainsCompletedWorkProgressionAndSharedConfiguration(t *testing.T) {
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
	w = completed(t, h, completionIntent(w, 0, 1)).Workout
	w = completionRecord(t, h, w, 1, 0, 3, "performed", 9)
	before := w.Exercises[0]
	draftVariant := w.Exercises[1].CurrentVariant
	result := quitSaved(t, h, quitIntent(w, 1))
	ended := result.Workout
	if ended.Status != "ended" || ended.EndedAt == nil || len(ended.Exercises) != 1 || !reflect.DeepEqual(ended.Exercises[0], before) {
		t.Fatalf("retained completion: %+v", ended)
	}
	if ended.Revision == w.Revision {
		t.Fatal("quit did not advance revision")
	}
	active := request(t, h, "GET", "/api/training/workouts/active", "", "", nil)
	if active.Body.String() != "{\"workout\":null}\n" {
		t.Fatal(active.Body.String())
	}
	selection := request(t, h, "GET", "/api/training/exercises/"+w.Exercises[1].Exercise.ID, "", "", nil)
	var read training.ExerciseSelection
	if e = json.Unmarshal(selection.Body.Bytes(), &read); e != nil {
		t.Fatal(e)
	}
	if len(read.Variants) != 1 || !reflect.DeepEqual(read.Variants[0], draftVariant) {
		t.Fatal("quit changed shared configuration")
	}
}

func TestQuitWithoutCompletionPreservesEditsCreatesNoRecencyAndReplaysOnlyOldWorkout(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if e := migrations.Apply(ctx, pool); e != nil {
		t.Fatal(e)
	}
	h, e := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if e != nil {
		t.Fatal(e)
	}
	w := configuredWorkout(t, h)
	w = configurationSaved(t, h, configurationIntent(w, 1, map[string]any{"exerciseName": "Edited press", "variantName": "Heavy", "repThreshold": 10, "autoIncrement": "1.25", "restSeconds": 90, "applyWeightToAllSets": "55.25"})).Workout
	w = completionRecord(t, h, w, 0, 0, 1, "performed", 12)
	intent := quitIntent(w, 1)
	accepted := quitSaved(t, h, intent)
	if accepted.Workout.Status != "ended" || accepted.Workout.EndedAt == nil || len(accepted.Workout.Exercises) != 0 {
		t.Fatalf("invented history: %+v", accepted)
	}
	if got := completionRead(t, h, w.ID); !reflect.DeepEqual(got, accepted.Workout) {
		t.Fatal("quit did not persist")
	}
	p := request(t, h, "GET", "/api/training/sessions/"+w.SourceSession.ID, "", "", nil)
	var plan training.SessionResult
	if e = json.Unmarshal(p.Body.Bytes(), &plan); p.Code != 200 || e != nil {
		t.Fatal(p.Body.String())
	}
	if plan.Session.LastPerformedAt != nil || plan.Session.LastPerformedStatus != "never" {
		t.Fatalf("unfinished created recency: %+v", plan.Session)
	}
	if !reflect.DeepEqual(plan.Session.Placements[0].Variant, w.Exercises[0].CurrentVariant) || plan.Session.Placements[0].Exercise.Name != "Edited press" {
		t.Fatal("quit reverted configuration")
	}
	// Starting again uses edited configuration and does not reintroduce discarded actuals.
	p = request(t, h, "POST", "/api/training/workouts/start", `{"operationId":"a3410000-0000-4000-8000-000000000001","sessionId":"`+w.SourceSession.ID+`","expectedSessionRevision":"1"}`, "https://app.example", nil)
	var later training.WorkoutResult
	if e = json.Unmarshal(p.Body.Bytes(), &later); p.Code != 200 || e != nil {
		t.Fatal(p.Body.String())
	}
	if later.Workout.ID == w.ID || later.Workout.Exercises[0].Sets[0].State != "pending" || later.Workout.Exercises[0].CurrentVariant.Name != "Heavy" {
		t.Fatalf("later workout: %+v", later)
	}
	replay := quitSaved(t, h, intent)
	if !reflect.DeepEqual(replay, accepted) {
		t.Fatal("lost-response retry changed ended outcome")
	}
	intent["operationId"] = "a3400000-0000-4000-8000-000000000002"
	repeated := quitSaved(t, h, intent)
	if !reflect.DeepEqual(repeated.Workout, accepted.Workout) {
		t.Fatal("new ID quit changed ended outcome")
	}
	if got := completionRead(t, h, later.Workout.ID); !reflect.DeepEqual(got, later.Workout) {
		t.Fatal("old quit affected newer active workout")
	}
	// The receipt cannot be repurposed, even for a valid later workout.
	changed := quitIntent(later.Workout, 1)
	if status, body := sendQuit(t, h, changed); status != 409 || !strings.Contains(body, "operation-mismatch") {
		t.Fatalf("repurposed receipt: %d %s", status, body)
	}
}

func TestQuitGuardsOwnershipRevisionAndStrictInput(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if e := migrations.Apply(ctx, pool); e != nil {
		t.Fatal(e)
	}
	h, e := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if e != nil {
		t.Fatal(e)
	}
	w := configuredWorkout(t, h)
	stale := quitIntent(w, 1)
	w = completionRecord(t, h, w, 0, 0, 1, "performed", 12)
	if status, body := sendQuit(t, h, stale); status != 409 || !strings.Contains(body, "stale-revision") {
		t.Fatalf("stale: %d %s", status, body)
	}
	for i, tc := range []struct {
		field  string
		value  any
		status int
		code   string
	}{
		{"workoutId", "a3420000-0000-4000-8000-000000000001", 404, "not-found"},
		{"workoutId", "active", 422, "invalid-field"},
		{"expectedWorkoutRevision", nil, 422, "invalid-field"},
		{"expectedWorkoutRevision", "01", 422, "invalid-field"},
		{"expectedWorkoutRevision", 2, 400, "malformed-request"},
		{"complete", true, 400, "malformed-request"},
	} {
		c := quitIntent(w, i+2)
		c[tc.field] = tc.value
		status, body := sendQuit(t, h, c)
		if status != tc.status || !strings.Contains(body, tc.code) {
			t.Fatalf("%s: %d %s", tc.field, status, body)
		}
	}
	for _, body := range []string{"null", "{} {}"} {
		p := request(t, h, "POST", "/api/training/workouts/quit", body, "https://app.example", nil)
		if p.Code != 400 {
			t.Fatalf("strict body: %d %s", p.Code, p.Body.String())
		}
	}
	if _, e = pool.Exec(ctx, `INSERT INTO public."User"(mail,name) VALUES('other@example.com','Other')`); e != nil {
		t.Fatal(e)
	}
	other, e := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{Model: application.ExerciseCentric}, Resolve: func(*http.Request) (application.Identity, error) {
		return application.Identity{Email: "other@example.com"}, nil
	}})
	if e != nil {
		t.Fatal(e)
	}
	if status, body := sendQuit(t, other, quitIntent(w, 20)); status != 404 {
		t.Fatalf("cross-owner: %d %s", status, body)
	}
	if got := completionRead(t, h, w.ID); !reflect.DeepEqual(got, w) {
		t.Fatal("rejected quit changed workout")
	}
	accepted := quitSaved(t, h, quitIntent(w, 21))
	if status, _ := sendQuit(t, other, quitIntent(accepted.Workout, 21)); status != 404 {
		t.Fatal("cross-owner accessed ended receipt")
	}
}

func TestQuitRollsBackDraftSetDeletionEndingAndReceiptTogether(t *testing.T) {
	for _, stage := range []string{"set deletion", "ending", "receipt"} {
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
			w = completionRecord(t, h, w, 0, 0, 1, "skipped", 0)
			w = completionRecord(t, h, w, 0, 1, 2, "performed", 0)
			w = completed(t, h, completionIntent(w, 0, 1)).Workout
			w = completionRecord(t, h, w, 1, 0, 3, "performed", 9)
			table, event := "workout_set", "DELETE"
			if stage == "ending" {
				table, event = "workout", "UPDATE"
			}
			if stage == "receipt" {
				table, event = "mutation_receipt", "INSERT"
			}
			sql := fmt.Sprintf(`CREATE FUNCTION public.fail_quit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced quit failure'; END $$; CREATE TRIGGER fail_quit BEFORE %s ON public.%s FOR EACH ROW EXECUTE FUNCTION public.fail_quit()`, event, table)
			if _, e = pool.Exec(ctx, sql); e != nil {
				t.Fatal(e)
			}
			intent := quitIntent(w, 1)
			if status, body := sendQuit(t, h, intent); status != 500 {
				t.Fatalf("forced failure: %d %s", status, body)
			}
			if got := completionRead(t, h, w.ID); !reflect.DeepEqual(got, w) {
				t.Fatal("partial quit escaped rollback")
			}
			p := request(t, h, "GET", "/api/training/workouts/active", "", "", nil)
			if !strings.Contains(p.Body.String(), w.ID) {
				t.Fatal("failed quit stranded active workout")
			}
			if _, e = pool.Exec(ctx, `DROP FUNCTION public.fail_quit() CASCADE`); e != nil {
				t.Fatal(e)
			}
			got := quitSaved(t, h, intent).Workout
			if got.Status != "ended" || len(got.Exercises) != 1 || !reflect.DeepEqual(got.Exercises[0], w.Exercises[0]) {
				t.Fatal("same operation did not succeed after rollback")
			}
		})
	}
}

func TestQuitSerializesDuplicateAndCompetingCompletion(t *testing.T) {
	for _, race := range []string{"identical", "new operation", "completion"} {
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
			intents := []map[string]any{quitIntent(w, 1), quitIntent(w, 1)}
			paths := []string{"/api/training/workouts/quit", "/api/training/workouts/quit"}
			if race == "new operation" {
				intents[1] = quitIntent(w, 2)
			}
			if race == "completion" {
				intents[1] = completionIntent(w, 0, 1)
				paths[1] = "/api/training/workouts/complete"
			}
			server := httptest.NewServer(h)
			defer server.Close()
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
			current := completionRead(t, h, w.ID)
			if race != "completion" {
				if statuses[0] != 200 || statuses[1] != 200 || current.Status != "ended" || len(current.Exercises) != 0 || current.Revision != "4" {
					t.Fatalf("duplicate quit: %v %v %+v", statuses, bodies, current)
				}
				return
			}
			if statuses[1] == 200 {
				if statuses[0] != 409 || !strings.Contains(bodies[0], "stale-revision") || current.Exercises[0].State != "completed" || current.Status != "active" {
					t.Fatalf("quit overtook completion: %v %v", statuses, bodies)
				}
				current = quitSaved(t, h, quitIntent(current, 4)).Workout
				if len(current.Exercises) != 1 || current.Exercises[0].CurrentVariant.TargetSets[0].TargetWeight != "43.00" {
					t.Fatal("reviewed quit lost accepted completion")
				}
			} else {
				if statuses[0] != 200 || statuses[1] != 404 || current.Status != "ended" || len(current.Exercises) != 0 {
					t.Fatalf("completion overtook quit: %v %v", statuses, bodies)
				}
			}
			// Whichever wins, replay cannot change the authoritative outcome.
			repeated := quitSaved(t, h, quitIntent(w, 3)).Workout
			if !reflect.DeepEqual(repeated, current) {
				t.Fatal("repeat changed serialized result")
			}
		})
	}
}
