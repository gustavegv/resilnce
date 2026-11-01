package training

import (
	"context"
	"errors"
	"sort"
	"strconv"

	"github.com/jackc/pgx/v5"
)

func readWorkout(ctx context.Context, tx pgx.Tx, owner, id string) (Workout, error) {
	w := Workout{Exercises: []WorkoutExercise{}}
	var sourceID, sourceName *string
	err := tx.QueryRow(ctx, `SELECT w.id::text,w.status,w.revision::text,w.origin,s.id::text,s.name,w.started_at,w.ended_at FROM public.workout w LEFT JOIN public.training_session s ON (s.owner_mail,s.id)=(w.owner_mail,w.source_session_id) WHERE w.owner_mail=$1 AND w.id=$2`, owner, id).Scan(&w.ID, &w.Status, &w.Revision, &w.Origin, &sourceID, &sourceName, &w.StartedAt, &w.EndedAt)
	if err != nil {
		return w, err
	}
	if sourceID != nil {
		w.SourceSession = &WorkoutSource{ID: *sourceID, Name: *sourceName}
	}
	rows, err := tx.Query(ctx, `SELECT a.id::text,a.variant_id::text,a.position,a.state,a.origin,a.config_revision::text,a.progression_mode,a.rep_threshold,a.auto_increment::text,a.rest_seconds,a.completed_at,a.recorded_rpe,a.notes,e.id::text,e.name,e.revision::text,e.muscle_group_codes,e.archived_at IS NOT NULL,v.name,v.revision::text,v.progression_mode,v.rep_threshold,v.auto_increment::text,v.rest_seconds,v.archived_at IS NOT NULL FROM public.workout_exercise a JOIN public.exercise_variant v ON (v.owner_mail,v.id)=(a.owner_mail,a.variant_id) JOIN public.exercise e ON (e.owner_mail,e.id)=(v.owner_mail,v.exercise_id) WHERE a.owner_mail=$1 AND a.workout_id=$2 ORDER BY a.position`, owner, id)
	if err != nil {
		return w, err
	}
	for rows.Next() {
		var a WorkoutExercise
		err = rows.Scan(&a.ID, &a.VariantID, &a.Position, &a.State, &a.Origin, &a.ConfigRevision, &a.ProgressionMode, &a.RepThreshold, &a.AutoIncrement, &a.RestSeconds, &a.CompletedAt, &a.RecordedRPE, &a.Notes, &a.Exercise.ID, &a.Exercise.Name, &a.Exercise.Revision, &a.Exercise.MuscleGroupCodes, &a.Exercise.Archived, &a.CurrentVariant.Name, &a.CurrentVariant.Revision, &a.CurrentVariant.ProgressionMode, &a.CurrentVariant.RepThreshold, &a.CurrentVariant.AutoIncrement, &a.CurrentVariant.RestSeconds, &a.CurrentVariant.Archived)
		if err != nil {
			rows.Close()
			return w, err
		}
		a.CurrentVariant.ID = a.VariantID
		w.Exercises = append(w.Exercises, a)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return w, err
	}
	for i := range w.Exercises {
		a := &w.Exercises[i]
		a.Sets = []WorkoutSet{}
		a.CurrentVariant.TargetSets = []TargetSet{}
		rows, err = tx.Query(ctx, `SELECT id::text,target_set_id::text,position,kind,state,target_reps,target_weight::text,recorded_reps,recorded_weight::text,recorded_at,late_weight_override FROM public.workout_set WHERE owner_mail=$1 AND workout_exercise_id=$2 ORDER BY position`, owner, a.ID)
		if err != nil {
			return w, err
		}
		for rows.Next() {
			var set WorkoutSet
			if err = rows.Scan(&set.ID, &set.TargetSetID, &set.Position, &set.Kind, &set.State, &set.TargetReps, &set.TargetWeight, &set.RecordedReps, &set.RecordedWeight, &set.RecordedAt, &set.LateWeightOverride); err != nil {
				rows.Close()
				return w, err
			}
			a.Sets = append(a.Sets, set)
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return w, err
		}
		rows, err = tx.Query(ctx, `SELECT id::text,position,target_reps,target_weight::text FROM public.variant_target_set WHERE owner_mail=$1 AND variant_id=$2 ORDER BY position`, owner, a.VariantID)
		if err != nil {
			return w, err
		}
		for rows.Next() {
			var target TargetSet
			if err = rows.Scan(&target.ID, &target.Position, &target.TargetReps, &target.TargetWeight); err != nil {
				rows.Close()
				return w, err
			}
			a.CurrentVariant.TargetSets = append(a.CurrentVariant.TargetSets, target)
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return w, err
		}
		a.CurrentVariant.PrescribedSets = len(a.CurrentVariant.TargetSets)
	}
	return w, nil
}

