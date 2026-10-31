//go:build integration

package porting

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

const unusedHistoryFixture = `
INSERT INTO public."Exercise" VALUES(0,'Unused historical',NULL,'a@example.com');
INSERT INTO public."History" VALUES('2026-01-15 13:00:00.123456',10,40,2,0),('2026-01-15 13:00:00.123457',11,40,2,0);
`

func TestUnusedHistoryCleanupThroughAPI(t *testing.T) {
	s, ctx := testService(t)
	execSQL(t, s, ctx, singleOwnerFixture+unusedHistoryFixture)
	h := schemaHTTP(s)
	start := func(path string) Run {
		t.Helper()
		r := httptest.NewRequest("POST", "/internal/porting/"+path, strings.NewReader(`{"owner":"a@example.com","options":{"deleteUnusedExercises":true,"deleteUnusedHistory":true}}`))
		r.Header.Set("Origin", "https://app.example")
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 202 {
			t.Fatal(w.Code, w.Body.String())
		}
		var job struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &job); err != nil {
			t.Fatal(err)
		}
		if _, err := s.WorkOne(ctx, job.ID); err != nil {
			t.Fatal(err)
		}
		run, err := s.GetRun(ctx, job.ID)
		if err != nil {
			t.Fatal(err)
		}
		return run
	}
	audit := start("audits")
	if audit.Status != "completed" || !audit.Options.DeleteUnusedHistory || audit.Owners[0].Counts["Exercise"].DeletePlanned != 1 || audit.Owners[0].Counts["History"].DeletePlanned != 2 {
		t.Fatal(audit)
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public."History"`) != 3 || scalar(t, s, ctx, `SELECT count(*) FROM public."Exercise"`) != 2 {
		t.Fatal("audit deleted source data")
	}
	run := start("runs")
	if run.Status != "completed" {
		t.Fatal(run)
	}
	ex, hist := run.Owners[0].Counts["Exercise"], run.Owners[0].Counts["History"]
	if ex.Deleted != 1 || hist.Deleted != 2 || hist.Imported != 1 || hist.Pending != 0 || hist.DeletePlanned != 0 || hist.Retained != 1 {
		t.Fatal(ex, hist)
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public."Exercise"`) != 1 || scalar(t, s, ctx, `SELECT count(*) FROM public."History"`) != 1 || scalar(t, s, ctx, `SELECT count(*) FROM public.legacy_history_entry`) != 1 {
		t.Fatal("cleanup removed unrelated history or imported deleted history")
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM internal.porting_map WHERE source_table='History' AND destinations->>'deletedUnused'='true'`) != 2 {
		t.Fatal("history deletion source keys missing")
	}
	second := start("runs")
	if second.Status != "completed" || second.Owners[0].Created != 0 || second.Owners[0].Counts["History"].Deleted != 0 {
		t.Fatal("cleanup rerun was not a no-op", second)
	}
	items, err := s.Inventory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.Table == "History" && (item.Source != 1 || item.Mapped != 1) {
			t.Fatal("history deletion inflated inventory", item)
		}
	}
}

func TestHistoryCleanupAndMappingsRollbackTogether(t *testing.T) {
	s, ctx := testService(t)
	execSQL(t, s, ctx, singleOwnerFixture+unusedHistoryFixture+`
CREATE FUNCTION internal.fail_history_cleanup() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN IF NEW.source_table='History' AND NEW.destinations->>'deletedUnused'='true' THEN RAISE EXCEPTION 'injected history cleanup failure'; END IF; RETURN NEW; END $$;
CREATE TRIGGER fail_history_cleanup BEFORE INSERT ON internal.porting_map FOR EACH ROW EXECUTE FUNCTION internal.fail_history_cleanup();`)
	id, err := s.StartRun(ctx, "test", "apply", "a@example.com", Options{DeleteUnusedExercises: true, DeleteUnusedHistory: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.WorkOne(ctx, id); err != nil {
		t.Fatal(err)
	}
	run, err := s.GetRun(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "incomplete" || run.Owners[0].Counts["History"].Deleted != 0 || run.Owners[0].Counts["Exercise"].Deleted != 0 || run.Owners[0].Created != 0 {
		t.Fatal(run)
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public."Exercise"`) != 2 || scalar(t, s, ctx, `SELECT count(*) FROM public."History"`) != 3 || scalar(t, s, ctx, `SELECT count(*) FROM internal.porting_map`) != 0 {
		t.Fatal("failed mapping left source cleanup committed")
	}
	execSQL(t, s, ctx, `DROP TRIGGER fail_history_cleanup ON internal.porting_map`)
	if err = s.ResumeRun(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err = s.WorkOne(ctx, id); err != nil {
		t.Fatal(err)
	}
	run, err = s.GetRun(ctx, id)
	if err != nil || run.Status != "completed" || run.Owners[0].Counts["History"].Deleted != 2 || !run.Options.DeleteUnusedHistory {
		t.Fatal(run, err)
	}
}

func TestHistoryCleanupRejectsChangedSourceKeysAndNewPlacements(t *testing.T) {
	for _, tc := range []struct{ name, sql, want string }{
		{"new history", `INSERT INTO public."History" VALUES('2026-01-17 12:00:00',12,40,2,0)`, "history changed"},
		{"replaced key", `UPDATE public."History" SET save_date='2026-01-17 12:00:00' WHERE ex_id=0 AND save_date='2026-01-15 13:00:00.123456'`, "history changed"},
		{"new placement", `INSERT INTO public."SesExercise" VALUES(1,0,12,2,ARRAY[12,10],ARRAY[40,50],0,2,false)`, "now in use"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, ctx := testService(t)
			execSQL(t, s, ctx, singleOwnerFixture+unusedHistoryFixture)
			all, err := s.readSource(ctx)
			if err != nil {
				t.Fatal(err)
			}
			execSQL(t, s, ctx, tc.sql)
			report, err := s.ImportOwner(ctx, all, "a@example.com", Options{DeleteUnusedExercises: true, DeleteUnusedHistory: true})
			if err == nil || !strings.Contains(err.Error(), tc.want) || report.Ready || report.Counts["History"].Deleted != 0 {
				t.Fatal(report, err)
			}
			if scalar(t, s, ctx, `SELECT count(*) FROM public."Exercise"`) != 2 || scalar(t, s, ctx, `SELECT count(*) FROM internal.porting_map`) != 0 {
				t.Fatal("stale snapshot deleted source data")
			}
		})
	}
}
