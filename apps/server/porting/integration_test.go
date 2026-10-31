//go:build integration

package porting

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gustavegv/resilnce/apps/server/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Only an explicit test DSN is accepted. Each test creates and drops its own
// database; no application .env, DATABASE_URL, or existing schema is touched.
func testService(t *testing.T) (*Service, context.Context) {
	t.Helper()
	s, ctx := testLegacyService(t)
	if err := migrations.Apply(ctx, s.Pool); err != nil {
		t.Fatal(err)
	}
	if err := migrations.Apply(ctx, s.Pool); err != nil {
		t.Fatalf("second migration run: %v", err)
	}
	return s, ctx
}

func testLegacyService(t *testing.T) (*Service, context.Context) {
	t.Helper()
	dsn := os.Getenv("PORTING_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set PORTING_TEST_DATABASE_URL to an isolated PostgreSQL database (role needs CREATEDB)")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid test DSN")
	}
	if !strings.HasSuffix(cfg.ConnConfig.Database, "_test") {
		t.Fatal("test database name must end in _test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	admin, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal("test database connection failed")
	}
	name := "resilnce_porting_" + strings.ReplaceAll(newID(), "-", "") + "_test"
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	copy := cfg.Copy()
	copy.ConnConfig.Database = name
	copy.MaxConns = 8
	pool, err := pgxpool.NewWithConfig(ctx, copy)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		cleanCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if _, err := admin.Exec(cleanCtx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)"); err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	legacy, err := os.ReadFile("testdata/legacy.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(legacy)); err != nil {
		t.Fatal(err)
	}
	return &Service{Pool: pool, AllowApply: true}, ctx
}

