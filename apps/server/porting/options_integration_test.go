//go:build integration

package porting

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUnusedExerciseOptionAndDefaultsThroughAPI(t *testing.T) {
	s, ctx := testService(t)
	execSQL(t, s, ctx, singleOwnerFixture+`
UPDATE public."SesExercise" SET rest_seconds=NULL;
INSERT INTO public."Exercise" VALUES(2,'Unused',NULL,'a@example.com');`)
	h := schemaHTTP(s)
	request := func(path string) Run {
		t.Helper()
		// Exercise the same JSON options sent by the page checkbox.
		body := `{"owner":"a@example.com","options":{"deleteUnusedExercises":true,"sourceTimezone":""}}`
		r := httptest.NewRequest("POST", "/internal/porting/"+path, strings.NewReader(body))
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
	audit := request("audits")
	if audit.Status != "completed" || audit.Options.SourceTimezone != DefaultSourceTimezone || !audit.Options.DeleteUnusedExercises || audit.Owners[0].Counts["Exercise"].DeletePlanned != 1 || audit.Owners[0].Counts["Exercise"].Deleted != 0 {
		t.Fatal(audit)
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public."Exercise"`) != 2 || scalar(t, s, ctx, `SELECT count(*) FROM public.exercise`) != 0 {
		t.Fatal("audit changed workout data")
	}
	run := request("runs")
	if run.Status != "completed" {
		t.Fatal(run)
	}
	c := run.Owners[0].Counts["Exercise"]
	if c.Source != 2 || c.Imported != 1 || c.Deleted != 1 || c.DeletePlanned != 0 || c.Pending != 0 {
		t.Fatal(c)
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public."Exercise"`) != 1 || scalar(t, s, ctx, `SELECT count(*) FROM public.exercise_variant WHERE rest_seconds=0`) != 1 {
		t.Fatal("unused exercise/default rest not applied")
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public.legacy_history_entry WHERE source_timezone='Europe/Stockholm' AND occurred_at='2026-01-15 11:00:00.123456+00'`) != 1 {
		t.Fatal("default timezone not applied")
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM internal.porting_map WHERE source_table='Exercise' AND source_key='[2]'::jsonb AND destinations->>'deletedUnused'='true' AND source_data->>'ex_name'='Unused'`) != 1 {
		t.Fatal("deletion audit record missing")
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM internal.porting_group WHERE normalized_name='unused'`) != 0 {
		t.Fatal("deleted source created a destination group")
	}
	second := request("runs")
	if second.Status != "completed" || second.Owners[0].Created != 0 || second.Owners[0].Counts["Exercise"].Deleted != 0 {
		t.Fatal("rerun wasn't a no-op", second)
	}
	items, err := s.Inventory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range items {
		if i.Table == "Exercise" && (i.Source != 1 || i.Mapped != 1) {
			t.Fatal("deletion tombstones inflated inventory", i)
		}
	}
}

func TestUnusedDeletionPreservesHistoricalAndImportedExercises(t *testing.T) {
	t.Run("history is retained", func(t *testing.T) {
		s, ctx := testService(t)
		execSQL(t, s, ctx, singleOwnerFixture+`INSERT INTO public."Exercise" VALUES(2,'Historical',2.5,'a@example.com'); INSERT INTO public."History" VALUES('2026-01-15 13:00:00',10,40,2,2);`)
		all, err := s.readSource(ctx)
		if err != nil {
			t.Fatal(err)
		}
		report, err := s.ImportOwner(ctx, all, "a@example.com", Options{DeleteUnusedExercises: true})
		if err != nil || report.Ready || report.Counts["Exercise"].DeletePlanned != 0 || report.Counts["Exercise"].Deleted != 0 {
			t.Fatal(report, err)
		}
		if scalar(t, s, ctx, `SELECT count(*) FROM public."History"`) != 2 || scalar(t, s, ctx, `SELECT count(*) FROM public."Exercise"`) != 2 {
			t.Fatal("historical source exercise was deleted")
		}
	})
	t.Run("mapped exercise is preserved", func(t *testing.T) {
		s, ctx := testService(t)
		execSQL(t, s, ctx, singleOwnerFixture+`DELETE FROM public."History";`)
		if run := applyRun(t, s, ctx, ""); run.Status != "completed" {
			t.Fatal(run)
		}
		execSQL(t, s, ctx, `DELETE FROM public."SesExercise";`)
		all, err := s.readSource(ctx)
		if err != nil {
			t.Fatal(err)
		}
		report, err := s.ImportOwner(ctx, all, "a@example.com", Options{DeleteUnusedExercises: true})
		if err != nil || !report.Ready || report.Counts["Exercise"].Deleted != 0 || scalar(t, s, ctx, `SELECT count(*) FROM public."Exercise"`) != 1 {
			t.Fatal(report, err)
		}
	})
}

func TestUnusedDeletionRollsBackWithGraphAndResumes(t *testing.T) {
	s, ctx := testService(t)
	execSQL(t, s, ctx, singleOwnerFixture+`
INSERT INTO public."Exercise" VALUES(0,'Unused',NULL,'a@example.com');
CREATE FUNCTION internal.fail_after_deletion() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.source_table='Session' THEN RAISE EXCEPTION 'injected failure'; END IF; RETURN NEW; END $$;
CREATE TRIGGER fail_after_deletion BEFORE INSERT ON internal.porting_map FOR EACH ROW EXECUTE FUNCTION internal.fail_after_deletion();`)
	id, err := s.StartRun(ctx, "test", "apply", "a@example.com", Options{DeleteUnusedExercises: true})
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
	if run.Status != "incomplete" || run.Owners[0].Counts["Exercise"].Deleted != 0 || scalar(t, s, ctx, `SELECT count(*) FROM public."Exercise"`) != 2 || scalar(t, s, ctx, `SELECT count(*) FROM internal.porting_map`) != 0 {
		t.Fatal("failed graph retained partial deletion", run)
	}
	execSQL(t, s, ctx, `DROP TRIGGER fail_after_deletion ON internal.porting_map;`)
	if err = s.ResumeRun(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err = s.WorkOne(ctx, id); err != nil {
		t.Fatal(err)
	}
	run, err = s.GetRun(ctx, id)
	if err != nil || run.Status != "completed" || run.Owners[0].Counts["Exercise"].Deleted != 1 {
		t.Fatal(run, err)
	}
}

func TestUnusedDeletionRechecksNewReferencesBeforeDeleting(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"history", `INSERT INTO public."History" VALUES('2026-01-15 13:00:00',10,40,2,0)`},
		{"session", `INSERT INTO public."SesExercise" VALUES(1,0,12,2,ARRAY[12,10],ARRAY[40,50],0,2,false)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, ctx := testService(t)
			execSQL(t, s, ctx, singleOwnerFixture+`INSERT INTO public."Exercise" VALUES(0,'Unused',2.5,'a@example.com');`)
			all, err := s.readSource(ctx)
			if err != nil {
				t.Fatal(err)
			}
			execSQL(t, s, ctx, tc.sql)
			report, err := s.ImportOwner(ctx, all, "a@example.com", Options{DeleteUnusedExercises: true})
			if err == nil || !strings.Contains(err.Error(), "now in use") || report.Ready || report.Counts["Exercise"].Deleted != 0 {
				t.Fatal(report, err)
			}
			if scalar(t, s, ctx, `SELECT count(*) FROM public."Exercise"`) != 2 || scalar(t, s, ctx, `SELECT count(*) FROM internal.porting_map`) != 0 {
				t.Fatal("stale audit deleted a newly referenced exercise")
			}
		})
	}
}

func TestExplicitSourceTimezoneStillOverridesDefault(t *testing.T) {
	s, ctx := testService(t)
	execSQL(t, s, ctx, singleOwnerFixture)
	id, err := s.StartRun(ctx, "test", "apply", "a@example.com", Options{SourceTimezone: "UTC"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.WorkOne(ctx, id); err != nil {
		t.Fatal(err)
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public.legacy_history_entry WHERE source_timezone='UTC' AND occurred_at='2026-01-15 12:00:00.123456+00'`) != 1 {
		t.Fatal("explicit timezone overridden")
	}
}
