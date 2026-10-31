//go:build integration

package porting

import (
	"errors"
	"testing"

	"github.com/gustavegv/resilnce/apps/server/migrations"
	"github.com/jackc/pgx/v5/pgconn"
)

const completedAttemptFixture = singleOwnerFixture + `
INSERT INTO public.exercise(id,owner_mail,name) VALUES('00000000-0000-0000-0000-000000000001','a@example.com','Bench');
INSERT INTO public.exercise_variant(id,owner_mail,exercise_id,name,progression_mode,rep_threshold,auto_increment,rest_seconds)
 VALUES('00000000-0000-0000-0000-000000000002','a@example.com','00000000-0000-0000-0000-000000000001','Standard','average',12,2.5,0);
INSERT INTO public.variant_target_set(id,owner_mail,variant_id,position,target_reps,target_weight)
 VALUES('00000000-0000-0000-0000-000000000003','a@example.com','00000000-0000-0000-0000-000000000002',1,12,50);
INSERT INTO public.workout(id,owner_mail,origin,status,started_at,ended_at)
 VALUES('00000000-0000-0000-0000-000000000004','a@example.com','free','ended','2026-01-15T12:00:00Z','2026-01-15T13:00:00Z');
INSERT INTO public.workout_exercise(id,owner_mail,workout_id,variant_id,position,state,origin,progression_mode,rep_threshold,auto_increment,config_revision,completed_at)
 VALUES('00000000-0000-0000-0000-000000000005','a@example.com','00000000-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000002',1,'completed','added','average',12,2.5,1,'2026-01-15T13:00:00Z');
INSERT INTO public.workout_set(id,owner_mail,workout_exercise_id,target_set_id,position,kind,state,target_reps,target_weight,recorded_reps,recorded_weight,recorded_at)
 VALUES('00000000-0000-0000-0000-000000000006','a@example.com','00000000-0000-0000-0000-000000000005','00000000-0000-0000-0000-000000000003',1,'prescribed','performed',12,50,11,50,'2026-01-15T12:30:00Z');`

func TestRPEMigrationUpgradesExistingData(t *testing.T) {
	s, ctx := testLegacyService(t)
	installMigrationVersions(t, s, ctx, "001_exercise_centric.sql", "002_porting.sql")
	execSQL(t, s, ctx, completedAttemptFixture)
	status, err := migrations.Inspect(ctx, s.Pool)
	if err != nil || status.Ready || !status.CanApply || len(status.Issues) != 0 {
		t.Fatalf("RPE upgrade should be applicable: %+v, %v", status, err)
	}
	for range 2 {
		if err = migrations.Apply(ctx, s.Pool); err != nil {
			t.Fatal(err)
		}
	}
	status, err = migrations.Inspect(ctx, s.Pool)
	if err != nil || !status.Ready {
		t.Fatalf("RPE upgrade not ready: %+v, %v", status, err)
	}
	if scalar(t, s, ctx, `SELECT count(*) FROM public.exercise_variant WHERE last_rpe IS NULL AND last_rpe_at IS NULL`) != 1 ||
		scalar(t, s, ctx, `SELECT count(*) FROM public.workout_exercise WHERE recorded_rpe IS NULL AND state='completed' AND completed_at='2026-01-15T13:00:00Z'`) != 1 ||
		scalar(t, s, ctx, `SELECT count(*) FROM public.variant_target_set WHERE target_weight=50 AND target_reps=12`) != 1 ||
		scalar(t, s, ctx, `SELECT count(*) FROM public.workout_set WHERE recorded_weight=50 AND recorded_reps=11`) != 1 {
		t.Fatal("upgrade invented RPE or changed existing targets/performance")
	}
	// These writes verify storage constraints only. Application-level completion
	// immutability and cache maintenance belong to the future workout operations.
	for _, rating := range []string{"0", "8", "10", "NULL"} {
		execSQL(t, s, ctx, "UPDATE public.workout_exercise SET recorded_rpe="+rating)
		stamp := "'2026-01-15T13:00:00Z'"
		if rating == "NULL" {
			stamp = "NULL"
		}
		execSQL(t, s, ctx, "UPDATE public.exercise_variant SET last_rpe="+rating+",last_rpe_at="+stamp)
		if scalar(t, s, ctx, "SELECT count(*) FROM public.workout_exercise WHERE recorded_rpe IS NOT DISTINCT FROM "+rating) != 1 ||
			scalar(t, s, ctx, "SELECT count(*) FROM public.exercise_variant WHERE last_rpe IS NOT DISTINCT FROM "+rating) != 1 {
			t.Fatal("rating was not retained")
		}
	}
	for _, sql := range []string{
		`UPDATE public.workout_exercise SET recorded_rpe=-1`,
		`UPDATE public.workout_exercise SET recorded_rpe=11`,
		`UPDATE public.exercise_variant SET last_rpe=-1,last_rpe_at=now()`,
		`UPDATE public.exercise_variant SET last_rpe=11,last_rpe_at=now()`,
		`UPDATE public.exercise_variant SET last_rpe=8,last_rpe_at=NULL`,
		`UPDATE public.exercise_variant SET last_rpe=NULL,last_rpe_at=now()`,
	} {
		_, err = s.Pool.Exec(ctx, sql)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
			t.Fatalf("expected RPE check constraint violation, got %v for %s", err, sql)
		}
	}
}
