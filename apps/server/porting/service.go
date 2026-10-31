package porting

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gustavegv/resilnce/apps/server/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	Pool       *pgxpool.Pool
	AllowApply bool
}

// An actual import upgrades and validates the layout before queueing work and
// again before writing an account. Audit never invokes this mutating preflight.
func (s *Service) prepareApply(ctx context.Context) error {
	if !s.AllowApply {
		return ErrApplyDisabled
	}
	return migrations.Apply(ctx, s.Pool)
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func loadState(ctx context.Context, tx pgx.Tx, owner string) (State, error) {
	s := emptyState()
	rows, err := tx.Query(ctx, `SELECT source_table,source_key,destinations FROM internal.porting_map WHERE migration_version=$1 AND owner_key=$2`, Version, owner)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var table string
		var key json.RawMessage
		var refs References
		if err = rows.Scan(&table, &key, &refs); err != nil {
			rows.Close()
			return s, err
		}
		var parts []any
		if err = json.Unmarshal(key, &parts); err != nil {
			rows.Close()
			return s, err
		}
		canonical, _ := json.Marshal(parts)
		s.Mapped[table+":"+string(canonical)] = refs
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return s, err
	}
	rows, err = tx.Query(ctx, `SELECT normalized_name,exercise_id::text,next_variant_number FROM internal.porting_group WHERE migration_version=$1 AND owner_mail=$2`, Version, owner)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var g Group
		if err = rows.Scan(&g.Name, &g.Exercise, &g.Next); err != nil {
			rows.Close()
			return s, err
		}
		s.Groups[g.Name] = g
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return s, err
	}
	rows, err = tx.Query(ctx, `SELECT id::text,lower(btrim(name)) FROM public.exercise WHERE owner_mail=$1`, owner)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var id, name string
		if err = rows.Scan(&id, &name); err != nil {
			rows.Close()
			return s, err
		}
		s.Exercises[id] = name
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return s, err
	}
	rows, err = tx.Query(ctx, `SELECT id::text,exercise_id::text,lower(btrim(name)) FROM public.exercise_variant WHERE owner_mail=$1`, owner)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var id, ex, name string
		if err = rows.Scan(&id, &ex, &name); err != nil {
			rows.Close()
			return s, err
		}
		s.Variants[id] = ex
		if s.VariantNames[ex] == nil {
			s.VariantNames[ex] = map[string]bool{}
		}
		s.VariantNames[ex][name] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return s, err
	}
	rows, err = tx.Query(ctx, `SELECT t.id::text,coalesce(max(s.position),0) FROM public.training_session t LEFT JOIN public.session_variant s ON s.session_id=t.id WHERE t.owner_mail=$1 GROUP BY t.id`, owner)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var id string
		var n int
		if err = rows.Scan(&id, &n); err != nil {
			rows.Close()
			return s, err
		}
		s.Sessions[id] = true
		s.LastPosition[id] = n
	}
	err = rows.Err()
	rows.Close()
	return s, err
}

func (s *Service) readSource(ctx context.Context) ([]Row, error) {
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())
	rows, err := snapshot(ctx, tx)
	if err != nil {
		return nil, err
	}
	return rows, tx.Commit(ctx)
}

// Audit is read-only, including importer metadata. Durable audit jobs persist the
// returned report separately. Both audit and apply call the same planner.
func (s *Service) Audit(ctx context.Context, scope string, opts Options) ([]OwnerReport, error) {
	opts = opts.normalized()
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())
	all, err := snapshot(ctx, tx)
	if err != nil {
		return nil, err
	}
	reports := []OwnerReport{}
	for _, owner := range owners(all, scope) {
		state, err := loadState(ctx, tx, owner)
		if err != nil {
			return nil, err
		}
		p := buildPlan(all, owner, state, opts)
		reports = append(reports, p.OwnerReport)
	}
	if scope != "" && len(reports) == 0 {
		return nil, fmt.Errorf("source account not found")
	}
	return reports, tx.Commit(ctx)
}

func lockOwner(ctx context.Context, tx pgx.Tx, owner string) error {
	// Same lock across migration scopes and CLI/worker processes. Domain writers
	// should also lock the User row before editing imported data.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(72643102,hashtext($1))`, owner); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `SELECT mail FROM public."User" WHERE mail=$1 FOR UPDATE`, owner)
	return err
}

func remember(ctx context.Context, tx pgx.Tx, r Row, refs References) error {
	_, err := tx.Exec(ctx, `INSERT INTO internal.porting_map(migration_version,source_table,source_key,owner_key,destinations,source_data) VALUES($1,$2,$3,$4,$5,$6)`, Version, r.Table, r.Key, r.Owner, refs, r.Raw)
	return err
}

