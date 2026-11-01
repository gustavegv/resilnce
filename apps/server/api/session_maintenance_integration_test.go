//go:build integration

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gustavegv/resilnce/apps/server/application"
	"github.com/gustavegv/resilnce/apps/server/internal/testdb"
	"github.com/gustavegv/resilnce/apps/server/migrations"
	"github.com/gustavegv/resilnce/apps/server/porting"
	"github.com/gustavegv/resilnce/apps/server/training"
)

func TestSavedSessionOrderedMaintenanceAndDeletion(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{DevelopmentMode: true}})
	if err != nil {
		t.Fatal(err)
	}
	w := request(t, h, "POST", "/api/training/sessions/create", manualPlan, "https://app.example", nil)
	var created training.SessionResult
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || w.Code != 200 {
		t.Fatal(w.Code, w.Body.String(), err)
	}
	a, b := created.Session.Placements[0], created.Session.Placements[1]
	path := "/api/training/sessions/" + created.Session.ID
	w = request(t, h, "POST", path+"/update", "null", "https://app.example", nil)
	if w.Code != 400 {
		t.Fatal("null maintenance command shape", w.Code, w.Body.String())
	}
	update := fmt.Sprintf(`{"operationId":"10000000-0000-4000-8000-000000000020","expectedRevision":"1","name":"Renamed","items":[{"kind":"retainPlacement","placementId":%q},{"kind":"retainPlacement","placementId":%q},{"kind":"createVariant","exerciseId":%q,"expectedExerciseRevision":"1","name":"Heavy","configuration":{"prescribedSets":1,"initialWeight":"80"}}]}`, b.ID, a.ID, a.Exercise.ID)
	w = request(t, h, "POST", path+"/update", update, "https://app.example", nil)
	var changed training.SessionResult
	if err := json.Unmarshal(w.Body.Bytes(), &changed); err != nil || w.Code != 200 {
		t.Fatalf("update: %d %s %v", w.Code, w.Body.String(), err)
	}
	if changed.Session.Name != "Renamed" || changed.Session.Revision != "2" || len(changed.Session.Placements) != 3 || changed.Session.Placements[0].ID != b.ID || changed.Session.Placements[1].ID != a.ID || changed.Session.Placements[1].Variant.TargetSets[0].ID != a.Variant.TargetSets[0].ID {
		t.Fatalf("retained identities/order: %+v", changed)
	}
	w = request(t, h, "POST", path+"/update", update, "https://app.example", nil)
	var replay training.SessionResult
	if err := json.Unmarshal(w.Body.Bytes(), &replay); err != nil || replay.Operation != changed.Operation || replay.Session.Placements[2].ID != changed.Session.Placements[2].ID {
		t.Fatal("replay changed identities", w.Body.String())
	}
	for _, items := range []string{
		fmt.Sprintf(`[{"kind":"retainPlacement","placementId":%q},{"kind":"retainPlacement","placementId":%q}]`, a.ID, a.ID),
		fmt.Sprintf(`[{"kind":"retainPlacement","placementId":%q},{"kind":"reuseVariant","exerciseId":%q,"variantId":%q,"expectedVariantRevision":"1"}]`, a.ID, a.Exercise.ID, a.VariantID),
		fmt.Sprintf(`[{"kind":"retainPlacement","placementId":%q,"name":""}]`, a.ID),
	} {
		invalid := fmt.Sprintf(`{"operationId":"10000000-0000-4000-8000-000000000024","expectedRevision":"2","name":"Rejected","items":%s}`, items)
		w = request(t, h, "POST", path+"/update", invalid, "https://app.example", nil)
		if w.Code != 422 {
			t.Fatal("duplicate/forbidden placement accepted", w.Code, w.Body.String())
		}
	}
	w = request(t, h, "POST", path+"/update", strings.Replace(update, "Renamed", "Changed intent", 1), "https://app.example", nil)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "operation-mismatch") {
		t.Fatal("changed maintenance replay", w.Code, w.Body.String())
	}
	if _, err = pool.Exec(ctx, `INSERT INTO public."User"(mail) VALUES('other@example.com')`); err != nil {
		t.Fatal(err)
	}
	other, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{Model: application.ExerciseCentric}, Resolve: func(*http.Request) (application.Identity, error) {
		return application.Identity{Email: "other@example.com"}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	w = request(t, other, "POST", path+"/update", strings.Replace(update, "000000000020", "000000000025", 1), "https://app.example", nil)
	if w.Code != 404 {
		t.Fatal("cross-owner update accepted", w.Code, w.Body.String())
	}
	w = request(t, other, "POST", path+"/delete", `{"operationId":"10000000-0000-4000-8000-000000000026","expectedRevision":"2"}`, "https://app.example", nil)
	if w.Code != 404 {
		t.Fatal("cross-owner delete accepted", w.Code, w.Body.String())
	}
	w = request(t, h, "GET", path, "", "", nil)
	var unchanged struct{ Session training.Session }
	if err = json.Unmarshal(w.Body.Bytes(), &unchanged); err != nil || unchanged.Session.Revision != "2" || unchanged.Session.Name != "Renamed" || unchanged.Session.Placements[0].ID != b.ID {
		t.Fatal("rejected edit changed plan", w.Code, w.Body.String())
	}
	stale := fmt.Sprintf(`{"operationId":"10000000-0000-4000-8000-000000000021","expectedRevision":"1","name":"Lost edit","items":[]}`)
	w = request(t, h, "POST", path+"/update", stale, "https://app.example", nil)
	if w.Code != 409 {
		t.Fatal("stale edit accepted", w.Code, w.Body.String())
	}
	empty := `{"operationId":"10000000-0000-4000-8000-000000000022","expectedRevision":"2","name":"Empty","items":[]}`
	w = request(t, h, "POST", path+"/update", empty, "https://app.example", nil)
	if w.Code != 200 {
		t.Fatal("empty edit", w.Code, w.Body.String())
	}
	w = request(t, h, "GET", "/api/training/exercises/"+a.Exercise.ID, "", "", nil)
	if w.Code != 200 {
		t.Fatal("removal deleted library", w.Body.String())
	}
	deletion := `{"operationId":"10000000-0000-4000-8000-000000000023","expectedRevision":"3"}`
	w = request(t, h, "POST", path+"/delete", deletion, "https://app.example", nil)
	if w.Code != 200 {
		t.Fatal("delete", w.Code, w.Body.String())
	}
	accepted := w.Body.String()
	w = request(t, h, "POST", path+"/delete", deletion, "https://app.example", nil)
	if w.Code != 200 || w.Body.String() != accepted {
		t.Fatal("delete receipt replay", w.Code, w.Body.String())
	}
	w = request(t, h, "GET", path, "", "", nil)
	if w.Code != 404 {
		t.Fatal("deleted session visible", w.Code, w.Body.String())
	}
	w = request(t, h, "GET", "/api/training/exercises/"+a.Exercise.ID, "", "", nil)
	if w.Code != 200 {
		t.Fatal("deletion deleted library", w.Body.String())
	}
}

