package training

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// StatsEntry preserves imported aggregates; only set-derived volume is exact.
// Decimal strings keep exact volume independent of display/average precision.
type StatsEntry struct {
	ID              string     `json:"id"`
	OccurredAt      *time.Time `json:"occurredAt"`
	SourceTimestamp *string    `json:"sourceTimestamp"`
	SourceTimezone  *string    `json:"sourceTimezone"`
	SetCount        *int       `json:"setCount"`
	AverageReps     *string    `json:"averageReps"`
	AverageWeight   *string    `json:"averageWeight"`
	Volume          *string    `json:"volume"`
	Representation  string     `json:"representation"`
	VolumeEstimated bool       `json:"volumeEstimated"`
}
type StatsTrack struct {
	ExerciseID   string       `json:"exerciseId"`
	ExerciseName string       `json:"exerciseName"`
	VariantID    *string      `json:"variantId"`
	VariantName  *string      `json:"variantName"`
	Entries      []StatsEntry `json:"entries"`
}

// Stats reads history without selectable-library or saved-session filtering.
func (s *Service) Stats(ctx context.Context, owner string) (out []StatsTrack, err error) {
	defer func() { err = storageError(err) }()
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT e.id::text,e.name,v.id::text,v.name FROM public.exercise e JOIN public.exercise_variant v ON (v.owner_mail,v.exercise_id)=(e.owner_mail,e.id) WHERE e.owner_mail=$1
 UNION ALL SELECT e.id::text,e.name,NULL,NULL FROM public.exercise e WHERE e.owner_mail=$1 AND EXISTS (SELECT 1 FROM public.legacy_history_entry h WHERE h.owner_mail=e.owner_mail AND h.exercise_id=e.id AND h.variant_id IS NULL)
 ORDER BY 2,4 NULLS LAST,1,3`, owner)
	if err != nil {
		return nil, err
	}
	out = []StatsTrack{}
	indexes := map[string]int{}
	key := func(e string, v *string) string {
		if v == nil {
			return e
		}
		return e + "/" + *v
	}
	for rows.Next() {
		var t StatsTrack
		if err = rows.Scan(&t.ExerciseID, &t.ExerciseName, &t.VariantID, &t.VariantName); err != nil {
			rows.Close()
			return nil, err
		}
		t.Entries = []StatsEntry{}
		indexes[key(t.ExerciseID, t.VariantID)] = len(out)
		out = append(out, t)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows, err = tx.Query(ctx, `SELECT v.exercise_id::text,a.variant_id::text,a.id::text,a.completed_at,NULL::text,NULL::text,
 count(s.id) FILTER (WHERE s.state='performed')::integer,
 (avg(s.recorded_reps) FILTER (WHERE s.state='performed'))::text,
 (avg(s.recorded_weight) FILTER (WHERE s.state='performed'))::text,
 coalesce(sum(s.recorded_weight*s.recorded_reps) FILTER (WHERE s.state='performed'),0.00)::text,
 'set_derived',false
 FROM public.workout_exercise a JOIN public.exercise_variant v ON (v.owner_mail,v.id)=(a.owner_mail,a.variant_id)
 LEFT JOIN public.workout_set s ON (s.owner_mail,s.workout_exercise_id)=(a.owner_mail,a.id)
 WHERE a.owner_mail=$1 AND a.state='completed' GROUP BY v.exercise_id,a.variant_id,a.id,a.completed_at
 UNION ALL
 SELECT h.exercise_id::text,h.variant_id::text,h.id::text,h.occurred_at,to_char(h.source_save_date,'YYYY-MM-DD"T"HH24:MI:SS.US'),h.source_timezone,
 h.no_of_sets,h.avg_rep::text,h.avg_weight::text,
 CASE WHEN h.avg_rep>=0 AND h.avg_weight>=0 AND h.avg_weight<>'NaN'::numeric AND h.no_of_sets>0 THEN (h.avg_rep::numeric*h.avg_weight*h.no_of_sets)::text END,
 h.representation,true FROM public.legacy_history_entry h WHERE h.owner_mail=$1
 ORDER BY 4 NULLS LAST,3`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var e StatsEntry
		var ex string
		var v *string
		if err = rows.Scan(&ex, &v, &e.ID, &e.OccurredAt, &e.SourceTimestamp, &e.SourceTimezone, &e.SetCount, &e.AverageReps, &e.AverageWeight, &e.Volume, &e.Representation, &e.VolumeEstimated); err != nil {
			return nil, err
		}
		if e.OccurredAt != nil {
			u := e.OccurredAt.UTC()
			e.OccurredAt = &u
		}
		if i, ok := indexes[key(ex, v)]; ok {
			out[i].Entries = append(out[i].Entries, e)
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return out, tx.Commit(ctx)
}
