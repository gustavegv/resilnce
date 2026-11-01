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

func TestPersonalLibrarySearchAndMixedSessionReuse(t *testing.T) {
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
	create := func(body string) training.SessionResult {
		w := request(t, h, "POST", "/api/training/sessions/create", body, "https://app.example", nil)
		if w.Code != 200 {
			t.Fatalf("create: %d %s", w.Code, w.Body.String())
		}
		var out training.SessionResult
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	first := create(manualPlan)
	p := first.Session.Placements[0]
	w := request(t, h, "GET", "/api/training/exercises?q=%20BENCH%20", "", "", nil)
	var found struct {
		Exercises []struct {
			Exercise training.Exercise  `json:"exercise"`
			Variants []training.Variant `json:"variants"`
		}
	}
	if err := json.Unmarshal(w.Body.Bytes(), &found); err != nil || w.Code != 200 {
		t.Fatalf("search: %d %s %v", w.Code, w.Body.String(), err)
	}
	if len(found.Exercises) != 1 || found.Exercises[0].Exercise.ID != p.Exercise.ID || len(found.Exercises[0].Variants) != 1 || found.Exercises[0].Variants[0].TargetSets[0].ID != p.Variant.TargetSets[0].ID {
		t.Fatalf("personal search: %s", w.Body.String())
	}
	w = request(t, other, "GET", "/api/training/exercises?q=Bench&mail=owner@example.com", "", "", nil)
	if w.Code != 200 || w.Body.String() != "{\"exercises\":[]}\n" {
		t.Fatalf("search ownership: %d %s", w.Code, w.Body.String())
	}
	w = request(t, other, "GET", "/api/training/exercises/"+p.Exercise.ID, "", "", nil)
	if w.Code != 404 {
		t.Fatalf("cross-owner review: %d %s", w.Code, w.Body.String())
	}
	foreign := fmt.Sprintf(`{"operationId":"10000000-0000-4000-8000-000000000002","name":"Foreign plan","items":[{"kind":"createExercise","name":"Partial foreign","configuration":{"prescribedSets":1,"initialWeight":"0"}},{"kind":"reuseVariant","exerciseId":%q,"variantId":%q,"expectedVariantRevision":"1"}]}`, p.Exercise.ID, p.VariantID)
	w = request(t, other, "POST", "/api/training/sessions/create", foreign, "https://app.example", nil)
	if w.Code != 404 {
		t.Fatalf("cross-owner reuse: %d %s", w.Code, w.Body.String())
	}
	w = request(t, other, "GET", "/api/training/exercises", "", "", nil)
	if w.Body.String() != "{\"exercises\":[]}\n" {
		t.Fatalf("cross-owner mixed draft persisted records: %s", w.Body.String())
	}
	mixed := fmt.Sprintf(`{"operationId":"10000000-0000-4000-8000-000000000002","name":"Mixed","items":[{"kind":"reuseVariant","exerciseId":%q,"variantId":%q,"expectedVariantRevision":%q},{"kind":"createVariant","exerciseId":%q,"expectedExerciseRevision":%q,"name":" Heavy ","configuration":{"prescribedSets":2,"initialWeight":"40.50"}},{"kind":"createExercise","name":"Squat","configuration":{"prescribedSets":1,"initialWeight":"60"}}]}`, p.Exercise.ID, p.VariantID, p.Variant.Revision, p.Exercise.ID, p.Exercise.Revision)
	second := create(mixed)
	if len(second.Session.Placements) != 3 {
		t.Fatalf("mixed plan: %+v", second)
	}
	reused, heavy := second.Session.Placements[0], second.Session.Placements[1]
	if reused.VariantID != p.VariantID || reused.Variant.TargetSets[0].ID != p.Variant.TargetSets[0].ID || reused.Variant.TargetSets[0].TargetWeight != "40.50" || heavy.VariantID == p.VariantID || heavy.Exercise.ID != p.Exercise.ID || heavy.Variant.Name != "Heavy" {
		t.Fatalf("mixed identity: %+v", second.Session.Placements)
	}
	retry := create(mixed)
	if retry.Session.ID != second.Session.ID || retry.Session.Placements[1].VariantID != heavy.VariantID || retry.Operation != second.Operation {
		t.Fatal("retry duplicated mixed plan")
	}
	for _, extra := range []string{`,"name":""`, `,"expectedExerciseRevision":""`, `,"configuration":null`, `,"configuration":{"prescribedSets":0,"initialWeight":""}`} {
		changed := strings.Replace(mixed, `"kind":"reuseVariant"`, `"kind":"reuseVariant"`+extra, 1)
		w = request(t, h, "POST", "/api/training/sessions/create", changed, "https://app.example", nil)
		if w.Code != 409 || !strings.Contains(w.Body.String(), "operation-mismatch") {
			t.Fatalf("changed forbidden field replayed (%s): %d %s", extra, w.Code, w.Body.String())
		}
	}
	w = request(t, h, "GET", "/api/training/exercises/"+p.Exercise.ID, "", "", nil)
	if w.Code != 200 {
		t.Fatalf("exercise review: %d %s", w.Code, w.Body.String())
	}
	var review struct {
		Exercise training.Exercise
		Variants []training.Variant
	}
	if err := json.Unmarshal(w.Body.Bytes(), &review); err != nil {
		t.Fatal(err)
	}
	if len(review.Variants) != 2 {
		t.Fatalf("named equal settings variant missing: %s", w.Body.String())
	}
}

func TestMixedSessionRejectsStaleAndUnavailableReferencesWithoutPartialWrites(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if err != nil {
		t.Fatal(err)
	}
	w := request(t, h, "POST", "/api/training/sessions/create", manualPlan, "https://app.example", nil)
	var first training.SessionResult
	if err := json.Unmarshal(w.Body.Bytes(), &first); err != nil || w.Code != 200 {
		t.Fatalf("seed: %s %v", w.Body.String(), err)
	}
	p := first.Session.Placements[0]
	other := first.Session.Placements[1]
	reuse := fmt.Sprintf(`{"kind":"reuseVariant","exerciseId":%q,"variantId":%q,"expectedVariantRevision":"1"}`, p.Exercise.ID, p.VariantID)
	command := func(item string) string {
		return fmt.Sprintf(`{"operationId":"10000000-0000-4000-8000-000000000002","name":"Rejected plan","items":[{"kind":"createExercise","name":"Keep draft","configuration":{"prescribedSets":1,"initialWeight":"20"}},%s]}`, item)
	}
	cases := []struct {
		name, item string
		status     int
		code       string
	}{
		{"duplicate variant", reuse + "," + reuse, 422, "invalid-field"},
		{"wrong parent", fmt.Sprintf(`{"kind":"reuseVariant","exerciseId":%q,"variantId":%q,"expectedVariantRevision":"1"}`, other.Exercise.ID, p.VariantID), 404, "not-found"},
		{"stale variant", strings.Replace(reuse, `"expectedVariantRevision":"1"`, `"expectedVariantRevision":"2"`, 1), 409, "stale-revision"},
		{"suggested override", strings.TrimSuffix(reuse, "}") + `,"configuration":{"prescribedSets":1,"initialWeight":"99"}}`, 422, "invalid-field"},
		{"null override", strings.TrimSuffix(reuse, "}") + `,"configuration":null}`, 422, "invalid-field"},
		{"stale parent", fmt.Sprintf(`{"kind":"createVariant","exerciseId":%q,"expectedExerciseRevision":"2","name":"Heavy","configuration":{"prescribedSets":1,"initialWeight":"50"}}`, p.Exercise.ID), 409, "stale-revision"},
		{"variant name collision", fmt.Sprintf(`{"kind":"createVariant","exerciseId":%q,"expectedExerciseRevision":"1","name":" sTANDARD ","configuration":{"prescribedSets":1,"initialWeight":"50"}}`, p.Exercise.ID), 409, "name-conflict"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := request(t, h, "POST", "/api/training/sessions/create", command(c.item), "https://app.example", nil)
			if w.Code != c.status || !strings.Contains(w.Body.String(), c.code) {
				t.Fatalf("want %d %s got %d %s", c.status, c.code, w.Code, w.Body.String())
			}
		})
	}
	// Database edits model concurrent accepted commands and imported archive state;
	// all observations below remain at the public HTTP seam.
	if _, err := pool.Exec(ctx, `UPDATE public.exercise_variant SET archived_at=now() WHERE id=$1`, p.VariantID); err != nil {
		t.Fatal(err)
	}
	w = request(t, h, "POST", "/api/training/sessions/create", command(reuse), "https://app.example", nil)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "resource-unavailable") {
		t.Fatalf("archived reuse: %d %s", w.Code, w.Body.String())
	}
	w = request(t, h, "GET", "/api/training/exercises?q=bench", "", "", nil)
	var selection struct {
		Exercises []struct{ Variants []training.Variant }
	}
	if err := json.Unmarshal(w.Body.Bytes(), &selection); err != nil || len(selection.Exercises) != 1 || len(selection.Exercises[0].Variants) != 0 {
		t.Fatalf("archived variant selected: %s", w.Body.String())
	}
	collision := fmt.Sprintf(`{"kind":"createVariant","exerciseId":%q,"expectedExerciseRevision":"1","name":" Standard ","configuration":{"prescribedSets":1,"initialWeight":"50"}}`, p.Exercise.ID)
	w = request(t, h, "POST", "/api/training/sessions/create", command(collision), "https://app.example", nil)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "name-conflict") {
		t.Fatalf("archived name reservation: %d %s", w.Code, w.Body.String())
	}
	if _, err := pool.Exec(ctx, `UPDATE public.exercise SET archived_at=now() WHERE id=$1`, p.Exercise.ID); err != nil {
		t.Fatal(err)
	}
	w = request(t, h, "GET", "/api/training/exercises?q=bench", "", "", nil)
	if w.Body.String() != "{\"exercises\":[]}\n" {
		t.Fatalf("archived exercise selected: %s", w.Body.String())
	}
	w = request(t, h, "GET", "/api/training/exercises/"+p.Exercise.ID, "", "", nil)
	if w.Code != 404 {
		t.Fatalf("archived exercise review: %d", w.Code)
	}
	w = request(t, h, "GET", "/api/training/exercises?q=Keep", "", "", nil)
	if w.Body.String() != "{\"exercises\":[]}\n" {
		t.Fatal("rejected mixed draft created library records")
	}
	w = request(t, h, "GET", "/api/training/sessions", "", "", nil)
	var plans struct{ Sessions []training.Session }
	if err := json.Unmarshal(w.Body.Bytes(), &plans); err != nil || len(plans.Sessions) != 1 {
		t.Fatalf("rejected mixed draft saved partial plan: %s", w.Body.String())
	}
	// A rejected command has no successful receipt, so the same ID can succeed
	// after the user replaces its invalid intent.
	valid := fmt.Sprintf(`{"kind":"reuseVariant","exerciseId":%q,"variantId":%q,"expectedVariantRevision":"1"}`, other.Exercise.ID, other.VariantID)
	w = request(t, h, "POST", "/api/training/sessions/create", command(valid), "https://app.example", nil)
	if w.Code != 200 {
		t.Fatalf("rejected command left receipt: %d %s", w.Code, w.Body.String())
	}
}

