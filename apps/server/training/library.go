package training

import (
	"context"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

func readExercise(ctx context.Context, tx pgx.Tx, owner, id string) (ExerciseSelection, error) {
	out := ExerciseSelection{Variants: []Variant{}}
	e := &out.Exercise
	if err := tx.QueryRow(ctx, `SELECT id::text,name,revision::text,muscle_group_codes,archived_at IS NOT NULL FROM public.exercise WHERE owner_mail=$1 AND id=$2 AND archived_at IS NULL`, owner, id).Scan(&e.ID, &e.Name, &e.Revision, &e.MuscleGroupCodes, &e.Archived); err != nil {
		return out, err
	}
	rows, err := tx.Query(ctx, `SELECT id::text,name,revision::text,progression_mode,rep_threshold,auto_increment::text,rest_seconds FROM public.exercise_variant WHERE owner_mail=$1 AND exercise_id=$2 AND archived_at IS NULL ORDER BY lower(name),id`, owner, id)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var v Variant
		v.TargetSets = []TargetSet{}
		if err = rows.Scan(&v.ID, &v.Name, &v.Revision, &v.ProgressionMode, &v.RepThreshold, &v.AutoIncrement, &v.RestSeconds); err != nil {
			rows.Close()
			return out, err
		}
		out.Variants = append(out.Variants, v)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return out, err
	}
	for i := range out.Variants {
		v := &out.Variants[i]
		rows, err = tx.Query(ctx, `SELECT id::text,position,target_reps,target_weight::text FROM public.variant_target_set WHERE owner_mail=$1 AND variant_id=$2 ORDER BY position`, owner, v.ID)
		if err != nil {
			return out, err
		}
		for rows.Next() {
			var t TargetSet
			if err = rows.Scan(&t.ID, &t.Position, &t.TargetReps, &t.TargetWeight); err != nil {
				rows.Close()
				return out, err
			}
			v.TargetSets = append(v.TargetSets, t)
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return out, err
		}
		v.PrescribedSets = len(v.TargetSets)
	}
	return out, nil
}

// SearchExercises searches literal personal-library substrings; it does not fuzzy
// match identities or expose archived records. Empty search lists the library.
func (s *Service) SearchExercises(ctx context.Context, owner, query string) (out []ExerciseSelection, err error) {
	defer func() { err = storageError(err) }()
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT id::text FROM public.exercise WHERE owner_mail=$1 AND archived_at IS NULL AND strpos(lower(name),lower($2))>0 ORDER BY lower(name),id`, owner, strings.TrimSpace(query))
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	out = []ExerciseSelection{}
	for _, id := range ids {
		e, err := readExercise(ctx, tx, owner, id)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, tx.Commit(ctx)
}
func (s *Service) GetExercise(ctx context.Context, owner, id string) (out ExerciseSelection, err error) {
	defer func() { err = storageError(err) }()
	if !validID(id) {
		return out, invalid("exerciseId", "Use a UUID exercise ID")
	}
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	out, err = readExercise(ctx, tx, owner, id)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

// Existing references are locked in stable parent/variant order after the owner
// command lock. Every review guard runs before inserts change parent revisions.
func lockSessionItems(ctx context.Context, tx pgx.Tx, owner string, items []SessionItem) error {
	exerciseIDs, variantIDs := map[string]bool{}, map[string]bool{}
	for _, item := range items {
		if item.Kind != "createExercise" {
			exerciseIDs[item.ExerciseID] = true
		}
		if item.Kind == "reuseVariant" {
			variantIDs[item.VariantID] = true
		}
	}
	sorted := func(ids map[string]bool) []string {
		out := []string{}
		for id := range ids {
			out = append(out, id)
		}
		sort.Strings(out)
		return out
	}
	exerciseRevisions, variantRevisions, parents := map[string]string{}, map[string]string{}, map[string]string{}
	for _, id := range sorted(exerciseIDs) {
		var revision string
		var archived bool
		if err := tx.QueryRow(ctx, `SELECT revision::text,archived_at IS NOT NULL FROM public.exercise WHERE owner_mail=$1 AND id=$2 FOR UPDATE`, owner, id).Scan(&revision, &archived); err != nil {
			return err
		}
		if archived {
			return conflict("resource-unavailable", "This exercise is archived. Choose another exercise")
		}
		exerciseRevisions[id] = revision
	}
	for _, id := range sorted(variantIDs) {
		var revision, parent string
		var archived bool
		if err := tx.QueryRow(ctx, `SELECT revision::text,exercise_id::text,archived_at IS NOT NULL FROM public.exercise_variant WHERE owner_mail=$1 AND id=$2 FOR UPDATE`, owner, id).Scan(&revision, &parent, &archived); err != nil {
			return err
		}
		if archived {
			return conflict("resource-unavailable", "This variant is archived. Choose another variant")
		}
		variantRevisions[id], parents[id] = revision, parent
	}
	for _, item := range items {
		if item.Kind == "createVariant" && exerciseRevisions[item.ExerciseID] != item.ExpectedExerciseRevision {
			return conflict("stale-revision", "The exercise changed. Review its current variants before saving")
		}
		if item.Kind == "reuseVariant" {
			if parents[item.VariantID] != item.ExerciseID {
				return &Error{404, "not-found", "Resource is unavailable to this account", nil}
			}
			if variantRevisions[item.VariantID] != item.ExpectedVariantRevision {
				return conflict("stale-revision", "The variant changed. Review its current configuration before saving")
			}
		}
	}
	return nil
}
func createVariant(ctx context.Context, tx pgx.Tx, owner, exerciseID, name string, cfg Configuration) (string, error) {
	id := newID()
	threshold, inc, rest := 12, "2.50", 0
	if cfg.RepThreshold != nil {
		threshold = *cfg.RepThreshold
	}
	if cfg.AutoIncrement != nil {
		inc = *cfg.AutoIncrement
	}
	if cfg.RestSeconds != nil {
		rest = *cfg.RestSeconds
	}
	if _, err := tx.Exec(ctx, `INSERT INTO public.exercise_variant(id,owner_mail,exercise_id,name,progression_mode,rep_threshold,auto_increment,rest_seconds) VALUES($1,$2,$3,$4,'average',$5,$6,$7)`, id, owner, exerciseID, name, threshold, inc, rest); err != nil {
		return "", err
	}
	for n := 1; n <= cfg.PrescribedSets; n++ {
		if _, err := tx.Exec(ctx, `INSERT INTO public.variant_target_set(id,owner_mail,variant_id,position,target_reps,target_weight) VALUES($1,$2,$3,$4,$5,$6)`, newID(), owner, id, n, threshold, cfg.InitialWeight); err != nil {
			return "", err
		}
	}
	return id, nil
}
