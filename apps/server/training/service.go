package training

import (
	"context"
	"crypto/rand"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct{ Pool *pgxpool.Pool }

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func readSession(ctx context.Context, q pgx.Tx, owner, id string) (Session, error) {
	s := Session{Placements: []Placement{}}
	err := q.QueryRow(ctx, `SELECT s.id::text,s.name,s.revision::text,s.created_at,
 coalesce(performance.completed_at,CASE WHEN imported.source_timezone IS NOT NULL THEN imported.session_last_ran_at END),
 CASE WHEN performance.completed_at IS NOT NULL OR (imported.source_timezone IS NOT NULL AND imported.session_last_ran_at IS NOT NULL) THEN 'dated'
      WHEN imported.raw_date IS NOT NULL THEN 'unknown' ELSE 'never' END
 FROM public.training_session s
 LEFT JOIN LATERAL (SELECT max(a.completed_at) AS completed_at FROM public.workout w JOIN public.workout_exercise a ON (a.owner_mail,a.workout_id)=(w.owner_mail,w.id) WHERE w.owner_mail=s.owner_mail AND w.source_session_id=s.id AND a.state='completed') performance ON true
 LEFT JOIN LATERAL (SELECT m.source_timezone,m.session_last_ran_at,m.source_data->>'date_last_ran' AS raw_date FROM internal.porting_map m WHERE m.owner_key=s.owner_mail AND m.migration_version='exercise-centric-v1' AND m.source_table='Session' AND m.destinations->>'session'=s.id::text ORDER BY m.source_key LIMIT 1) imported ON true
 WHERE s.owner_mail=$1 AND s.id=$2`, owner, id).Scan(&s.ID, &s.Name, &s.Revision, &s.CreatedAt, &s.LastPerformedAt, &s.LastPerformedStatus)
	if err != nil {
		return s, err
	}
	if s.LastPerformedAt != nil {
		normalized := s.LastPerformedAt.UTC()
		s.LastPerformedAt = &normalized
	}
	rows, err := q.Query(ctx, `SELECT p.id::text,p.variant_id::text,p.position,e.id::text,e.name,e.revision::text,e.muscle_group_codes,e.archived_at IS NOT NULL,v.name,v.revision::text,v.progression_mode,v.rep_threshold,v.auto_increment::text,v.rest_seconds,v.archived_at IS NOT NULL FROM public.session_variant p JOIN public.exercise_variant v ON (v.owner_mail,v.id)=(p.owner_mail,p.variant_id) JOIN public.exercise e ON (e.owner_mail,e.id)=(v.owner_mail,v.exercise_id) WHERE p.owner_mail=$1 AND p.session_id=$2 ORDER BY p.position`, owner, id)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var p Placement
		err = rows.Scan(&p.ID, &p.VariantID, &p.Position, &p.Exercise.ID, &p.Exercise.Name, &p.Exercise.Revision, &p.Exercise.MuscleGroupCodes, &p.Exercise.Archived, &p.Variant.Name, &p.Variant.Revision, &p.Variant.ProgressionMode, &p.Variant.RepThreshold, &p.Variant.AutoIncrement, &p.Variant.RestSeconds, &p.Variant.Archived)
		if err != nil {
			rows.Close()
			return s, err
		}
		p.Variant.ID = p.VariantID
		s.Placements = append(s.Placements, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return s, err
	}
	for i := range s.Placements {
		v := &s.Placements[i].Variant
		v.TargetSets = []TargetSet{}
		rows, err := q.Query(ctx, `SELECT id::text,position,target_reps,target_weight::text FROM public.variant_target_set WHERE owner_mail=$1 AND variant_id=$2 ORDER BY position`, owner, v.ID)
		if err != nil {
			return s, err
		}
		for rows.Next() {
			var t TargetSet
			if err = rows.Scan(&t.ID, &t.Position, &t.TargetReps, &t.TargetWeight); err != nil {
				rows.Close()
				return s, err
			}
			v.TargetSets = append(v.TargetSets, t)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return s, err
		}
		v.PrescribedSets = len(v.TargetSets)
	}
	return s, nil
}

func (s *Service) GetSession(ctx context.Context, owner, id string) (out Session, err error) {
	defer func() { err = storageError(err) }()
	if !validID(id) {
		return Session{}, invalid("sessionId", "Use a UUID session ID")
	}
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback(ctx)
	result, err := readSession(ctx, tx, owner, id)
	if err != nil {
		return Session{}, err
	}
	return result, tx.Commit(ctx)
}
func (s *Service) ListSessions(ctx context.Context, owner string) (out []Session, err error) {
	defer func() { err = storageError(err) }()
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT id::text FROM public.training_session WHERE owner_mail=$1 ORDER BY created_at DESC,id`, owner)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := []Session{}
	for _, id := range ids {
		v, err := readSession(ctx, tx, owner, id)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	sort.SliceStable(result, func(i, j int) bool {
		a, b := result[i].LastPerformedAt, result[j].LastPerformedAt
		if a == nil && b == nil {
			return result[i].ID < result[j].ID
		}
		if a == nil {
			return false
		}
		if b == nil {
			return true
		}
		if a.Equal(*b) {
			return result[i].ID < result[j].ID
		}
		return a.After(*b)
	})
	return result, tx.Commit(ctx)
}
func (s *Service) CreateSession(ctx context.Context, owner string, c CreateSession) (out SessionResult, err error) {
	defer func() { err = storageError(err) }()
	command, err := s.beginCommand(ctx, owner, c.OperationID, "create-session", "", c)
	if err != nil {
		return SessionResult{}, err
	}
	defer command.rollback(ctx)
	tx := command.tx
	if command.replay {
		plan, err := readSession(ctx, tx, owner, command.targetID)
		if err != nil {
			return SessionResult{}, err
		}
		if err := command.commit(ctx, "", ""); err != nil {
			return SessionResult{}, err
		}
		return SessionResult{Operation: command.operation, Session: plan}, nil
	}
	if err := validateCreate(c); err != nil {
		return SessionResult{}, err
	}
	id := newID()
	if _, err = tx.Exec(ctx, `INSERT INTO public.training_session(id,owner_mail,name) VALUES($1,$2,$3)`, id, owner, strings.TrimSpace(c.Name)); err != nil {
		return SessionResult{}, err
	}
	if err = lockSessionItems(ctx, tx, owner, c.Items); err != nil {
		return SessionResult{}, err
	}
	seen := map[string]bool{}
	changedExercises := map[string]bool{}
	for i, item := range c.Items {
		v := item.VariantID
		if item.Kind != "reuseVariant" {
			e := item.ExerciseID
			if item.Kind == "createExercise" {
				e = newID()
				codes, _ := normalizeCodes(item.MuscleGroupCodes)
				if _, err = tx.Exec(ctx, `INSERT INTO public.exercise(id,owner_mail,name,muscle_group_codes) VALUES($1,$2,$3,$4)`, e, owner, strings.TrimSpace(item.Name), codes); err != nil {
					return SessionResult{}, err
				}
			} else {
				changedExercises[e] = true
			}
			name := strings.TrimSpace(item.Name)
			if item.Kind == "createExercise" {
				name = "Standard"
			}
			v, err = createVariant(ctx, tx, owner, e, name, item.Configuration)
			if err != nil {
				return SessionResult{}, err
			}
		}
		if seen[v] {
			return SessionResult{}, invalid("items", "A variant can appear only once in a session")
		}
		seen[v] = true
		if _, err = tx.Exec(ctx, `INSERT INTO public.session_variant(id,owner_mail,session_id,variant_id,position) VALUES($1,$2,$3,$4,$5)`, newID(), owner, id, v, i+1); err != nil {
			return SessionResult{}, err
		}
	}
	for e := range changedExercises {
		if _, err = tx.Exec(ctx, `UPDATE public.exercise SET revision=revision+1,updated_at=now() WHERE owner_mail=$1 AND id=$2`, owner, e); err != nil {
			return SessionResult{}, err
		}
	}
	plan, err := readSession(ctx, tx, owner, id)
	if err != nil {
		return SessionResult{}, err
	}
	if err = command.commit(ctx, id, plan.Revision); err != nil {
		return SessionResult{}, err
	}
	return SessionResult{Operation: command.operation, Session: plan}, nil
}
