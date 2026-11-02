//go:build integration

package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gustavegv/resilnce/apps/server/application"
	"github.com/gustavegv/resilnce/apps/server/internal/testdb"
	"github.com/gustavegv/resilnce/apps/server/migrations"
	"github.com/gustavegv/resilnce/apps/server/training"
)

func TestGeneratedReviewUsesAuthoritativeMuscleCatalog(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if err != nil {
		t.Fatal(err)
	}
	w := request(t, h, "GET", "/api/training/catalog", "", "", nil)
	var catalog struct {
		ExerciseNames []string `json:"exerciseNames"`
		MuscleGroups  []struct {
			Code, Label string
			SortOrder   int `json:"sort_order"`
		}
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &catalog) != nil || len(catalog.MuscleGroups) != 26 || catalog.MuscleGroups[0].Code != "chest" || catalog.MuscleGroups[0].Label != "Chest" || catalog.MuscleGroups[0].SortOrder != 1 || catalog.MuscleGroups[25].Code != "neck" || catalog.MuscleGroups[25].SortOrder != 4 {
		t.Fatalf("catalog: %d %s", w.Code, w.Body.String())
	}
	if len(catalog.ExerciseNames) != 10 || catalog.ExerciseNames[0] != "Bench press" {
		t.Fatalf("exercise-name suggestions: %s", w.Body.String())
	}
	var exercises int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM public.exercise`).Scan(&exercises); err != nil || exercises != 0 {
		t.Fatalf("reading name suggestions must not create personal exercises: count=%d, err=%v", exercises, err)
	}
	w = request(t, h, "POST", "/api/training/catalog", "{}", "https://app.example", nil)
	if w.Code != 405 {
		t.Fatalf("catalog method: %d %s", w.Code, w.Body.String())
	}
}

func TestGeneratedDraftRemainsReadOnlyUntilAtomicReviewedSave(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENAI_API_KEY", "isolated-test-key")
	t.Setenv("OPENAI_MODEL", "test-model")
	draft := `[{"name":"Bench press","sets":4,"weight":100,"repThreshold":5,"autoIncrease":10}]`
	external := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Errorf("external path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"output": []any{map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": draft}}}}})
	}))
	defer external.Close()
	original := http.DefaultTransport
	transport := original.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // owned local TLS stub only
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, external.Listener.Addr().String())
	}
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = original; transport.CloseIdleConnections() })
	quotaKey := ""
	h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}, AIQuota: func(_ context.Context, key string, limit int64) (bool, error) { quotaKey = key; return true, nil }})
	if err != nil {
		t.Fatal(err)
	}
	w := request(t, h, "POST", "/api/training/sessions/create", manualPlan, "https://app.example", nil)
	var seed training.SessionResult
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &seed) != nil {
		t.Fatalf("seed: %d %s", w.Code, w.Body.String())
	}
	before := seed.Session.Placements[0]
	w = request(t, h, "POST", "/ai/quick?mail=foreign@example.com", `{"userInput":"Bench press 4 sets at 100kg","promptSelections":[false,false,false,false]}`, "https://app.example", nil)
	if w.Code != 200 || w.Body.String() != draft || quotaKey != (application.Identity{Email: "development@resilnce.invalid"}).QuotaKey() {
		t.Fatalf("generated draft/identity: %d %s", w.Code, w.Body.String())
	}
	// Generation and cancellation are read-only. Observe through public operations.
	w = request(t, h, "GET", "/api/training/sessions", "", "", nil)
	var plans struct{ Sessions []training.Session }
	if err := json.Unmarshal(w.Body.Bytes(), &plans); err != nil || len(plans.Sessions) != 1 {
		t.Fatalf("generation stored a plan: %s", w.Body.String())
	}
	w = request(t, h, "GET", "/api/training/exercises/"+before.Exercise.ID, "", "", nil)
	var library training.ExerciseSelection
	if err := json.Unmarshal(w.Body.Bytes(), &library); err != nil || len(library.Variants) != 1 || library.Variants[0].TargetSets[0].TargetWeight != "40.50" {
		t.Fatalf("generation changed library: %s", w.Body.String())
	}
	command := fmt.Sprintf(`{"operationId":"35000000-0000-4000-8000-000000000001","name":"Reviewed draft","items":[{"kind":"reuseVariant","exerciseId":%q,"variantId":%q,"expectedVariantRevision":%q},{"kind":"createVariant","exerciseId":%q,"expectedExerciseRevision":%q,"name":"Suggested","configuration":{"prescribedSets":4,"initialWeight":"100","repThreshold":5,"autoIncrement":"10"}}]}`, before.Exercise.ID, before.Variant.ID, before.Variant.Revision, before.Exercise.ID, before.Exercise.Revision)
	w = request(t, h, "POST", "/api/training/sessions/create", command, "https://app.example", nil)
	var accepted training.SessionResult
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &accepted) != nil {
		t.Fatalf("reviewed create: %d %s", w.Code, w.Body.String())
	}
	if len(accepted.Session.Placements) != 2 || accepted.Session.Placements[0].Variant.ID != before.Variant.ID || accepted.Session.Placements[0].Variant.TargetSets[0].ID != before.Variant.TargetSets[0].ID || accepted.Session.Placements[0].Variant.TargetSets[0].TargetWeight != "40.50" || accepted.Session.Placements[1].Variant.TargetSets[0].TargetWeight != "100.00" {
		t.Fatalf("reviewed choices did not preserve existing configuration: %s", w.Body.String())
	}
	w = request(t, h, "POST", "/api/training/sessions/create", command, "https://app.example", nil)
	var replay training.SessionResult
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &replay) != nil || replay.Session.ID != accepted.Session.ID || replay.Operation != accepted.Operation {
		t.Fatalf("identical review retry: %d %s", w.Code, w.Body.String())
	}
	invalid := strings.Replace(command, "35000000-0000-4000-8000-000000000001", "35000000-0000-4000-8000-000000000002", 1)
	w = request(t, h, "POST", "/api/training/sessions/create", invalid, "https://app.example", nil)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "stale-revision") {
		t.Fatalf("concurrent review must remain stale: %d %s", w.Code, w.Body.String())
	}
	w = request(t, h, "GET", "/api/training/sessions", "", "", nil)
	if err := json.Unmarshal(w.Body.Bytes(), &plans); err != nil || len(plans.Sessions) != 2 {
		t.Fatalf("replay/rejected review stored a duplicate: %s", w.Body.String())
	}
}
