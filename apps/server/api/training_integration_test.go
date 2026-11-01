//go:build integration

package api

import (
	"encoding/json"
	"fmt"
	"github.com/gustavegv/resilnce/apps/server/training"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gustavegv/resilnce/apps/server/application"
	"github.com/gustavegv/resilnce/apps/server/internal/testdb"
	"github.com/gustavegv/resilnce/apps/server/migrations"
)

const manualPlan = `{"operationId":"10000000-0000-4000-8000-000000000001","name":"  Upper body  ","items":[{"kind":"createExercise","name":"  Bench press  ","muscleGroupCodes":[],"configuration":{"prescribedSets":2,"initialWeight":"40.5"}},{"kind":"createExercise","name":"Row","muscleGroupCodes":[],"configuration":{"prescribedSets":1,"initialWeight":"0","repThreshold":11,"autoIncrement":"0.25","restSeconds":60}}]}`

func TestManualSessionCreatesOrderedStandardVariantsAndReadsOwnedPlan(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if err != nil {
		t.Fatal(err)
	}
	w := request(t, h, "POST", "/api/training/sessions/create", manualPlan, "https://app.example", nil)
	if w.Code != 200 {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created struct {
		Operation struct{ OperationID, AcceptedRevision string }
		Session   struct {
			ID, Name, Revision string
			Placements         []struct {
				ID       string
				Position int
				Exercise struct{ ID, Name, Revision string }
				Variant  struct {
					ID, Name, Revision, AutoIncrement, ProgressionMode string
					RepThreshold, PrescribedSets, RestSeconds          int
					TargetSets                                         []struct {
						ID, TargetWeight     string
						Position, TargetReps int
					}
				}
			}
		}
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	s := created.Session
	if s.Name != "Upper body" || s.Revision != "1" || len(s.ID) != 36 || created.Operation.AcceptedRevision != "1" || created.Operation.OperationID != "10000000-0000-4000-8000-000000000001" || len(s.Placements) != 2 {
		t.Fatalf("session contract: %s", w.Body.String())
	}
	first := s.Placements[0]
	if first.Position != 1 || len(first.ID) != 36 || first.Exercise.Name != "Bench press" || len(first.Exercise.ID) != 36 || first.Exercise.Revision != "1" || first.Variant.Name != "Standard" || len(first.Variant.ID) != 36 || first.Variant.Revision != "1" || first.Variant.AutoIncrement != "2.50" || first.Variant.ProgressionMode != "average" || first.Variant.RepThreshold != 12 || first.Variant.PrescribedSets != 2 || len(first.Variant.TargetSets) != 2 {
		t.Fatalf("variant contract: %s", w.Body.String())
	}
	for i, target := range first.Variant.TargetSets {
		if len(target.ID) != 36 || target.Position != i+1 || target.TargetReps != 12 || target.TargetWeight != "40.50" {
			t.Fatalf("target contract: %+v", target)
		}
	}
	if s.Placements[1].Position != 2 || s.Placements[1].Variant.TargetSets[0].TargetWeight != "0.00" || s.Placements[1].Variant.TargetSets[0].TargetReps != 11 {
		t.Fatal("order/zero/threshold not retained")
	}
	read := request(t, h, "GET", "/api/training/sessions/"+s.ID+"?mail=other@example.com", "", "", nil)
	if read.Code != 200 || !strings.Contains(read.Body.String(), first.Variant.ID) {
		t.Fatalf("read: %d %s", read.Code, read.Body.String())
	}
	list := request(t, h, "GET", "/api/training/sessions", "", "", nil)
	if list.Code != 200 || !strings.Contains(list.Body.String(), s.ID) || !strings.Contains(list.Body.String(), `"lastPerformedAt":null`) {
		t.Fatalf("list: %d %s", list.Code, list.Body.String())
	}
	var legacy int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM public."Session")+(SELECT count(*) FROM public."Exercise")+(SELECT count(*) FROM public."SesExercise")`).Scan(&legacy); err != nil || legacy != 0 {
		t.Fatalf("new writes reached legacy storage: %d %v", legacy, err)
	}
}

func TestManualSessionPlacementBoundsCatalogOrderAndCompletedRecency(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if err != nil {
		t.Fatal(err)
	}
	var c training.CreateSession
	if err := json.Unmarshal([]byte(manualPlan), &c); err != nil {
		t.Fatal(err)
	}
	c.Items = c.Items[:1]
	c.Items[0].MuscleGroupCodes = training.Codes{"back", "chest"}
	c.Items[0].Name = "\t" + strings.Repeat("é", 100) + "\n"
	c.Items[0].Configuration.InitialWeight = "9999.00"
	c.Items[0].Configuration.PrescribedSets = 20
	raw, _ := json.Marshal(c)
	w := request(t, h, "POST", "/api/training/sessions/create", string(raw), "https://app.example", nil)
	var first training.SessionResult
	if err := json.Unmarshal(w.Body.Bytes(), &first); w.Code != 200 || err != nil {
		t.Fatalf("valid limits: %d %s", w.Code, w.Body.String())
	}
	if first.Session.Placements[0].Exercise.Name != strings.Repeat("é", 100) || strings.Join(first.Session.Placements[0].Exercise.MuscleGroupCodes, ",") != "chest,back" {
		t.Fatalf("trim/catalog order: %s", w.Body.String())
	}
	c.OperationID = "10000000-0000-4000-8000-000000000002"
	c.Name = "100 placements"
	c.Items = []training.SessionItem{}
	for i := range 100 {
		c.Items = append(c.Items, training.SessionItem{Kind: "createExercise", Name: fmt.Sprintf("Lift %d", i), Configuration: training.Configuration{PrescribedSets: 1, InitialWeight: "0"}})
	}
	raw, _ = json.Marshal(c)
	w = request(t, h, "POST", "/api/training/sessions/create", string(raw), "https://app.example", nil)
	var second training.SessionResult
	if err := json.Unmarshal(w.Body.Bytes(), &second); w.Code != 200 || err != nil || len(second.Session.Placements) != 100 {
		t.Fatalf("100 placements: %d %s", w.Code, w.Body.String())
	}
	c.OperationID = "10000000-0000-4000-8000-000000000003"
	c.Items = append(c.Items, training.SessionItem{Kind: "createExercise", Name: "Extra", Configuration: training.Configuration{PrescribedSets: 1, InitialWeight: "0"}})
	raw, _ = json.Marshal(c)
	w = request(t, h, "POST", "/api/training/sessions/create", string(raw), "https://app.example", nil)
	if w.Code != 422 {
		t.Fatalf("101 placements: %d", w.Code)
	}
	// Starting alone is not performance; a completed attempt in an active workout
	// is performance even before its remaining exercises are completed.
	workoutID := "70000000-0000-4000-8000-000000000001"
	attemptID := "80000000-0000-4000-8000-000000000001"
	if _, err := pool.Exec(ctx, `INSERT INTO public.workout(id,owner_mail,source_session_id,origin,status) VALUES($1,'development@resilnce.invalid',$2,'session','active')`, workoutID, first.Session.ID); err != nil {
		t.Fatal(err)
	}
	list := request(t, h, "GET", "/api/training/sessions", "", "", nil)
	if strings.Count(list.Body.String(), `"lastPerformedAt":null`) != 2 {
		t.Fatal("start fabricated performance")
	}
	if _, err := pool.Exec(ctx, `INSERT INTO public.workout_exercise(id,owner_mail,workout_id,variant_id,position,state,origin,progression_mode,rep_threshold,auto_increment,config_revision,completed_at) VALUES($1,'development@resilnce.invalid',$2,$3,1,'completed','planned','average',12,2.5,1,'2026-11-01T10:00:00Z')`, attemptID, workoutID, first.Session.Placements[0].VariantID); err != nil {
		t.Fatal(err)
	}
	list = request(t, h, "GET", "/api/training/sessions", "", "", nil)
	var result struct{ Sessions []training.Session }
	if err := json.Unmarshal(list.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Sessions) != 2 || result.Sessions[0].ID != first.Session.ID || result.Sessions[0].LastPerformedAt == nil || result.Sessions[0].LastPerformedAt.UTC().Format(time.RFC3339) != "2026-11-01T10:00:00Z" || result.Sessions[1].LastPerformedAt != nil {
		t.Fatalf("completed recency: %s", list.Body.String())
	}
	w = request(t, h, "GET", "/api/training/sessions/bad-id", "", "", nil)
	if w.Code != 422 {
		t.Fatalf("invalid identity: %d %s", w.Code, w.Body.String())
	}
}

func TestManualSessionRetryReturnsCommittedIdentityAndRejectsChangedIntent(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if err != nil {
		t.Fatal(err)
	}
	var responses [2]*httptest.ResponseRecorder
	var wg sync.WaitGroup
	for i := range responses {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			responses[i] = request(t, h, "POST", "/api/training/sessions/create", manualPlan, "https://app.example", nil)
		}(i)
	}
	wg.Wait()
	var first training.SessionResult
	for _, w := range responses {
		if w.Code != 200 {
			t.Fatalf("concurrent replay: %d %s", w.Code, w.Body.String())
		}
		var got training.SessionResult
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if first.Session.ID == "" {
			first = got
		} else if got.Session.ID != first.Session.ID || got.Session.Placements[0].ID != first.Session.Placements[0].ID || got.Operation != first.Operation {
			t.Fatal("retry created a second identity/acknowledgment")
		}
	}
	w := request(t, h, "POST", "/api/training/sessions/create", strings.Replace(manualPlan, "Upper body", "Changed", 1), "https://app.example", nil)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "operation-mismatch") {
		t.Fatalf("changed operation: %d %s", w.Code, w.Body.String())
	}
	w = request(t, h, "POST", "/api/training/sessions/create", strings.Replace(manualPlan, "Upper body", " ", 1), "https://app.example", nil)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "operation-mismatch") {
		t.Fatalf("reused operation with invalid changed intent: %d %s", w.Code, w.Body.String())
	}
	// Simulate a later accepted edit: replay returns current state alongside the
	// original receipt, never regressing the caller's revision to that receipt.
	if _, err := pool.Exec(ctx, `UPDATE public.training_session SET name='Renamed',revision=9007199254740993 WHERE id=$1`, first.Session.ID); err != nil {
		t.Fatal(err)
	}
	w = request(t, h, "POST", "/api/training/sessions/create", manualPlan, "https://app.example", nil)
	var retry training.SessionResult
	if err := json.Unmarshal(w.Body.Bytes(), &retry); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || retry.Session.Name != "Renamed" || retry.Session.Revision != "9007199254740993" || retry.Operation.AcceptedRevision != "1" || retry.Operation != first.Operation {
		t.Fatalf("replay lost current state: %s", w.Body.String())
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM public.mutation_receipt`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("receipt invariant: %d %v", count, err)
	}
}

