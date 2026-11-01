package training

import (
	"context"
	"strconv"
)

// CompleteAttempt freezes actual performance and progresses current targets in
// one owner-serialized transaction. No actual values are supplied by completion.
type CompleteAttempt struct {
	OperationID             string `json:"operationId"`
	WorkoutID               string `json:"workoutId"`
	AttemptID               string `json:"attemptId"`
	ExerciseID              string `json:"exerciseId"`
	VariantID               string `json:"variantId"`
	ExpectedWorkoutRevision string `json:"expectedWorkoutRevision"`
	ExpectedVariantRevision string `json:"expectedVariantRevision"`
}

func (s *Service) CompleteAttempt(ctx context.Context, owner string, c CompleteAttempt) (out WorkoutResult, err error) {
	defer func() { err = storageError(err) }()
	command, err := s.beginCommand(ctx, owner, c.OperationID, "complete-attempt", c.WorkoutID, c)
	if err != nil {
		return out, err
	}
	defer command.rollback(ctx)
	tx := command.tx
	finish := func() (WorkoutResult, error) {
		w, e := readWorkout(ctx, tx, owner, command.targetID)
		if e != nil {
			return out, e
		}
		if e = command.commit(ctx, w.ID, w.Revision); e != nil {
			return out, e
		}
		return WorkoutResult{Operation: command.operation, Workout: w}, nil
	}
	if command.replay {
		return finish()
	}
	for field, value := range map[string]string{"workoutId": c.WorkoutID, "attemptId": c.AttemptID, "exerciseId": c.ExerciseID, "variantId": c.VariantID} {
		if !validID(value) {
			return out, invalid(field, "Use the exact UUID identity")
		}
	}
	for field, value := range map[string]string{"expectedWorkoutRevision": c.ExpectedWorkoutRevision, "expectedVariantRevision": c.ExpectedVariantRevision} {
		n, e := strconv.ParseInt(value, 10, 64)
		if e != nil || n < 1 || strconv.FormatInt(n, 10) != value {
			return out, invalid(field, "Use the reviewed positive revision string")
		}
	}
	var status, wr, state, configRevision, vr string
	if err = tx.QueryRow(ctx, `SELECT status,revision::text FROM public.workout WHERE owner_mail=$1 AND id=$2 FOR UPDATE`, owner, c.WorkoutID).Scan(&status, &wr); err != nil {
		return out, err
	}
	// Authorize every supplied parent before returning a completed acknowledgment.
	if err = tx.QueryRow(ctx, `SELECT a.state,a.config_revision::text FROM public.workout_exercise a JOIN public.exercise_variant v ON (v.owner_mail,v.id)=(a.owner_mail,a.variant_id) WHERE a.owner_mail=$1 AND a.workout_id=$2 AND a.id=$3 AND a.variant_id=$4 AND v.exercise_id=$5`, owner, c.WorkoutID, c.AttemptID, c.VariantID, c.ExerciseID).Scan(&state, &configRevision); err != nil {
		return out, err
	}
	var archivedExercise, archivedVariant bool
	if err = tx.QueryRow(ctx, `SELECT archived_at IS NOT NULL FROM public.exercise WHERE owner_mail=$1 AND id=$2 FOR UPDATE`, owner, c.ExerciseID).Scan(&archivedExercise); err != nil {
		return out, err
	}
	if err = tx.QueryRow(ctx, `SELECT revision::text,archived_at IS NOT NULL FROM public.exercise_variant WHERE owner_mail=$1 AND id=$2 AND exercise_id=$3 FOR UPDATE`, owner, c.VariantID, c.ExerciseID).Scan(&vr, &archivedVariant); err != nil {
		return out, err
	}
	// Finality is independent of receipt identity: a new operation ID can safely
	// acknowledge this frozen completion even after another workout progressed.
	if state == "completed" {
		return finish()
	}
	if wr != c.ExpectedWorkoutRevision || vr != c.ExpectedVariantRevision || configRevision != vr {
		return out, conflict("stale-revision", "Workout or configuration changed. Review current state before finishing")
	}
	if status != "active" || state != "draft" {
		return out, conflict("resource-finalized", "This exercise or workout is finalized")
	}
	if archivedExercise || archivedVariant {
		return out, conflict("resource-unavailable", "This exercise or variant is archived")
	}
	w, err := readWorkout(ctx, tx, owner, c.WorkoutID)
	if err != nil {
		return out, err
	}
	var attempt WorkoutExercise
	for _, a := range w.Exercises {
		if a.ID == c.AttemptID {
			attempt = a
			break
		}
	}
	next, err := NextTargets(attempt.CurrentVariant, attempt.Sets)
	if err != nil {
		return out, err
	}
	for _, target := range next {
		result, e := tx.Exec(ctx, `UPDATE public.variant_target_set SET target_reps=$4,target_weight=$5 WHERE owner_mail=$1 AND variant_id=$2 AND id=$3`, owner, c.VariantID, target.ID, target.TargetReps, target.TargetWeight)
		if e != nil {
			return out, e
		}
		if result.RowsAffected() != 1 {
			return out, conflict("stale-revision", "Prescribed targets changed. Review current configuration")
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE public.exercise_variant SET revision=revision+1 WHERE owner_mail=$1 AND id=$2`, owner, c.VariantID); err != nil {
		return out, err
	}
	if _, err = tx.Exec(ctx, `UPDATE public.workout_exercise SET state='completed',completed_at=clock_timestamp() WHERE owner_mail=$1 AND id=$2`, owner, c.AttemptID); err != nil {
		return out, err
	}
	if _, err = tx.Exec(ctx, `UPDATE public.workout w SET revision=revision+1,status=CASE WHEN EXISTS(SELECT 1 FROM public.workout_exercise a WHERE a.owner_mail=w.owner_mail AND a.workout_id=w.id AND a.state='draft') THEN 'active' ELSE 'ended' END,ended_at=CASE WHEN EXISTS(SELECT 1 FROM public.workout_exercise a WHERE a.owner_mail=w.owner_mail AND a.workout_id=w.id AND a.state='draft') THEN NULL ELSE clock_timestamp() END WHERE w.owner_mail=$1 AND w.id=$2`, owner, c.WorkoutID); err != nil {
		return out, err
	}
	return finish()
}