func execSQL(t *testing.T, s *Service, ctx context.Context, sql string) {
	t.Helper()
	if _, err := s.Pool.Exec(ctx, sql); err != nil {
		t.Fatal(err)
	}
}
func scalar(t *testing.T, s *Service, ctx context.Context, sql string) int {
	t.Helper()
	var n int
	if err := s.Pool.QueryRow(ctx, sql).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
func applyRun(t *testing.T, s *Service, ctx context.Context, scope string) Run {
	t.Helper()
	id, err := s.StartRun(ctx, "test-operator", "apply", scope, Options{SourceTimezone: "Europe/Stockholm"})
	if err != nil {
		t.Fatal(err)
	}
	worked, err := s.WorkOne(ctx, id)
	if err != nil || !worked {
		t.Fatalf("work: %v, %v", worked, err)
	}
	run, err := s.GetRun(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return run
}

const singleOwnerFixture = `
INSERT INTO public."User"(mail,name) VALUES ('a@example.com','A');
INSERT INTO public."Exercise" VALUES (1,'Bench',2.50,'a@example.com');
INSERT INTO public."Session" VALUES (1,'Push',NULL,'a@example.com');
INSERT INTO public."SesExercise" VALUES (1,1,12,2,ARRAY[12,10],ARRAY[40,50],0,1,true);
INSERT INTO public."History" VALUES ('2026-01-15 12:00:00.123456',10,45.25,2,1);
`

func TestImportReconcilesAndSkipsMappedRows(t *testing.T) {
	s, ctx := testService(t)
	execSQL(t, s, ctx, singleOwnerFixture+`
INSERT INTO public."Exercise" VALUES (2,' bench ',2.50,'a@example.com');
INSERT INTO public."SesExercise" VALUES (1,2,12,2,ARRAY[12,10],ARRAY[40,50],0,1,false);
INSERT INTO public."History" VALUES ('2026-01-15 12:00:00.123457',11,46.25,2,1);
INSERT INTO public."Set" VALUES (1,42.75,9,1,1);
INSERT INTO public."User"(mail) VALUES ('broken@example.com');
INSERT INTO public."Exercise" VALUES (3,'Orphan',NULL,'broken@example.com');
INSERT INTO public."Exercise" VALUES (4,'Unowned',2.5,NULL);
`)
	reports, err := s.Audit(ctx, "", Options{SourceTimezone: "Europe/Stockholm"})
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 3 {
		t.Fatal(reports)
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public.exercise`) != 0 {
		t.Fatal("audit changed the destination")
	}
	run := applyRun(t, s, ctx, "")
	if run.Status != "incomplete" {
		t.Fatalf("broken accounts must stay incomplete: %+v", run)
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public.exercise`) != 1 || scalar(t, s, ctx, `SELECT count(*) FROM public.exercise_variant`) != 2 {
		t.Fatal("duplicate names were not grouped with separate variants")
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public.legacy_history_entry`) != 2 || scalar(t, s, ctx, `SELECT count(*) FROM public.legacy_set_entry`) != 1 {
		t.Fatal("legacy performance missing")
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public.workout`)+scalar(t, s, ctx, `SELECT count(*) FROM public.workout_set`) != 0 {
		t.Fatal("fabricated a workout")
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public.legacy_history_entry WHERE avg_weight=45.25 AND avg_rep=10 AND occurred_at='2026-01-15 11:00:00.123456+00'`) != 1 {
		t.Fatal("legacy values/timestamp changed")
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public.variant_target_set WHERE target_weight=40 AND target_reps=12`) != 2 {
		t.Fatal("targets were progressed or lost")
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public.session_variant WHERE position IN (1,2)`) != 2 {
		t.Fatal("tied source order not resolved")
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM internal.porting_map WHERE owner_key='a@example.com'`) != 9 {
		t.Fatal("not every valid source row has a mapping")
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public."History"`) != 2 {
		t.Fatal("source changed")
	}
	execSQL(t, s, ctx, `UPDATE public.exercise SET name='Renamed'; UPDATE public.exercise_variant SET name='Renamed '||name; UPDATE public.variant_target_set SET target_weight=60;`)
	second := applyRun(t, s, ctx, "a@example.com")
	if second.Status != "completed" || second.Owners[0].Created != 0 {
		t.Fatalf("second run wasn't a no-op: %+v", second)
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public.variant_target_set WHERE target_weight=60`) != 4 {
		t.Fatal("user edits overwritten")
	}
	execSQL(t, s, ctx, `INSERT INTO public."History" VALUES ('2026-01-15 12:00:00.123458',12,49,2,1);`)
	third := applyRun(t, s, ctx, "a@example.com")
	if third.Status != "completed" || third.Owners[0].Counts["History"].Imported != 1 || scalar(t, s, ctx, `SELECT count(*) FROM public.legacy_history_entry`) != 3 {
		t.Fatal(third)
	}
	// New configurations get the durable counter despite user renames.
	execSQL(t, s, ctx, `INSERT INTO public."Session" VALUES (2,'Full body',NULL,'a@example.com'); INSERT INTO public."SesExercise" VALUES (2,1,12,2,ARRAY[12,10],ARRAY[40,50],0,NULL,false);`)
	fourth := applyRun(t, s, ctx, "a@example.com")
	if fourth.Status != "completed" {
		t.Fatal(fourth)
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public.exercise_variant WHERE name='Variant 3'`) != 1 {
		t.Fatal("unstable variant numbering")
	}
	// Previously unambiguous history attribution remains untouched on skip-only reruns.
	if scalar(t, s, ctx, `SELECT count(*) FROM public.legacy_history_entry WHERE variant_id IS NOT NULL`) != 3 {
		t.Fatal("rewrote imported history attribution")
	}
}

func TestImportRollbackAndResumeAfterCommittedGraph(t *testing.T) {
	s, ctx := testService(t)
	execSQL(t, s, ctx, singleOwnerFixture)
	execSQL(t, s, ctx, `CREATE FUNCTION internal.inject_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.source_table='SesExercise' THEN RAISE EXCEPTION 'injected crash'; END IF; RETURN NEW; END $$;
CREATE TRIGGER inject_failure BEFORE INSERT ON internal.porting_map FOR EACH ROW EXECUTE FUNCTION internal.inject_failure();`)
	run := applyRun(t, s, ctx, "")
	if run.Status != "incomplete" || scalar(t, s, ctx, `SELECT count(*) FROM public.exercise`) != 0 || scalar(t, s, ctx, `SELECT count(*) FROM internal.porting_map`) != 0 {
		t.Fatal("graph/mapping did not roll back together")
	}
	if run.Owners[0].Created != 0 || run.Owners[0].Counts["Exercise"].Imported != 0 {
		t.Fatal("rolled-back writes reported as imported")
	}
	execSQL(t, s, ctx, `CREATE OR REPLACE FUNCTION internal.inject_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.source_table='History' THEN RAISE EXCEPTION 'injected crash'; END IF; RETURN NEW; END $$;`)
	if err := s.ResumeRun(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.WorkOne(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public.exercise`) != 1 || scalar(t, s, ctx, `SELECT count(*) FROM public.legacy_history_entry`) != 0 {
		t.Fatal("graph wasn't committed separately from failing history batch")
	}
	execSQL(t, s, ctx, `DROP TRIGGER inject_failure ON internal.porting_map;`)
	if err := s.ResumeRun(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.WorkOne(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	resumed, err := s.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Status != "completed" || scalar(t, s, ctx, `SELECT count(*) FROM public.exercise`) != 1 || scalar(t, s, ctx, `SELECT count(*) FROM public.legacy_history_entry`) != 1 {
		t.Fatal(resumed)
	}
	if err := s.ResumeRun(ctx, run.ID); err != ErrNotResumable {
		t.Fatalf("completed run resumed: %v", err)
	}
}

func TestDeletedDestinationsStayDeleted(t *testing.T) {
	s, ctx := testService(t)
	execSQL(t, s, ctx, singleOwnerFixture)
	if run := applyRun(t, s, ctx, ""); run.Status != "completed" {
		t.Fatal(run)
	}
	execSQL(t, s, ctx, `DELETE FROM public.session_variant; DELETE FROM public.exercise;`)
	if run := applyRun(t, s, ctx, ""); run.Owners[0].Created != 0 {
		t.Fatal("deleted rows resurrected")
	}
	execSQL(t, s, ctx, `INSERT INTO public."History" VALUES ('2026-01-16 12:00:00',12,50,2,1);`)
	run := applyRun(t, s, ctx, "")
	if run.Status != "incomplete" || scalar(t, s, ctx, `SELECT count(*) FROM public.exercise`) != 0 {
		t.Fatal(run)
	}
}

func TestConcurrentRunsAndExpiredLease(t *testing.T) {
	s, ctx := testService(t)
	execSQL(t, s, ctx, singleOwnerFixture)
	ids := []string{}
	for range 2 {
		id, err := s.StartRun(ctx, "operator", "apply", "", Options{SourceTimezone: "UTC"})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, id := range ids {
		wg.Add(1)
		go func(id string) { defer wg.Done(); _, err := s.WorkOne(ctx, id); errs <- err }(id)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public.exercise`) != 1 || scalar(t, s, ctx, `SELECT count(*) FROM public.exercise_variant`) != 1 || scalar(t, s, ctx, `SELECT count(*) FROM internal.porting_map`) != 5 {
		t.Fatal("concurrent workers duplicated data")
	}
	id, err := s.StartRun(ctx, "operator", "audit", "", Options{SourceTimezone: "UTC"})
	if err != nil {
		t.Fatal(err)
	}
	execSQL(t, s, ctx, fmt.Sprintf(`UPDATE internal.porting_run SET status='running',lease_token='%s',lease_until=now()-interval '1 minute' WHERE id='%s'`, newID(), id))
	if worked, err := s.WorkOne(ctx, id); err != nil || !worked {
		t.Fatalf("expired job not reclaimed: %v", err)
	}
	run, err := s.GetRun(ctx, id)
	if err != nil || run.Status != "completed" {
		t.Fatal(run, err)
	}
}

func TestSchemaOwnershipAndDomainInvariants(t *testing.T) {
	s, ctx := testService(t)
	execSQL(t, s, ctx, singleOwnerFixture+`INSERT INTO public."User"(mail) VALUES('b@example.com');`)
	if r := applyRun(t, s, ctx, "a@example.com"); r.Status != "completed" {
		t.Fatal(r)
	}
	var exID string
	if err := s.Pool.QueryRow(ctx, `SELECT id::text FROM public.exercise`).Scan(&exID); err != nil {
		t.Fatal(err)
	}
	queries := []string{
		fmt.Sprintf(`INSERT INTO public.exercise_variant(id,owner_mail,exercise_id,name,progression_mode,rep_threshold,auto_increment,rest_seconds) VALUES('%s','b@example.com','%s','Heavy','average',12,2.5,0)`, newID(), exID),
		fmt.Sprintf(`INSERT INTO public.exercise(id,owner_mail,name) VALUES('%s','a@example.com','bEnCh')`, newID()),
		fmt.Sprintf(`INSERT INTO public.exercise_variant(id,owner_mail,exercise_id,name,progression_mode,rep_threshold,auto_increment,rest_seconds) VALUES('%s','a@example.com','%s','No sets','average',12,2.5,0)`, newID(), exID),
		`DELETE FROM public.variant_target_set`,
	}
	for _, query := range queries {
		if _, err := s.Pool.Exec(ctx, query); err == nil {
			t.Fatalf("constraint accepted invalid write: %s", query)
		}
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='exercise' AND c.relrowsecurity`) != 1 {
		t.Fatal("RLS missing")
	}
}

func TestSourceExceptionsRemainVisiblePerAccount(t *testing.T) {
	s, ctx := testService(t)
	execSQL(t, s, ctx, singleOwnerFixture+`
UPDATE public."User" SET active_session='1';
UPDATE public."Exercise" SET auto_inc=0;
UPDATE public."SesExercise" SET rep_per_set=ARRAY[[12,10]];
INSERT INTO public."User"(mail) VALUES('b@example.com'),('c@example.com');
INSERT INTO public."Exercise" VALUES(2,NULL,NULL,'b@example.com'),(3,'Squat',2.5,'c@example.com');
INSERT INTO public."Session" VALUES(2,'Other',NULL,'b@example.com'),(3,'Legs',NULL,'c@example.com');
INSERT INTO public."SesExercise" VALUES(2,1,12,2,ARRAY[12,10],ARRAY[40,50],0,NULL,false),(3,3,12,2,ARRAY[12,10],ARRAY[40,50],0,1,false);
`)
	reports, err := s.Audit(ctx, "", Options{})
	if err != nil {
		t.Fatal(err)
	}
	byOwner := map[string]OwnerReport{}
	codes := map[string]bool{}
	for _, r := range reports {
		byOwner[r.Owner] = r
		for _, i := range r.Issues {
			codes[i.Code] = true
		}
	}
	if byOwner["a@example.com"].Ready || byOwner["b@example.com"].Ready || !byOwner["c@example.com"].Ready {
		t.Fatal(reports)
	}
	for _, code := range []string{"active_legacy_workout", "invalid_increment", "unsupported_source_values", "invalid_arrays", "cross_owner_reference", "invalid_name"} {
		if !codes[code] {
			t.Errorf("missing source exception %s", code)
		}
	}
	run := applyRun(t, s, ctx, "")
	if run.Status != "incomplete" || scalar(t, s, ctx, `SELECT count(*) FROM public.exercise WHERE owner_mail='c@example.com'`) != 1 {
		t.Fatal(run)
	}
}

func TestInternalAPIQueuesAndInspectsSharedImporter(t *testing.T) {
	s, ctx := testService(t)
	execSQL(t, s, ctx, singleOwnerFixture)
	h := HTTP{Service: s, Origin: "https://app.example", Authenticate: func(*http.Request) (string, error) { return "gustavengvall@yahoo.se", nil }}
	request := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "/internal/porting/"+path, strings.NewReader(body))
		r.Header.Set("Origin", "https://app.example")
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	for _, path := range []string{"audits", "runs"} {
		w := request("POST", path, `{"owner":"a@example.com","options":{"sourceTimezone":"UTC"}}`)
		if w.Code != 202 {
			t.Fatalf("queue %s: %d %s", path, w.Code, w.Body.String())
		}
		var job struct{ ID string }
		if err := json.Unmarshal(w.Body.Bytes(), &job); err != nil {
			t.Fatal(err)
		}
		if scalar(t, s, ctx, `SELECT count(*) FROM public.exercise`) != 0 {
			t.Fatal("HTTP request synchronously imported data")
		}
		if worked, err := s.WorkOne(ctx, job.ID); err != nil || !worked {
			t.Fatal(worked, err)
		}
		w = request("GET", "runs/"+job.ID, "")
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		var run Run
		if err := json.Unmarshal(w.Body.Bytes(), &run); err != nil {
			t.Fatal(err)
		}
		if run.Status != "completed" || run.Actor != "gustavengvall@yahoo.se" || len(run.Owners) != 1 {
			t.Fatal(run)
		}
	}
	w := request("GET", "overview", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"mapped":1`) {
		t.Fatal(w.Code, w.Body.String())
	}
}