func (s *Service) GetWorkout(ctx context.Context, owner, id string) (out Workout, err error) {
	defer func() { err = storageError(err) }()
	if !validID(id) {
		return out, invalid("workoutId", "Use a UUID workout ID")
	}
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	out, err = readWorkout(ctx, tx, owner, id)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}
func (s *Service) ActiveWorkout(ctx context.Context, owner string) (out *Workout, err error) {
	defer func() { err = storageError(err) }()
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var id string
	err = tx.QueryRow(ctx, `SELECT id::text FROM public.workout WHERE owner_mail=$1 AND status='active'`, owner).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, tx.Commit(ctx)
	}
	if err != nil {
		return nil, err
	}
	w, err := readWorkout(ctx, tx, owner, id)
	if err != nil {
		return nil, err
	}
	return &w, tx.Commit(ctx)
}

func (s *Service) StartWorkout(ctx context.Context, owner string, c StartWorkout) (out WorkoutResult, err error) {
	defer func() { err = storageError(err) }()
	command, err := s.beginCommand(ctx, owner, c.OperationID, "start-workout", "", c)
	if err != nil {
		return out, err
	}
	defer command.rollback(ctx)
	tx := command.tx
	finish := func(id string) (WorkoutResult, error) {
		w, err := readWorkout(ctx, tx, owner, id)
		if err != nil {
			return out, err
		}
		if err = command.commit(ctx, id, w.Revision); err != nil {
			return out, err
		}
		return WorkoutResult{Operation: command.operation, Workout: w}, nil
	}
	if command.replay {
		return finish(command.targetID)
	}
	if !validID(c.SessionID) {
		return out, invalid("sessionId", "Use a UUID session ID")
	}
	n, parseErr := strconv.ParseInt(c.ExpectedSessionRevision, 10, 64)
	if parseErr != nil || n < 1 || strconv.FormatInt(n, 10) != c.ExpectedSessionRevision {
		return out, invalid("expectedSessionRevision", "Use the reviewed positive session revision string")
	}
	// Resolve the active workout before stale requested-plan guards. A second
	// device must learn the actual source rather than replacing or restarting it.
	var active string
	err = tx.QueryRow(ctx, `SELECT id::text FROM public.workout WHERE owner_mail=$1 AND status='active' FOR UPDATE`, owner).Scan(&active)
	if err == nil {
		return finish(active)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	var revision string
	if err = tx.QueryRow(ctx, `SELECT revision::text FROM public.training_session WHERE owner_mail=$1 AND id=$2 FOR UPDATE`, owner, c.SessionID).Scan(&revision); err != nil {
		return out, err
	}
	if revision != c.ExpectedSessionRevision {
		return out, conflict("stale-revision", "The session changed. Review its current plan before starting")
	}
	plan, err := readSession(ctx, tx, owner, c.SessionID)
	if err != nil {
		return out, err
	}
	if len(plan.Placements) == 0 {
		return out, invalid("sessionId", "Add a variant before starting this empty session")
	}
	// All application mutations first serialize on the owner; lock referenced
	// exercise/variant rows in stable order as a final consistent snapshot guard.
	items := []SessionItem{}
	for _, p := range plan.Placements {
		items = append(items, SessionItem{Kind: "reuseVariant", ExerciseID: p.Exercise.ID, VariantID: p.VariantID, ExpectedVariantRevision: p.Variant.Revision})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].VariantID < items[j].VariantID })
	if err = lockSessionItems(ctx, tx, owner, items); err != nil {
		return out, err
	}
	id := newID()
	if _, err = tx.Exec(ctx, `INSERT INTO public.workout(id,owner_mail,source_session_id,origin,status) VALUES($1,$2,$3,'session','active')`, id, owner, c.SessionID); err != nil {
		return out, err
	}
	for _, p := range plan.Placements {
		a := newID()
		v := p.Variant
		if _, err = tx.Exec(ctx, `INSERT INTO public.workout_exercise(id,owner_mail,workout_id,variant_id,position,state,origin,progression_mode,rep_threshold,auto_increment,config_revision,rest_seconds) VALUES($1,$2,$3,$4,$5,'draft','planned',$6,$7,$8,$9,$10)`, a, owner, id, v.ID, p.Position, v.ProgressionMode, v.RepThreshold, v.AutoIncrement, v.Revision, v.RestSeconds); err != nil {
			return out, err
		}
		for _, t := range v.TargetSets {
			if _, err = tx.Exec(ctx, `INSERT INTO public.workout_set(id,owner_mail,workout_exercise_id,target_set_id,position,kind,state,target_reps,target_weight) VALUES($1,$2,$3,$4,$5,'prescribed','pending',$6,$7)`, newID(), owner, a, t.ID, t.Position, t.TargetReps, t.TargetWeight); err != nil {
				return out, err
			}
		}
	}
	return finish(id)
}
