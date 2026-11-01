package training

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"
)

// RecordSet describes one explicit state transition. Weight comes from this
// exact workout set, never from a browser-supplied target or another set.
type RecordSet struct {
	OperationID             string `json:"operationId"`
	WorkoutID               string `json:"workoutId"`
	AttemptID               string `json:"attemptId"`
	SetID                   string `json:"setId"`
	ExpectedWorkoutRevision string `json:"expectedWorkoutRevision"`
	State                   string `json:"state"`
	RecordedReps            *int   `json:"recordedReps,omitempty"`
	repsPresent             bool
}

func (c *RecordSet) UnmarshalJSON(raw []byte) error {
	type intent RecordSet
	var v intent
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&v); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	*c = RecordSet(v)
	_, c.repsPresent = fields["recordedReps"]
	return nil
}

func (s *Service) RecordSet(ctx context.Context, owner string, c RecordSet) (out WorkoutResult, err error) {
	defer func() { err = storageError(err) }()
	payload := struct {
		Intent      RecordSet
		RepsPresent bool
	}{c, c.repsPresent || c.RecordedReps != nil}
	command, err := s.beginCommand(ctx, owner, c.OperationID, "record-set", c.WorkoutID, payload)
	if err != nil {
		return out, err
	}
	defer command.rollback(ctx)
	tx := command.tx
	finish := func() (WorkoutResult, error) {
		w, err := readWorkout(ctx, tx, owner, command.targetID)
		if err != nil {
			return out, err
		}
		if err = command.commit(ctx, w.ID, w.Revision); err != nil {
			return out, err
		}
		return WorkoutResult{Operation: command.operation, Workout: w}, nil
	}
	if command.replay {
		return finish()
	}
	for _, id := range []struct{ field, value string }{{"workoutId", c.WorkoutID}, {"attemptId", c.AttemptID}, {"setId", c.SetID}} {
		if !validID(id.value) {
			return out, invalid(id.field, "Use the exact UUID identity")
		}
	}
	n, parseErr := strconv.ParseInt(c.ExpectedWorkoutRevision, 10, 64)
	if parseErr != nil || n < 1 || strconv.FormatInt(n, 10) != c.ExpectedWorkoutRevision {
		return out, invalid("expectedWorkoutRevision", "Use the reviewed positive workout revision string")
	}
	switch c.State {
	case "performed":
		if c.RecordedReps == nil || *c.RecordedReps < 0 || int64(*c.RecordedReps) > 2147483647 {
			return out, invalid("recordedReps", "Perform with explicit whole reps from 0 to 2147483647")
		}
	case "pending", "skipped":
		if c.repsPresent || c.RecordedReps != nil {
			return out, invalid("recordedReps", "Pending and skipped sets have no recorded reps; omit this field")
		}
	default:
		return out, invalid("state", "Choose performed, pending or skipped")
	}
	var status, revision string
	if err = tx.QueryRow(ctx, `SELECT status,revision::text FROM public.workout WHERE owner_mail=$1 AND id=$2 FOR UPDATE`, owner, c.WorkoutID).Scan(&status, &revision); err != nil {
		return out, err
	}
	// Authorize all nested identities before exposing aggregate conflict state.
	var attemptState string
	if err = tx.QueryRow(ctx, `SELECT a.state FROM public.workout_exercise a JOIN public.workout_set s ON (s.owner_mail,s.workout_exercise_id)=(a.owner_mail,a.id) WHERE a.owner_mail=$1 AND a.workout_id=$2 AND a.id=$3 AND s.id=$4`, owner, c.WorkoutID, c.AttemptID, c.SetID).Scan(&attemptState); err != nil {
		return out, err
	}
	if revision != c.ExpectedWorkoutRevision {
		return out, conflict("stale-revision", "The workout changed. Review current state before accepting it or explicitly reapplying your input")
	}
	if status != "active" || attemptState != "draft" {
		return out, conflict("resource-finalized", "This exercise or workout is finalized; its performance cannot be edited")
	}
	_, err = tx.Exec(ctx, `UPDATE public.workout_set SET state=$4,recorded_reps=$5,recorded_weight=CASE WHEN $4='performed' THEN CASE WHEN state='performed' THEN recorded_weight ELSE target_weight END ELSE NULL END,recorded_at=CASE WHEN $4='performed' THEN CASE WHEN state='performed' THEN recorded_at ELSE clock_timestamp() END ELSE NULL END,late_weight_override=CASE WHEN $4='performed' AND state='performed' THEN late_weight_override ELSE false END WHERE owner_mail=$1 AND workout_exercise_id=$2 AND id=$3`, owner, c.AttemptID, c.SetID, c.State, c.RecordedReps)
	if err != nil {
		return out, err
	}
	if _, err = tx.Exec(ctx, `UPDATE public.workout SET revision=revision+1 WHERE owner_mail=$1 AND id=$2`, owner, c.WorkoutID); err != nil {
		return out, err
	}
	return finish()
}