// ImportOwner commits the exercise/session graph together, then imports legacy
// performance in bounded batches. Successful source mappings are never overwritten.
func (s *Service) ImportOwner(ctx context.Context, all []Row, owner string, opts Options) (report OwnerReport, err error) {
	opts = opts.normalized()
	defer func() {
		if err != nil {
			report.Ready = false
		}
	}()
	if !s.AllowApply {
		return report, ErrApplyDisabled
	}
	if err = opts.Validate(); err != nil {
		return report, err
	}
	if err = s.prepareApply(ctx); err != nil {
		return report, err
	}
	// The deletion recheck must see references committed before its source-row
	// lock was obtained, even if the database's default isolation is stronger.
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return report, err
	}
	defer tx.Rollback(context.Background())
	if err = lockOwner(ctx, tx, owner); err != nil {
		return report, err
	}
	state, err := loadState(ctx, tx, owner)
	if err != nil {
		return report, err
	}
	p := buildPlan(all, owner, state, opts)
	report = p.OwnerReport
	if !p.Ready {
		return report, nil
	}
	graphCreated := 0
	graphImported := map[string]int{}
	graphDeleted := map[string]int{}
	var exs = map[int]Row{}
	var sessions = map[int]Row{}
	history := map[int][]Row{}
	for _, r := range all {
		if r.Table == "Exercise" {
			exs[r.Data.ExID] = r
		}
		if r.Table == "Session" {
			sessions[r.Data.SesID] = r
		}
		if r.Table == "History" {
			history[r.Data.ExID] = append(history[r.Data.ExID], r)
		}
	}
	for _, r := range p.Rows {
		if _, done := state.Mapped[r.identity()]; done {
			continue
		}
		refs := References{}
		switch r.Table {
		case "User":
			refs.Account = owner
		case "Exercise":
			if p.DeleteUnused[r.identity()] {
				if err = deleteUnusedExercise(ctx, tx, r, history[r.Data.ExID], opts.DeleteUnusedHistory); err != nil {
					return report, err
				}
				refs.DeletedUnused = true
				break
			}
			g, ok := state.Groups[r.Data.NormalizedName]
			if !ok {
				g = Group{Name: r.Data.NormalizedName, Exercise: newID(), Next: 1}
				if _, err = tx.Exec(ctx, `INSERT INTO public.exercise(id,owner_mail,name) VALUES($1,$2,$3)`, g.Exercise, owner, strings.Trim(*r.Data.Name, " ")); err != nil {
					return report, err
				}
				if _, err = tx.Exec(ctx, `INSERT INTO internal.porting_group(migration_version,owner_mail,normalized_name,exercise_id) VALUES($1,$2,$3,$4)`, Version, owner, g.Name, g.Exercise); err != nil {
					return report, err
				}
				state.Groups[g.Name] = g
				graphCreated++
			}
			refs.Exercise = g.Exercise
		case "Session":
			refs.Session = newID()
			if _, err = tx.Exec(ctx, `INSERT INTO public.training_session(id,owner_mail,name) VALUES($1,$2,$3)`, refs.Session, owner, strings.Trim(*r.Data.SessionName, " ")); err != nil {
				return report, err
			}
			graphCreated++
		case "SesExercise":
			a := p.AssignmentsByKey[r.identity()]
			ex := exs[r.Data.ExID]
			refs.Exercise = state.Mapped[ex.identity()].Exercise
			refs.Session = state.Mapped[sessions[r.Data.SesID].identity()].Session
			refs.Variant = newID()
			refs.Placement = newID()
			inc, _ := decimal(ex.Data.Increment, "0.25", "99")
			if _, err = tx.Exec(ctx, `INSERT INTO public.exercise_variant(id,owner_mail,exercise_id,name,progression_mode,rep_threshold,auto_increment,rest_seconds) VALUES($1,$2,$3,$4,'average',$5,$6,$7)`, refs.Variant, owner, refs.Exercise, a.Name, *r.Data.Threshold, inc, a.RestSeconds); err != nil {
				return report, err
			}
			for i, reps := range a.Reps {
				id := newID()
				refs.Targets = append(refs.Targets, id)
				if _, err = tx.Exec(ctx, `INSERT INTO public.variant_target_set(id,owner_mail,variant_id,position,target_reps,target_weight) VALUES($1,$2,$3,$4,$5,$6)`, id, owner, refs.Variant, i+1, reps, a.Weights[i]); err != nil {
					return report, err
				}
			}
			if _, err = tx.Exec(ctx, `INSERT INTO public.session_variant(id,owner_mail,session_id,variant_id,position) VALUES($1,$2,$3,$4,$5)`, refs.Placement, owner, refs.Session, refs.Variant, a.Position); err != nil {
				return report, err
			}
			if a.Name != "Standard" {
				var n int
				fmt.Sscanf(a.Name, "Variant %d", &n)
				if _, err = tx.Exec(ctx, `UPDATE internal.porting_group SET next_variant_number=greatest(next_variant_number,$4) WHERE migration_version=$1 AND owner_mail=$2 AND normalized_name=$3`, Version, owner, a.Group, n+1); err != nil {
					return report, err
				}
			}
			graphCreated += 2 + len(a.Reps)
		case "History":
			if !p.DeleteUnused[r.identity()] {
				continue
			}
			refs.DeletedUnused = true
		default:
			continue
		}
		if err = remember(ctx, tx, r, refs); err != nil {
			return report, err
		}
		state.Mapped[r.identity()] = refs
		if refs.DeletedUnused {
			graphDeleted[r.Table]++
		} else {
			graphImported[r.Table]++
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return report, err
	}
	report.Created = graphCreated
	for table, n := range graphImported {
		report.Counts[table].Imported = n
		report.Counts[table].Pending -= n
	}
	for table, n := range graphDeleted {
		report.Counts[table].Deleted = n
		report.Counts[table].DeletePlanned -= n
		report.Counts[table].Pending -= n
	}
	// Only committed graph counts are reported if a subsequent batch fails.
	pending := []Row{}
	for _, r := range p.Rows {
		if r.Table == "History" || r.Table == "Set" {
			if _, done := state.Mapped[r.identity()]; !done {
				pending = append(pending, r)
			}
		}
	}
	for start := 0; start < len(pending); start += 500 {
		end := min(start+500, len(pending))
		counts, created, batchErr := s.importLegacyBatch(ctx, pending[start:end], all, owner, opts)
		if batchErr != nil {
			report.Ready = false
			return report, batchErr
		}
		for table, n := range counts {
			report.Counts[table].Imported += n
			report.Counts[table].Retained += n
			report.Counts[table].Pending -= n
		}
		report.Created += created
	}
	// Another run may have committed one of our pending batches while we waited
	// for the owner lock. Reconcile those rows as skipped, not outstanding work.
	finalTx, e := s.Pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if e != nil {
		return report, e
	}
	finalState, e := loadState(ctx, finalTx, owner)
	finalTx.Rollback(context.Background())
	if e != nil {
		return report, e
	}
	mapped := map[string]int{}
	deletedMappings := map[string]int{}
	for _, r := range p.Rows {
		if refs, done := finalState.Mapped[r.identity()]; done {
			mapped[r.Table]++
			if refs.DeletedUnused {
				deletedMappings[r.Table]++
			}
		}
	}
	for table, c := range report.Counts {
		c.Mapped = mapped[table] - c.Imported - c.Deleted
		c.Pending = c.Source - mapped[table]
		if table == "History" || table == "Set" {
			c.Retained = mapped[table] - deletedMappings[table]
		}
		if c.Pending != 0 {
			report.Ready = false
		}
	}
	return report, nil
}