func TestConcurrentNamedVariantReviewAndRetry(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if err != nil {
		t.Fatal(err)
	}
	w := request(t, h, "POST", "/api/training/sessions/create", manualPlan, "https://app.example", nil)
	var seed training.SessionResult
	if err := json.Unmarshal(w.Body.Bytes(), &seed); err != nil || w.Code != 200 {
		t.Fatalf("seed: %d %s", w.Code, w.Body.String())
	}
	p := seed.Session.Placements[0]
	bodies := [2]string{}
	for i := range bodies {
		bodies[i] = fmt.Sprintf(`{"operationId":"20000000-0000-4000-8000-%012d","name":"Race %d","items":[{"kind":"createExercise","name":"Race lift %d","configuration":{"prescribedSets":1,"initialWeight":"0"}},{"kind":"createVariant","exerciseId":%q,"expectedExerciseRevision":"1","name":" Heavy ","configuration":{"prescribedSets":2,"initialWeight":"40.50"}}]}`, i+1, i, i, p.Exercise.ID)
	}
	responses := [2]*httptest.ResponseRecorder{}
	var wg sync.WaitGroup
	for i := range responses {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			responses[i] = request(t, h, "POST", "/api/training/sessions/create", bodies[i], "https://app.example", nil)
		}(i)
	}
	wg.Wait()
	winner, loser := -1, -1
	for i, w := range responses {
		if w.Code == 200 {
			winner = i
		} else if w.Code == 409 && strings.Contains(w.Body.String(), "stale-revision") {
			loser = i
		} else {
			t.Fatalf("concurrent named review: %d %s", w.Code, w.Body.String())
		}
	}
	if winner < 0 || loser < 0 {
		t.Fatal("both stale parent reviews succeeded")
	}
	w = request(t, h, "GET", fmt.Sprintf("/api/training/exercises?q=Race%%20lift%%20%d", loser), "", "", nil)
	if w.Body.String() != "{\"exercises\":[]}\n" {
		t.Fatal("losing race persisted partial library")
	}
	// Replay the accepted command after its own parent revision advanced.
	replay := request(t, h, "POST", "/api/training/sessions/create", bodies[winner], "https://app.example", nil)
	var accepted, retried training.SessionResult
	if err := json.Unmarshal(responses[winner].Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(replay.Body.Bytes(), &retried); err != nil {
		t.Fatal(err)
	}
	if replay.Code != 200 || retried.Session.ID != accepted.Session.ID || retried.Operation != accepted.Operation {
		t.Fatal("concurrent winner retry failed")
	}
	// The loser explicitly reviews the new parent revision, then discovers the
	// reserved variant name rather than adding a second equal-settings variant.
	revised := strings.Replace(bodies[loser], `"expectedExerciseRevision":"1"`, `"expectedExerciseRevision":"2"`, 1)
	w = request(t, h, "POST", "/api/training/sessions/create", revised, "https://app.example", nil)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "name-conflict") {
		t.Fatalf("concurrent name identity: %d %s", w.Code, w.Body.String())
	}
	w = request(t, h, "GET", "/api/training/exercises/"+p.Exercise.ID, "", "", nil)
	var current struct {
		Exercise training.Exercise
		Variants []training.Variant
	}
	if err := json.Unmarshal(w.Body.Bytes(), &current); err != nil || len(current.Variants) != 2 || current.Exercise.Revision != "2" {
		t.Fatalf("partial race state: %s", w.Body.String())
	}
}
