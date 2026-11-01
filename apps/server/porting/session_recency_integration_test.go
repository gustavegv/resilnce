//go:build integration

package porting

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gustavegv/resilnce/apps/server/migrations"
)

func TestSessionRecencyUpgradeLeavesOldPoliciesUnknownAndTombstonesDurable(t *testing.T) {
	s, ctx := testLegacyService(t)
	installMigrationVersions(t, s, ctx, "001_exercise_centric.sql", "002_porting.sql", "003_exercise_rpe.sql", "004_notes.sql")
	execSQL(t, s, ctx, `INSERT INTO public."User"(mail) VALUES('a@example.com');
 INSERT INTO public."Session" VALUES(1,'Old dated','2026-01-15 12:00:00.123456','a@example.com'),(2,'Old null',NULL,'a@example.com'),(3,'Deleted','2026-07-15 12:00:00','a@example.com');
 INSERT INTO public.training_session(id,owner_mail,name) VALUES('20000000-0000-4000-8000-000000000001','a@example.com','Old dated'),('20000000-0000-4000-8000-000000000002','a@example.com','Old null');
 INSERT INTO internal.porting_map(migration_version,source_table,source_key,owner_key,destinations,source_data)
 SELECT 'exercise-centric-v1','Session',jsonb_build_array(ses_id),'a@example.com',jsonb_build_object('session','20000000-0000-4000-8000-00000000000'||ses_id),to_jsonb(s) FROM public."Session" s;
 INSERT INTO internal.porting_run(id,migration_version,actor,mode,status,options) VALUES('60000000-0000-4000-8000-000000000001','exercise-centric-v1','test','apply','completed','{"sourceTimezone":"UTC"}'),('60000000-0000-4000-8000-000000000002','exercise-centric-v1','test','audit','completed','{"sourceTimezone":"Europe/Stockholm"}');`)
	var before []byte
	if err := s.Pool.QueryRow(ctx, `SELECT jsonb_agg(to_jsonb(m) ORDER BY source_key) FROM internal.porting_map m`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	status, err := migrations.Inspect(ctx, s.Pool)
	if err != nil || !status.CanApply || status.Ready {
		t.Fatal(status, err)
	}
	for range 2 {
		if err = migrations.Apply(ctx, s.Pool); err != nil {
			t.Fatal(err)
		}
	}
	status, err = migrations.Inspect(ctx, s.Pool)
	if err != nil || !status.Ready {
		t.Fatal(status, err)
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM internal.schema_migration WHERE version IN ('001_exercise_centric.sql','002_porting.sql','003_exercise_rpe.sql','004_notes.sql') AND applied_at='2000-01-01T00:00:00Z'`) != 4 {
		t.Fatal("historical migrations changed")
	}
	for _, zone := range []string{"UTC", "Europe/Stockholm"} {
		id, err := s.StartRun(ctx, "test", "apply", "a@example.com", Options{SourceTimezone: zone})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.WorkOne(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM internal.porting_map WHERE source_table='Session' AND source_timezone IS NULL AND session_last_ran_at IS NULL`) != 3 || scalar(t, s, ctx, `SELECT count(*) FROM public.training_session`) != 2 {
		t.Fatal("upgrade/rerun guessed metadata or resurrected deleted session")
	}
	var after []byte
	if err = s.Pool.QueryRow(ctx, `SELECT jsonb_agg(to_jsonb(m)-'source_timezone'-'session_last_ran_at' ORDER BY source_key) FROM internal.porting_map m WHERE source_table='Session'`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	var a, b any
	if err = json.Unmarshal(before, &a); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(after, &b); err != nil {
		t.Fatal(err)
	}
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	if string(left) != string(right) {
		t.Fatal("mapping keys/raw data/destinations changed", string(after))
	}
}

func TestSessionRecencyUpgradeRejectsUntrackedInternalColumns(t *testing.T) {
	for _, column := range []string{"source_timezone text", "session_last_ran_at timestamptz"} {
		t.Run(column, func(t *testing.T) {
			s, ctx := testLegacyService(t)
			installMigrationVersions(t, s, ctx, "001_exercise_centric.sql", "002_porting.sql", "003_exercise_rpe.sql", "004_notes.sql")
			execSQL(t, s, ctx, "ALTER TABLE internal.porting_map ADD COLUMN "+column)
			status, err := migrations.Inspect(ctx, s.Pool)
			if err != nil || status.CanApply || status.Ready || !strings.Contains(strings.Join(status.Issues, " "), "Untracked column already exists: internal.porting_map.") {
				t.Fatal(status, err)
			}
		})
	}
}
