package training

import (
	"context"
	"strconv"
)

// QuitWorkout ends one reviewed workout, retaining only completed attempts.
// Receipt and owner serialization are shared with recording and completion.
type QuitWorkout struct {
	OperationID             string `json:"operationId"`
	WorkoutID               string `json:"workoutId"`
	ExpectedWorkoutRevision string `json:"expectedWorkoutRevision"`
}

func (s *Service) QuitWorkout(ctx context.Context, owner string, c QuitWorkout) (out WorkoutResult, err error) {
	defer func() { err = storageError(err) }()
	command, err := s.beginCommand(ctx, owner, c.OperationID, "quit-workout", c.WorkoutID, c)
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
	if !validID(c.WorkoutID) {
		return out, invalid("workoutId", "Use the exact workout UUID identity")
	}
	n, e := strconv.ParseInt(c.ExpectedWorkoutRevision, 10, 64)
	if e != nil || n < 1 || strconv.FormatInt(n, 10) != c.ExpectedWorkoutRevision {
		return out, invalid("expectedWorkoutRevision", "Use the reviewed positive revision string")
	}
	var status, revision string
	if err = tx.QueryRow(ctx, `SELECT status,revision::text FROM public.workout WHERE owner_mail=$1 AND id=$2 FOR UPDATE`, owner, c.WorkoutID).Scan(&status, &revision); err != nil {
		return out, err
	}
	// Repeated quit is final even with a new operation ID or after a later start.
	if status == "ended" {
		return finish()
	}
	if revision != c.ExpectedWorkoutRevision {
		return out, conflict("stale-revision", "Workout changed. Review completed and unfinished work before quitting")
	}
	if _, err = tx.Exec(ctx, `DELETE FROM public.workout_exercise WHERE owner_mail=$1 AND workout_id=$2 AND state='draft'`, owner, c.WorkoutID); err != nil {
		return out, err
	}
	if _, err = tx.Exec(ctx, `UPDATE public.workout SET status='ended',ended_at=clock_timestamp(),revision=revision+1 WHERE owner_mail=$1 AND id=$2`, owner, c.WorkoutID); err != nil {
		return out, err
	}
	return finish()
}