func TestImportedSessionRecencyIsAccountScopedAndDoesNotGuessOldTimezone(t *testing.T) {
	pool, ctx := testdb.New(t, "../porting/testdata/legacy.sql")
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO public."User"(mail) VALUES('owner@example.com'),('other@example.com'); INSERT INTO public."Session" VALUES(1,'Imported','2026-01-15 12:00:00.123456','owner@example.com'),(2,'Never',NULL,'owner@example.com');`); err != nil {
		t.Fatal(err)
	}
	importer := porting.Service{Pool: pool, AllowApply: true}
	id, err := importer.StartRun(ctx, "test", "apply", "owner@example.com", porting.Options{SourceTimezone: "Europe/Stockholm"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = importer.WorkOne(ctx, id); err != nil {
		t.Fatal(err)
	}
	handler := func(email string) http.Handler {
		h, err := NewHandler(ctx, Options{Pool: pool, Origin: "https://app.example", Application: application.Config{Model: application.ExerciseCentric}, Resolve: func(*http.Request) (application.Identity, error) { return application.Identity{Email: email}, nil }})
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	h := handler("owner@example.com")
	w := request(t, h, "GET", "/api/training/sessions", "", "", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"lastPerformedAt":"2026-01-15T11:00:00.123456Z"`) || !strings.Contains(w.Body.String(), `"lastPerformedStatus":"never"`) {
		t.Fatal("imported dates", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "source_timezone") || strings.Contains(w.Body.String(), "sourceTimezone") || strings.Contains(w.Body.String(), "source_data") || strings.Contains(w.Body.String(), "migration_version") {
		t.Fatal("private bookkeeping leaked", w.Body.String())
	}
	var listed struct{ Sessions []training.Session }
	if err = json.Unmarshal(w.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO public."Session" VALUES(3,'UTC equal','2026-01-15 11:00:00.123456','owner@example.com'),(4,'UTC summer','2026-07-15 12:00:00.123456','owner@example.com');`); err != nil {
		t.Fatal(err)
	}
	id, err = importer.StartRun(ctx, "test", "apply", "owner@example.com", porting.Options{SourceTimezone: "UTC"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = importer.WorkOne(ctx, id); err != nil {
		t.Fatal(err)
	}
	w = request(t, h, "GET", "/api/training/sessions", "", "", nil)
	var ordered struct{ Sessions []training.Session }
	if err = json.Unmarshal(w.Body.Bytes(), &ordered); err != nil {
		t.Fatal(err)
	}
	if len(ordered.Sessions) != 4 || ordered.Sessions[0].Name != "UTC summer" || ordered.Sessions[0].LastPerformedAt.UTC().Format("2006-01-02T15:04:05.999999Z") != "2026-07-15T12:00:00.123456Z" || ordered.Sessions[1].LastPerformedAt.UTC().Format("2006-01-02T15:04:05.999999Z") != "2026-01-15T11:00:00.123456Z" || !ordered.Sessions[1].LastPerformedAt.Equal(*ordered.Sessions[2].LastPerformedAt) || ordered.Sessions[1].ID > ordered.Sessions[2].ID || ordered.Sessions[3].Name != "Never" {
		t.Fatal("mixed import policies/equal instant order", w.Code, w.Body.String())
	}
	if _, err = pool.Exec(ctx, `UPDATE internal.porting_map SET source_timezone=NULL,session_last_ran_at=NULL WHERE source_table='Session' AND source_key='[1]'::jsonb;`); err != nil {
		t.Fatal(err)
	}
	for _, zone := range []string{"UTC", "Europe/Stockholm"} {
		id, err = importer.StartRun(ctx, "test", "apply", "owner@example.com", porting.Options{SourceTimezone: zone})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = importer.WorkOne(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	w = request(t, h, "GET", "/api/training/sessions/"+listed.Sessions[0].ID, "", "", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"lastPerformedStatus":"unknown"`) || !strings.Contains(w.Body.String(), `"lastPerformedAt":null`) {
		t.Fatal("old policy guessed", w.Code, w.Body.String())
	}
	w = request(t, handler("other@example.com"), "GET", "/api/training/sessions", "", "", nil)
	if w.Code != 200 || w.Body.String() != "{\"sessions\":[]}\n" {
		t.Fatal("cross-owner recency leaked", w.Code, w.Body.String())
	}
	w = request(t, h, "POST", "/api/training/sessions/"+listed.Sessions[0].ID+"/delete", `{"operationId":"10000000-0000-4000-8000-000000000028","expectedRevision":"1"}`, "https://app.example", nil)
	if w.Code != 200 {
		t.Fatal("delete imported plan", w.Code, w.Body.String())
	}
	id, err = importer.StartRun(ctx, "test", "apply", "owner@example.com", porting.Options{SourceTimezone: "UTC"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = importer.WorkOne(ctx, id); err != nil {
		t.Fatal(err)
	}
	run, err := importer.GetRun(ctx, id)
	if err != nil || run.Status != "completed" || run.Owners[0].Created != 0 || run.Owners[0].Counts["Session"].Mapped != 4 {
		t.Fatal("deleted destination mapping lost", run, err)
	}
	w = request(t, h, "GET", "/api/training/sessions/"+listed.Sessions[0].ID, "", "", nil)
	if w.Code != 404 {
		t.Fatal("import resurrected deleted plan", w.Code, w.Body.String())
	}
}