func deleteUnusedExercise(ctx context.Context, tx pgx.Tx, r Row, expectedHistory []Row, deleteHistory bool) error {
	// Lock before checking references so a concurrent history/session insert
	// cannot slip between the check and the legacy ON DELETE CASCADE.
	var id int
	if err := tx.QueryRow(ctx, `SELECT ex_id FROM public."Exercise" WHERE ex_id=$1 AND mail=$2 FOR UPDATE`, r.Data.ExID, r.Owner).Scan(&id); err != nil {
		return err
	}
	if deleteHistory {
		// The exercise lock prevents new FK references. Lock existing history and
		// compare its exact keys, so a stale scan cannot cascade-delete extra rows.
		rows, err := tx.Query(ctx, `SELECT to_jsonb(h) FROM public."History" h WHERE h.ex_id=$1 ORDER BY h.save_date FOR UPDATE`, r.Data.ExID)
		if err != nil {
			return err
		}
		current := []Row{}
		for rows.Next() {
			var raw json.RawMessage
			if err = rows.Scan(&raw); err != nil {
				rows.Close()
				return err
			}
			history, err := sourceRow("History", raw, r.Owner)
			if err != nil {
				rows.Close()
				return err
			}
			current = append(current, history)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if !sameSourceKeys(expectedHistory, current) {
			return errors.New("unused exercise history changed; run a new audit before deleting it")
		}
	}
	tag, err := tx.Exec(ctx, `DELETE FROM public."Exercise" e WHERE e.ex_id=$1 AND e.mail=$2
 AND NOT EXISTS(SELECT 1 FROM public."SesExercise" WHERE ex_id=e.ex_id)
 AND ($3 OR NOT EXISTS(SELECT 1 FROM public."History" WHERE ex_id=e.ex_id))
 AND NOT EXISTS(SELECT 1 FROM public."Set" WHERE ex_id=e.ex_id)`, r.Data.ExID, r.Owner, deleteHistory)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("exercise is now in use; run the audit again before deleting unused exercises")
	}
	return nil
}

