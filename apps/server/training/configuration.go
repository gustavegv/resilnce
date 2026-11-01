package training

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"
)

// ConfigurationPatch is an edit, not the creation-only Configuration. Omitted
// fields retain current values; initialWeight is deliberately not an edit.
type ConfigurationPatch struct {
	ExerciseName         *string `json:"exerciseName,omitempty"`
	VariantName          *string `json:"variantName,omitempty"`
	RepThreshold         *int    `json:"repThreshold,omitempty"`
	PrescribedSets       *int    `json:"prescribedSets,omitempty"`
	AutoIncrement        *string `json:"autoIncrement,omitempty"`
	RestSeconds          *int    `json:"restSeconds,omitempty"`
	ApplyWeightToAllSets *string `json:"applyWeightToAllSets,omitempty"`
	fields               map[string]json.RawMessage
}

func (p *ConfigurationPatch) UnmarshalJSON(raw []byte) error {
	type value ConfigurationPatch
	var v value
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&v); err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &v.fields); err != nil {
		return err
	}
	*p = ConfigurationPatch(v)
	return nil
}

type UpdateConfiguration struct {
	OperationID              string             `json:"operationId"`
	WorkoutID                string             `json:"workoutId"`
	AttemptID                string             `json:"attemptId"`
	ExerciseID               string             `json:"exerciseId"`
	VariantID                string             `json:"variantId"`
	ExpectedWorkoutRevision  string             `json:"expectedWorkoutRevision"`
	ExpectedExerciseRevision string             `json:"expectedExerciseRevision"`
	ExpectedVariantRevision  string             `json:"expectedVariantRevision"`
	Patch                    ConfigurationPatch `json:"patch"`
}
type ConfigurationResult struct {
	Operation Acknowledgment `json:"operation"`
	Workout   Workout        `json:"workout"`
	Exercise  Exercise       `json:"exercise"`
	Variant   Variant        `json:"variant"`
}

