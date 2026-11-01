//go:build integration

package api

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gustavegv/resilnce/apps/server/application"
	"github.com/gustavegv/resilnce/apps/server/internal/testdb"
	"github.com/gustavegv/resilnce/apps/server/migrations"
	"github.com/gustavegv/resilnce/apps/server/training"
)

func TestMaintainingExistingSparseOrderKeepsPlacementIdentity(t *testing.T) {
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
	if err = json.Unmarshal(w.Body.Bytes(), &first); err != nil || w.Code != 200 {
		t.Fatal(w.Code, w.Body.String(), err)
	}
	// An existing imported layout may have sparse positive positions, including
	// values near the storage limit. They are order, never placement identity.
	if _, err = pool.Exec(ctx, `UPDATE public.session_variant SET position=CASE position WHEN 1 THEN 2147483646 ELSE 2147483647 END WHERE session_id=$1`, first.Session.ID); err != nil {
		t.Fatal(err)
	}
	c := fmt.Sprintf(`{"operationId":"10000000-0000-4000-8000-000000000027","expectedRevision":"1","name":"Ordered","items":[{"kind":"retainPlacement","placementId":%q},{"kind":"retainPlacement","placementId":%q}]}`, first.Session.Placements[1].ID, first.Session.Placements[0].ID)
	w = request(t, h, "POST", "/api/training/sessions/"+first.Session.ID+"/update", c, "https://app.example", nil)
	var changed training.SessionResult
	if err = json.Unmarshal(w.Body.Bytes(), &changed); err != nil || w.Code != 200 {
		t.Fatal(w.Code, w.Body.String(), err)
	}
	if changed.Session.Placements[0].ID != first.Session.Placements[1].ID || changed.Session.Placements[0].Position != 1 || changed.Session.Placements[1].Position != 2 {
		t.Fatal("sparse retained order", w.Body.String())
	}
}
