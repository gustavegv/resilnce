//go:build integration

package porting

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/gustavegv/resilnce/apps/server/migrations"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestNotesMigrationUpgradesPopulatedDatabase(t *testing.T) {
	s, ctx := testLegacyService(t)
	installMigrationVersions(t, s, ctx, "001_exercise_centric.sql", "002_porting.sql", "003_exercise_rpe.sql")
	execSQL(t, s, ctx, completedAttemptFixture+`
UPDATE public.workout_exercise SET recorded_rpe=8;
UPDATE public.exercise_variant SET last_rpe=8,last_rpe_at='2026-01-15T13:00:00Z';`)
	before, err := s.readSource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	status, err := migrations.Inspect(ctx, s.Pool)
	if err != nil || status.Ready || !status.CanApply || len(status.Issues) != 0 {
		t.Fatalf("notes upgrade should be applicable: %+v, %v", status, err)
	}
	for range 2 {
		if err = migrations.Apply(ctx, s.Pool); err != nil {
			t.Fatal(err)
		}
	}
	status, err = migrations.Inspect(ctx, s.Pool)
	if err != nil || !status.Ready {
		t.Fatalf("notes upgrade not ready: %+v, %v", status, err)
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM internal.schema_migration WHERE version<>'004_notes.sql' AND applied_at='2000-01-01T00:00:00Z'`) != 3 ||
		scalar(t, s, ctx, `SELECT count(*) FROM public.exercise_variant WHERE notes IS NULL AND last_rpe=8 AND last_rpe_at='2026-01-15T13:00:00Z' AND revision=1`) != 1 ||
		scalar(t, s, ctx, `SELECT count(*) FROM public.workout_exercise WHERE notes IS NULL AND recorded_rpe=8 AND state='completed'`) != 1 ||
		scalar(t, s, ctx, `SELECT count(*) FROM public.workout WHERE notes IS NULL AND status='ended'`) != 1 ||
		scalar(t, s, ctx, `SELECT count(*) FROM public.variant_target_set WHERE target_reps=12 AND target_weight=50`) != 1 ||
		scalar(t, s, ctx, `SELECT count(*) FROM public.workout_set WHERE recorded_reps=11 AND recorded_weight=50`) != 1 {
		t.Fatal("notes upgrade changed existing migrations, RPE, targets, or performance")
	}
	after, err := s.readSource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	oldJSON, _ := json.Marshal(before)
	newJSON, _ := json.Marshal(after)
	if string(oldJSON) != string(newJSON) {
		t.Fatal("notes upgrade changed legacy data")
	}
	for _, table := range []string{"exercise_variant", "workout", "workout_exercise"} {
		t.Run(table, func(t *testing.T) {
			query := "UPDATE public." + table + " SET notes=$1"
			for _, note := range []any{"First line\nSecond line: åäö", strings.Repeat("å", 4000), nil} {
				if _, err := s.Pool.Exec(ctx, query, note); err != nil {
					t.Fatal(err)
				}
				var stored *string
				if err := s.Pool.QueryRow(ctx, "SELECT notes FROM public."+table).Scan(&stored); err != nil {
					t.Fatal(err)
				}
				if note == nil {
					if stored != nil {
						t.Fatal("null note was not preserved")
					}
				} else if stored == nil || *stored != note.(string) {
					t.Fatal("note did not round-trip exactly")
				}
			}
			_, err := s.Pool.Exec(ctx, query, strings.Repeat("å", 4001))
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
				t.Fatalf("expected notes length constraint violation, got %v", err)
			}
		})
	}
}

func TestImportAppliesMigrationsAndPreservesUserNotesOnRetry(t *testing.T) {
	s, ctx := testLegacyService(t)
	execSQL(t, s, ctx, singleOwnerFixture)
	// API and worker share the default-enabled policy, with no enable flag needed.
	t.Setenv("PORTING_APPLY_ENABLED", "")
	s.AllowApply = ApplyEnabled()
	first := applyRun(t, s, ctx, "a@example.com")
	if first.Status != "completed" || scalar(t, s, ctx, `SELECT count(*) FROM internal.schema_migration`) != 6 ||
		scalar(t, s, ctx, `SELECT count(*) FROM public.exercise_variant WHERE notes IS NULL AND last_rpe IS NULL AND last_rpe_at IS NULL`) != 1 ||
		scalar(t, s, ctx, `SELECT count(*) FROM public.legacy_history_entry`) != 1 ||
		scalar(t, s, ctx, `SELECT count(*) FROM public.workout`) != 0 {
		t.Fatal("import did not apply the full layout or preserve legacy history", first)
	}
	execSQL(t, s, ctx, `UPDATE public.exercise_variant SET notes='User instructions';`)
	second := applyRun(t, s, ctx, "a@example.com")
	if second.Status != "completed" || second.Owners[0].Created != 0 ||
		scalar(t, s, ctx, `SELECT count(*) FROM public.exercise_variant WHERE notes='User instructions'`) != 1 {
		t.Fatal("retry overwrote user notes or duplicated imported data", second)
	}
}

func TestImportSchemaPreflightRejectsDriftBeforeQueueingOrWriting(t *testing.T) {
	s, ctx := testService(t)
	execSQL(t, s, ctx, singleOwnerFixture)
	rows, err := s.readSource(ctx)
	if err != nil {
		t.Fatal(err)
	}
	execSQL(t, s, ctx, `ALTER TABLE public.workout DROP CONSTRAINT workout_notes_length;`)
	_, err = s.StartRun(ctx, "operator", "apply", "a@example.com", Options{})
	var layoutErr *migrations.LayoutError
	if !errors.As(err, &layoutErr) {
		t.Fatalf("expected layout error before enqueue, got %v", err)
	}
	_, err = s.ImportOwner(ctx, rows, "a@example.com", Options{})
	if !errors.As(err, &layoutErr) {
		t.Fatalf("expected layout error before account writes, got %v", err)
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM internal.porting_run`) != 0 ||
		scalar(t, s, ctx, `SELECT count(*) FROM public.exercise`) != 0 ||
		scalar(t, s, ctx, `SELECT count(*) FROM internal.porting_map`) != 0 {
		t.Fatal("schema drift was detected after queueing or importing data")
	}
}