func (s *Service) UpdateConfiguration(ctx context.Context, owner string, c UpdateConfiguration) (out ConfigurationResult, err error) {
	defer func() { err = storageError(err) }()
	payload := struct {
		Intent   UpdateConfiguration
		Presence map[string]json.RawMessage
	}{c, c.Patch.fields}
	command, err := s.beginCommand(ctx, owner, c.OperationID, "update-configuration", c.WorkoutID, payload)
	if err != nil {
		return out, err
	}
	defer command.rollback(ctx)
	tx := command.tx
	finish := func() (ConfigurationResult, error) {
		w, e := readWorkout(ctx, tx, owner, command.targetID)
		if e != nil {
			return out, e
		}
		selection, e := readExercise(ctx, tx, owner, c.ExerciseID)
		if e != nil {
			return out, e
		}
		var v Variant
		for _, item := range selection.Variants {
			if item.ID == c.VariantID {
				v = item
				break
			}
		}
		if e = command.commit(ctx, w.ID, w.Revision); e != nil {
			return out, e
		}
		return ConfigurationResult{Operation: command.operation, Workout: w, Exercise: selection.Exercise, Variant: v}, nil
	}
	if command.replay {
		return finish()
	}
	for field, value := range map[string]string{"workoutId": c.WorkoutID, "attemptId": c.AttemptID, "exerciseId": c.ExerciseID, "variantId": c.VariantID} {
		if !validID(value) {
			return out, invalid(field, "Use the exact UUID identity")
		}
	}
	for field, value := range map[string]string{"expectedWorkoutRevision": c.ExpectedWorkoutRevision, "expectedExerciseRevision": c.ExpectedExerciseRevision, "expectedVariantRevision": c.ExpectedVariantRevision} {
		n, e := strconv.ParseInt(value, 10, 64)
		if e != nil || n < 1 || strconv.FormatInt(n, 10) != value {
			return out, invalid(field, "Use the reviewed positive revision string")
		}
	}
	p := c.Patch
	for field, value := range p.fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return out, invalid("patch."+field, "Supply a value or omit the field")
		}
	}
	if p.ExerciseName != nil {
		if e := validateName(*p.ExerciseName, "exerciseName"); e != nil {
			return out, e
		}
	}
	if p.VariantName != nil {
		if e := validateName(*p.VariantName, "variantName"); e != nil {
			return out, e
		}
	}
	if p.RepThreshold != nil && (*p.RepThreshold < 1 || *p.RepThreshold > 99) {
		return out, invalid("repThreshold", "Use a threshold of 1–99")
	}
	if p.PrescribedSets != nil && (*p.PrescribedSets < 1 || *p.PrescribedSets > 20) {
		return out, invalid("prescribedSets", "Use 1–20 prescribed sets")
	}
	if p.AutoIncrement != nil {
		if _, e := parseCents(*p.AutoIncrement, 25, 9900); e != nil {
			return out, invalid("autoIncrement", e.Error())
		}
	}
	if p.RestSeconds != nil && (*p.RestSeconds < 0 || int64(*p.RestSeconds) > 2147483647) {
		return out, invalid("restSeconds", "Use nonnegative whole rest seconds within storage range")
	}
	if p.ApplyWeightToAllSets != nil {
		if _, e := parseCents(*p.ApplyWeightToAllSets, 0, 999900); e != nil {
			return out, invalid("applyWeightToAllSets", e.Error())
		}
	}
	if p.ExerciseName == nil && p.VariantName == nil && p.RepThreshold == nil && p.PrescribedSets == nil && p.AutoIncrement == nil && p.RestSeconds == nil && p.ApplyWeightToAllSets == nil {
		return out, invalid("patch", "Supply an explicit configuration change")
	}
	var status, wr, attemptState, er, vr string
	var archivedExercise, archivedVariant bool
	if err = tx.QueryRow(ctx, `SELECT status,revision::text FROM public.workout WHERE owner_mail=$1 AND id=$2 FOR UPDATE`, owner, c.WorkoutID).Scan(&status, &wr); err != nil {
		return out, err
	}
	if err = tx.QueryRow(ctx, `SELECT a.state FROM public.workout_exercise a JOIN public.exercise_variant v ON (v.owner_mail,v.id)=(a.owner_mail,a.variant_id) WHERE a.owner_mail=$1 AND a.workout_id=$2 AND a.id=$3 AND a.variant_id=$4 AND v.exercise_id=$5`, owner, c.WorkoutID, c.AttemptID, c.VariantID, c.ExerciseID).Scan(&attemptState); err != nil {
		return out, err
	}
	if err = tx.QueryRow(ctx, `SELECT revision::text,archived_at IS NOT NULL FROM public.exercise WHERE owner_mail=$1 AND id=$2 FOR UPDATE`, owner, c.ExerciseID).Scan(&er, &archivedExercise); err != nil {
		return out, err
	}
	if err = tx.QueryRow(ctx, `SELECT revision::text,archived_at IS NOT NULL FROM public.exercise_variant WHERE owner_mail=$1 AND id=$2 AND exercise_id=$3 FOR UPDATE`, owner, c.VariantID, c.ExerciseID).Scan(&vr, &archivedVariant); err != nil {
		return out, err
	}
	if wr != c.ExpectedWorkoutRevision || er != c.ExpectedExerciseRevision || vr != c.ExpectedVariantRevision {
		return out, conflict("stale-revision", "Configuration or workout changed. Review current settings before reapplying this edit")
	}
	if status != "active" || attemptState != "draft" {
		return out, conflict("resource-finalized", "This exercise or workout is finalized")
	}
	if archivedExercise || archivedVariant {
		return out, conflict("resource-unavailable", "This exercise or variant is archived")
	}
	if p.ExerciseName != nil {
		if _, err = tx.Exec(ctx, `UPDATE public.exercise SET name=$3,revision=revision+1 WHERE owner_mail=$1 AND id=$2`, owner, c.ExerciseID, strings.TrimSpace(*p.ExerciseName)); err != nil {
			return out, err
		}
	}
	if p.VariantName != nil || p.RepThreshold != nil || p.PrescribedSets != nil || p.AutoIncrement != nil || p.RestSeconds != nil || p.ApplyWeightToAllSets != nil {
		var name *string
		if p.VariantName != nil {
			value := strings.TrimSpace(*p.VariantName)
			name = &value
		}
		if _, err = tx.Exec(ctx, `UPDATE public.exercise_variant SET name=COALESCE($3,name),rep_threshold=COALESCE($4,rep_threshold),auto_increment=COALESCE($5,auto_increment),rest_seconds=COALESCE($6,rest_seconds),revision=revision+1 WHERE owner_mail=$1 AND id=$2`, owner, c.VariantID, name, p.RepThreshold, p.AutoIncrement, p.RestSeconds); err != nil {
			return out, err
		}

		// Retain existing targets in order. Count growth uses the last retained
		// weight and current threshold; only the explicit all-sets intent equalizes.
		selection, e := readExercise(ctx, tx, owner, c.ExerciseID)
		if e != nil {
			return out, e
		}
		var current Variant
		for _, v := range selection.Variants {
			if v.ID == c.VariantID {
				current = v
				break
			}
		}
		count := current.PrescribedSets
		if p.PrescribedSets != nil {
			count = *p.PrescribedSets
		}
		if count < current.PrescribedSets {
			// Remove draft work before deleting provenance; completed rows are never
			// touched and their optional link is cleared by the target FK only.
			if _, err = tx.Exec(ctx, `DELETE FROM public.workout_set s USING public.workout_exercise a WHERE (a.owner_mail,a.id)=(s.owner_mail,s.workout_exercise_id) AND a.owner_mail=$1 AND a.workout_id=$2 AND a.variant_id=$3 AND a.state='draft' AND s.position>$4`, owner, c.WorkoutID, c.VariantID, count); err != nil {
				return out, err
			}
			if _, err = tx.Exec(ctx, `DELETE FROM public.variant_target_set WHERE owner_mail=$1 AND variant_id=$2 AND position>$3`, owner, c.VariantID, count); err != nil {
				return out, err
			}
		}
		if count > current.PrescribedSets {
			weight := current.TargetSets[len(current.TargetSets)-1].TargetWeight
			if p.ApplyWeightToAllSets != nil {
				weight = *p.ApplyWeightToAllSets
			}
			for n := current.PrescribedSets + 1; n <= count; n++ {
				id := newID()
				if _, err = tx.Exec(ctx, `INSERT INTO public.variant_target_set(id,owner_mail,variant_id,position,target_reps,target_weight) VALUES($1,$2,$3,$4,$5,$6)`, id, owner, c.VariantID, n, current.RepThreshold, weight); err != nil {
					return out, err
				}
				if _, err = tx.Exec(ctx, `INSERT INTO public.workout_set(id,owner_mail,workout_exercise_id,target_set_id,position,kind,state,target_reps,target_weight) SELECT gen_random_uuid(),owner_mail,id,$4,$5,'prescribed','pending',$6,$7 FROM public.workout_exercise WHERE owner_mail=$1 AND workout_id=$2 AND variant_id=$3 AND state='draft'`, owner, c.WorkoutID, c.VariantID, id, n, current.RepThreshold, weight); err != nil {
					return out, err
				}
			}
		}
		if p.ApplyWeightToAllSets != nil {
			if _, err = tx.Exec(ctx, `UPDATE public.variant_target_set SET target_weight=$3 WHERE owner_mail=$1 AND variant_id=$2`, owner, c.VariantID, *p.ApplyWeightToAllSets); err != nil {
				return out, err
			}
			if _, err = tx.Exec(ctx, `UPDATE public.workout_set s SET target_weight=$4,late_weight_override=s.late_weight_override OR (s.state='performed' AND s.target_weight<>$4::numeric) FROM public.workout_exercise a WHERE (a.owner_mail,a.id)=(s.owner_mail,s.workout_exercise_id) AND a.owner_mail=$1 AND a.workout_id=$2 AND a.variant_id=$3 AND a.state='draft' AND s.kind='prescribed'`, owner, c.WorkoutID, c.VariantID, *p.ApplyWeightToAllSets); err != nil {
				return out, err
			}
		}
		if _, err = tx.Exec(ctx, `UPDATE public.workout_exercise a SET rep_threshold=v.rep_threshold,auto_increment=v.auto_increment,rest_seconds=v.rest_seconds,config_revision=v.revision FROM public.exercise_variant v WHERE a.owner_mail=$1 AND a.workout_id=$2 AND a.variant_id=$3 AND a.state='draft' AND (v.owner_mail,v.id)=(a.owner_mail,a.variant_id)`, owner, c.WorkoutID, c.VariantID); err != nil {
			return out, err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE public.workout SET revision=revision+1 WHERE owner_mail=$1 AND id=$2`, owner, c.WorkoutID); err != nil {
		return out, err
	}
	return finish()
}