func sameSourceKeys(expected, current []Row) bool {
	if len(expected) != len(current) {
		return false
	}
	keys := map[string]bool{}
	for _, r := range expected {
		keys[r.identity()] = true
	}
	for _, r := range current {
		if !keys[r.identity()] {
			return false
		}
		delete(keys, r.identity())
	}
	return len(keys) == 0
}

func (s *Service) importLegacyBatch(ctx context.Context, rows, all []Row, owner string, opts Options) (map[string]int, int, error) {
	counts := map[string]int{}
	created := 0
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return counts, 0, err
	}
	defer tx.Rollback(context.Background())
	if err = lockOwner(ctx, tx, owner); err != nil {
		return counts, 0, err
	}
	state, err := loadState(ctx, tx, owner)
	if err != nil {
		return counts, 0, err
	}
	exs := map[int]Row{}
	placements := map[int][]Row{}
	for _, r := range all {
		if r.Table == "Exercise" {
			exs[r.Data.ExID] = r
		}
		if r.Table == "SesExercise" {
			placements[r.Data.ExID] = append(placements[r.Data.ExID], r)
		}
	}
	for _, r := range rows {
		if _, done := state.Mapped[r.identity()]; done {
			continue
		}
		refs := References{Exercise: state.Mapped[exs[r.Data.ExID].identity()].Exercise}
		if _, alive := state.Exercises[refs.Exercise]; !alive {
			return counts, 0, errors.New("dependent source row points to a deleted exercise")
		}
		var variant any
		for _, p := range placements[r.Data.ExID] {
			if (r.Table == "History" && len(placements[r.Data.ExID]) == 1) || (r.Table == "Set" && r.Data.SesID == p.Data.SesID) {
				refs.Variant = state.Mapped[p.identity()].Variant
				variant = refs.Variant
			}
		}
		if refs.Variant != "" {
			if parent, alive := state.Variants[refs.Variant]; !alive || parent != refs.Exercise {
				return counts, 0, errors.New("dependent source row points to a deleted or moved variant; resolve mapping redirect")
			}
		}
		id := newID()
		if r.Table == "History" {
			at, err := occurrence(r.Data.SaveDate, opts.SourceTimezone)
			if err != nil {
				return counts, 0, err
			}
			refs.LegacyHistory = id
			_, err = tx.Exec(ctx, `INSERT INTO public.legacy_history_entry(id,owner_mail,exercise_id,variant_id,source_ex_id,source_save_date,occurred_at,source_timezone,avg_rep,avg_weight,no_of_sets,source_data)
 SELECT $1,$2,$3,$4,$5,$6::text::timestamp,$7,$8,(j->>'avg_rep')::integer,(j->>'avg_weight')::numeric,(j->>'no_of_sets')::integer,j FROM (SELECT $9::jsonb j) s`, id, owner, refs.Exercise, variant, r.Data.ExID, r.Data.SaveDate, at, opts.SourceTimezone, r.Raw)
			if err != nil {
				return counts, 0, err
			}
		} else {
			refs.LegacySet = id
			_, err = tx.Exec(ctx, `INSERT INTO public.legacy_set_entry(id,owner_mail,exercise_id,variant_id,source_ses_id,source_ex_id,source_set_count,weight,rep,source_data)
 SELECT $1,$2,$3,$4,$5,$6,$7,(j->>'weight')::numeric,(j->>'rep')::integer,j FROM (SELECT $8::jsonb j) s`, id, owner, refs.Exercise, variant, r.Data.SesID, r.Data.ExID, *r.Data.SetCount, r.Raw)
			if err != nil {
				return counts, 0, err
			}
		}
		if err = remember(ctx, tx, r, refs); err != nil {
			return counts, 0, err
		}
		counts[r.Table]++
		created++
	}
	return counts, created, tx.Commit(ctx)
}
