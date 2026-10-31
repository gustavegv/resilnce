//go:build integration

package porting

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gustavegv/resilnce/apps/server/migrations"
)

func schemaRequest(h HTTP, method, path string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "/internal/porting/"+path, strings.NewReader(`{}`))
	r.Header.Set("Origin", "https://app.example")
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func schemaHTTP(s *Service) HTTP {
	return HTTP{Service: s, Origin: "https://app.example", Authenticate: func(*http.Request) (string, error) { return "gustavengvall@yahoo.se", nil }}
}

func installMigrationVersions(t *testing.T, s *Service, ctx context.Context, names ...string) {
	t.Helper()
	execSQL(t, s, ctx, `CREATE SCHEMA internal;
CREATE TABLE internal.schema_migration(version text PRIMARY KEY,checksum text NOT NULL,applied_at timestamptz NOT NULL DEFAULT now());`)
	for _, name := range names {
		sql, err := os.ReadFile("../migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		execSQL(t, s, ctx, string(sql))
		if _, err = s.Pool.Exec(ctx, `INSERT INTO internal.schema_migration(version,checksum,applied_at) VALUES($1,$2,'2000-01-01T00:00:00Z')`, name, fmt.Sprintf("%x", sha256.Sum256(sql))); err != nil {
			t.Fatal(err)
		}
	}
}

func TestToolBootstrapsSchemaFromLegacyDatabase(t *testing.T) {
	s, ctx := testLegacyService(t)
	execSQL(t, s, ctx, singleOwnerFixture)
	s.AllowApply = false // Data import policy does not disable schema setup.
	h := schemaHTTP(s)
	before, err := s.readSource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	original, _ := json.Marshal(before)
	w := schemaRequest(h, "GET", "overview")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var overview struct {
		Schema    migrations.Status `json:"schema"`
		Inventory []Inventory       `json:"inventory"`
		Runs      []Run             `json:"runs"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &overview); err != nil {
		t.Fatal(err)
	}
	if overview.Schema.Ready || !overview.Schema.CanApply || len(overview.Schema.Migrations) != 4 || len(overview.Inventory) != 0 || len(overview.Runs) != 0 {
		t.Fatal(w.Body.String())
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM pg_namespace WHERE nspname='internal'`) != 0 {
		t.Fatal("status check wrote bookkeeping")
	}
	w = schemaRequest(h, "POST", "audits")
	if w.Code != 409 {
		t.Fatalf("audit accepted before setup: %d %s", w.Code, w.Body.String())
	}
	// Two simultaneous clicks serialize and finish with the same schema.
	responses := make(chan *httptest.ResponseRecorder, 2)
	for range 2 {
		go func() { responses <- schemaRequest(h, "POST", "schema/apply") }()
	}
	for range 2 {
		w = <-responses
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		var status migrations.Status
		if err = json.Unmarshal(w.Body.Bytes(), &status); err != nil {
			t.Fatal(err)
		}
		if !status.Ready || status.CanApply || len(status.Issues) != 0 {
			t.Fatal(w.Body.String())
		}
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM internal.schema_migration`) != 4 || scalar(t, s, ctx, `SELECT count(*) FROM public.exercise`) != 0 {
		t.Fatal("setup ran data imports or duplicated migrations")
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM internal.porting_run`) != 0 {
		t.Fatal("setup depends on a queued worker job")
	}
	after, err := s.readSource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	current, _ := json.Marshal(after)
	if string(original) != string(current) {
		t.Fatal("schema setup changed legacy data")
	}
	w = schemaRequest(h, "GET", "overview")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if err = json.Unmarshal(w.Body.Bytes(), &overview); err != nil {
		t.Fatal(err)
	}
	if !overview.Schema.Ready || len(overview.Inventory) == 0 {
		t.Fatal(w.Body.String())
	}
}

func TestSchemaCheckDetectsDriftAndRefusesDestructiveRepair(t *testing.T) {
	for _, tc := range []struct{ name, sql, want string }{
		{"missing column", `ALTER TABLE public.exercise DROP COLUMN name`, "public.exercise.name"},
		{"missing RPE", `ALTER TABLE public.workout_exercise DROP COLUMN recorded_rpe`, "public.workout_exercise.recorded_rpe"},
		{"missing last RPE", `ALTER TABLE public.exercise_variant DROP COLUMN last_rpe`, "public.exercise_variant.last_rpe"},
		{"missing RPE timestamp", `ALTER TABLE public.exercise_variant DROP COLUMN last_rpe_at`, "public.exercise_variant.last_rpe_at"},
		{"missing variant notes", `ALTER TABLE public.exercise_variant DROP COLUMN notes`, "public.exercise_variant.notes"},
		{"missing workout notes", `ALTER TABLE public.workout DROP COLUMN notes`, "public.workout.notes"},
		{"missing attempt notes", `ALTER TABLE public.workout_exercise DROP COLUMN notes`, "public.workout_exercise.notes"},
		{"wrong notes type", `ALTER TABLE public.workout ALTER COLUMN notes TYPE varchar(4000)`, "public.workout.notes"},
		{"wrong notes nullability", `ALTER TABLE public.workout ALTER COLUMN notes SET NOT NULL`, "public.workout.notes"},
		{"missing notes check", `ALTER TABLE public.workout DROP CONSTRAINT workout_notes_length`, "public.workout.workout_notes_length"},
		{"unvalidated notes check", `ALTER TABLE public.workout DROP CONSTRAINT workout_notes_length; ALTER TABLE public.workout ADD CONSTRAINT workout_notes_length CHECK(char_length(notes)<=4000) NOT VALID`, "public.workout.workout_notes_length"},
		{"altered notes check", `ALTER TABLE public.workout DROP CONSTRAINT workout_notes_length; ALTER TABLE public.workout ADD CONSTRAINT workout_notes_length CHECK(char_length(notes)<=9000)`, "public.workout.workout_notes_length"},
		{"missing RPE check", `ALTER TABLE public.workout_exercise DROP CONSTRAINT workout_exercise_rpe_range`, "public.workout_exercise.workout_exercise_rpe_range"},
		{"altered RPE check", `ALTER TABLE public.workout_exercise DROP CONSTRAINT workout_exercise_rpe_range; ALTER TABLE public.workout_exercise ADD CONSTRAINT workout_exercise_rpe_range CHECK(recorded_rpe BETWEEN 0 AND 20)`, "public.workout_exercise.workout_exercise_rpe_range"},
		{"missing RPE timestamp check", `ALTER TABLE public.exercise_variant DROP CONSTRAINT exercise_variant_rpe_timestamp`, "public.exercise_variant.exercise_variant_rpe_timestamp"},
		{"wrong precision", `ALTER TABLE public.variant_target_set ALTER COLUMN target_weight TYPE numeric(9,3)`, "public.variant_target_set.target_weight"},
		{"missing table", `DROP TABLE internal.porting_issue`, "Missing table internal.porting_issue"},
		{"checksum", `UPDATE internal.schema_migration SET checksum='changed' WHERE version='001_exercise_centric.sql'`, "checksum differs"},
		{"row security", `ALTER TABLE public.exercise DISABLE ROW LEVEL SECURITY`, "Row level security is disabled"},
		{"missing index", `DROP INDEX public.exercise_owner_name`, "Missing or invalid index"},
		{"disabled trigger", `ALTER TABLE public.exercise_variant DISABLE TRIGGER variant_requires_targets`, "Missing or disabled trigger"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, ctx := testService(t)
			execSQL(t, s, ctx, tc.sql)
			h := schemaHTTP(s)
			w := schemaRequest(h, "GET", "schema")
			if w.Code != 200 {
				t.Fatal(w.Code, w.Body.String())
			}
			var status migrations.Status
			if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
				t.Fatal(err)
			}
			if status.Ready || status.CanApply || !strings.Contains(w.Body.String(), tc.want) {
				t.Fatal(w.Body.String())
			}
			w = schemaRequest(h, "POST", "schema/apply")
			if w.Code != 409 {
				t.Fatalf("unsafe repair accepted: %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestSchemaSetupRequiresCompatibleSourceAndUnoccupiedDestinations(t *testing.T) {
	for _, tc := range []struct{ name, sql, want string }{
		{"source column missing", `ALTER TABLE public."Exercise" DROP COLUMN auto_inc`, "public.Exercise.auto_inc"},
		{"untracked destination", `CREATE TABLE public.exercise(id uuid)`, "Untracked table already exists: public.exercise"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, ctx := testLegacyService(t)
			execSQL(t, s, ctx, tc.sql)
			h := schemaHTTP(s)
			w := schemaRequest(h, "GET", "overview")
			if w.Code != 200 || !strings.Contains(w.Body.String(), tc.want) {
				t.Fatal(w.Code, w.Body.String())
			}
			w = schemaRequest(h, "POST", "schema/apply")
			if w.Code != 409 {
				t.Fatal(w.Code, w.Body.String())
			}
			if scalar(t, s, ctx, `SELECT count(*) FROM pg_namespace WHERE nspname='internal'`) != 0 {
				t.Fatal("blocked setup left a partial migration")
			}
		})
	}
}

func TestToolAppliesOnlyPendingMigrations(t *testing.T) {
	s, ctx := testLegacyService(t)
	first, err := os.ReadFile("../migrations/001_exercise_centric.sql")
	if err != nil {
		t.Fatal(err)
	}
	execSQL(t, s, ctx, `CREATE SCHEMA internal; CREATE TABLE internal.schema_migration(version text PRIMARY KEY,checksum text NOT NULL,applied_at timestamptz NOT NULL DEFAULT now());`)
	execSQL(t, s, ctx, string(first))
	if _, err = s.Pool.Exec(ctx, `INSERT INTO internal.schema_migration VALUES('001_exercise_centric.sql',$1,'2000-01-01T00:00:00Z')`, fmt.Sprintf("%x", sha256.Sum256(first))); err != nil {
		t.Fatal(err)
	}
	h := schemaHTTP(s)
	w := schemaRequest(h, "GET", "schema")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var status migrations.Status
	if err = json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.Ready || !status.CanApply || !status.Migrations[0].Applied || status.Migrations[1].Applied {
		t.Fatal(w.Body.String())
	}
	w = schemaRequest(h, "POST", "schema/apply")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM internal.schema_migration WHERE version='001_exercise_centric.sql' AND applied_at='2000-01-01T00:00:00Z'`) != 1 {
		t.Fatal("existing migration was reapplied")
	}
}

func TestSchemaSetupRollsBackOnDatabaseFailure(t *testing.T) {
	s, ctx := testLegacyService(t)
	execSQL(t, s, ctx, `CREATE FUNCTION public.fail_porting_ddl() RETURNS event_trigger LANGUAGE plpgsql AS $$
 BEGIN IF EXISTS(SELECT 1 FROM pg_event_trigger_ddl_commands() WHERE object_identity='public.exercise_variant') THEN RAISE EXCEPTION 'injected DDL failure'; END IF; END $$;
 CREATE EVENT TRIGGER fail_porting_ddl ON ddl_command_end EXECUTE FUNCTION public.fail_porting_ddl();`)
	h := schemaHTTP(s)
	w := schemaRequest(h, "POST", "schema/apply")
	if w.Code != 500 {
		t.Fatal(w.Code, w.Body.String())
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='exercise'`) != 0 || scalar(t, s, ctx, `SELECT count(*) FROM pg_namespace WHERE nspname='internal'`) != 0 {
		t.Fatal("failed setup left a partially applied layout")
	}
	execSQL(t, s, ctx, `DROP EVENT TRIGGER fail_porting_ddl;`)
	w = schemaRequest(h, "POST", "schema/apply")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
}