func TestManualSessionOwnershipNameConflictsAndFailureAreAtomic(t *testing.T) {
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
	owner, other := handler("owner@example.com"), handler("other@example.com")
	w := request(t, owner, "POST", "/api/training/sessions/create", manualPlan, "https://app.example", nil)
	var created training.SessionResult
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || w.Code != 200 {
		t.Fatalf("create %d %s %v", w.Code, w.Body.String(), err)
	}
	w = request(t, other, "GET", "/api/training/sessions/"+created.Session.ID, "", "", nil)
	if w.Code != 404 {
		t.Fatalf("cross-owner read: %d %s", w.Code, w.Body.String())
	}
	w = request(t, other, "GET", "/api/training/sessions?mail=owner@example.com", "", "", nil)
	if w.Body.String() != "{\"sessions\":[]}\n" {
		t.Fatal("owner chosen by browser")
	}
	w = request(t, other, "POST", "/api/training/sessions/create", manualPlan, "https://app.example", nil)
	if w.Code != 200 {
		t.Fatalf("independent owner naming/receipt: %d %s", w.Code, w.Body.String())
	}
	duplicate := strings.Replace(manualPlan, "10000000-0000-4000-8000-000000000001", "10000000-0000-4000-8000-000000000002", 1)
	duplicate = strings.Replace(duplicate, "  Bench press  ", " BENCH PRESS ", 1)
	w = request(t, owner, "POST", "/api/training/sessions/create", duplicate, "https://app.example", nil)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "name-conflict") {
		t.Fatalf("name identity: %d %s", w.Code, w.Body.String())
	}
	// The database boundary fails the second exercise after the first has already
	// been inserted; neither library records, placements, plan nor receipt survive.
	if _, err := pool.Exec(ctx, `CREATE FUNCTION internal.fail_manual_create() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.name='Failure' THEN RAISE EXCEPTION 'forced failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER fail_manual_create BEFORE INSERT ON public.exercise FOR EACH ROW EXECUTE FUNCTION internal.fail_manual_create()`); err != nil {
		t.Fatal(err)
	}
	broken := strings.ReplaceAll(duplicate, " BENCH PRESS ", "New lift")
	broken = strings.Replace(broken, `"name":"Row"`, `"name":"Failure"`, 1)
	w = request(t, owner, "POST", "/api/training/sessions/create", broken, "https://app.example", nil)
	if w.Code != 500 {
		t.Fatalf("forced failure: %d %s", w.Code, w.Body.String())
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM public.exercise WHERE name IN('New lift','Failure'))+(SELECT count(*) FROM public.mutation_receipt WHERE operation_id='10000000-0000-4000-8000-000000000002')`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("failure left storage/receipt: %d %v", count, err)
	}
	list := request(t, owner, "GET", "/api/training/sessions", "", "", nil)
	var plans struct{ Sessions []training.Session }
	if err := json.Unmarshal(list.Body.Bytes(), &plans); err != nil || len(plans.Sessions) != 1 {
		t.Fatalf("partial plan remained: %s", list.Body.String())
	}
	if _, err := pool.Exec(ctx, `DROP TRIGGER fail_manual_create ON public.exercise`); err != nil {
		t.Fatal(err)
	}
	w = request(t, owner, "POST", "/api/training/sessions/create", broken, "https://app.example", nil)
	if w.Code != 200 {
		t.Fatalf("failed command not retryable: %d %s", w.Code, w.Body.String())
	}
}

func TestManualSessionRejectsInvalidDraftsWithoutPartialLibraryOrPlan(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, body string
		status     int
	}{
		{"null command", "null", 400},
		{"empty session", strings.Replace(manualPlan, `"name":"  Upper body  "`, `"name":"  "`, 1), 422},
		{"empty exercise", strings.Replace(manualPlan, "  Bench press  ", "  ", 1), 422},
		{"long name", strings.Replace(manualPlan, "  Bench press  ", strings.Repeat("é", 101), 1), 422},
		{"zero placements", `{"operationId":"10000000-0000-4000-8000-000000000001","name":"Plan","items":[]}`, 422},
		{"zero sets", strings.Replace(manualPlan, `"prescribedSets":2`, `"prescribedSets":0`, 1), 422},
		{"too many sets", strings.Replace(manualPlan, `"prescribedSets":2`, `"prescribedSets":21`, 1), 422},
		{"negative weight", strings.Replace(manualPlan, `"40.5"`, `"-1"`, 1), 422},
		{"weight precision", strings.Replace(manualPlan, `"40.5"`, `"40.501"`, 1), 422},
		{"weight limit", strings.Replace(manualPlan, `"40.5"`, `"9999.01"`, 1), 422},
		{"nan", strings.Replace(manualPlan, `"40.5"`, `"NaN"`, 1), 422},
		{"float weight", strings.Replace(manualPlan, `"40.5"`, `40.5`, 1), 400},
		{"threshold", strings.Replace(manualPlan, `"repThreshold":11`, `"repThreshold":100`, 1), 422},
		{"null threshold", strings.Replace(manualPlan, `"repThreshold":11`, `"repThreshold":null`, 1), 422},
		{"null increment", strings.Replace(manualPlan, `"autoIncrement":"0.25"`, `"autoIncrement":null`, 1), 422},
		{"null rest", strings.Replace(manualPlan, `"restSeconds":60`, `"restSeconds":null`, 1), 422},
		{"increment", strings.Replace(manualPlan, `"autoIncrement":"0.25"`, `"autoIncrement":"0.24"`, 1), 422},
		{"increment precision", strings.Replace(manualPlan, `"autoIncrement":"0.25"`, `"autoIncrement":"0.251"`, 1), 422},
		{"negative rest", strings.Replace(manualPlan, `"restSeconds":60`, `"restSeconds":-1`, 1), 422},
		{"rest overflow", strings.Replace(manualPlan, `"restSeconds":60`, `"restSeconds":2147483648`, 1), 422},
		{"unknown group", strings.Replace(manualPlan, `"muscleGroupCodes":[]`, `"muscleGroupCodes":["pecs"]`, 1), 422},
		{"duplicate group", strings.Replace(manualPlan, `"muscleGroupCodes":[]`, `"muscleGroupCodes":["chest","chest"]`, 1), 422},
		{"null group", strings.Replace(manualPlan, `"muscleGroupCodes":[]`, `"muscleGroupCodes":[null]`, 1), 422},
		{"null groups", strings.Replace(manualPlan, `"muscleGroupCodes":[]`, `"muscleGroupCodes":null`, 1), 422},
		{"unsupported intent", strings.Replace(manualPlan, `"createExercise"`, `"reuseVariant"`, 1), 422},
		{"operation uuid", strings.Replace(manualPlan, "10000000-0000-4000-8000-000000000001", "bad", 1), 422},
		{"owner field", strings.Replace(manualPlan, `"name":"  Upper body  "`, `"name":"Plan","ownerMail":"other@example.com"`, 1), 400},
		{"unsupported feature", strings.Replace(manualPlan, `"prescribedSets":2`, `"prescribedSets":2,"progressionMode":"off"`, 1), 400},
		{"trailing json", manualPlan + `{}`, 400},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := request(t, h, "POST", "/api/training/sessions/create", tc.body, "https://app.example", nil)
			if w.Code != tc.status {
				t.Fatalf("want %d, got %d %s", tc.status, w.Code, w.Body.String())
			}
		})
	}
	list := request(t, h, "GET", "/api/training/sessions", "", "", nil)
	if list.Body.String() != "{\"sessions\":[]}\n" {
		t.Fatalf("invalid draft left a plan: %s", list.Body.String())
	}
	var records int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM public.exercise)+(SELECT count(*) FROM public.exercise_variant)+(SELECT count(*) FROM public.variant_target_set)+(SELECT count(*) FROM public.mutation_receipt)`).Scan(&records); err != nil || records != 0 {
		t.Fatalf("invalid draft left storage: %d %v", records, err)
	}
}
