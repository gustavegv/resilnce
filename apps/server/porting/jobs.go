package porting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrApplyDisabled = errors.New("apply is disabled; enable PORTING_APPLY_ENABLED only for a rehearsal or reviewed cutover")
var ErrNotResumable = errors.New("run is still active or already completed")

type Run struct {
	ID         string        `json:"id"`
	Actor      string        `json:"actor"`
	Mode       string        `json:"mode"`
	Scope      *string       `json:"ownerScope"`
	Status     string        `json:"status"`
	Options    Options       `json:"options"`
	CreatedAt  time.Time     `json:"createdAt"`
	StartedAt  *time.Time    `json:"startedAt"`
	FinishedAt *time.Time    `json:"finishedAt"`
	LeaseUntil *time.Time    `json:"leaseUntil"`
	Error      *string       `json:"error"`
	Owners     []OwnerReport `json:"owners"`
}

func (s *Service) StartRun(ctx context.Context, actor, mode, scope string, opts Options) (string, error) {
	opts = opts.normalized()
	if mode != "audit" && mode != "apply" {
		return "", errors.New("mode must be audit or apply")
	}
	if mode == "apply" && !s.AllowApply {
		return "", ErrApplyDisabled
	}
	if actor == "" {
		return "", errors.New("initiating actor is required")
	}
	if err := opts.Validate(); err != nil {
		return "", err
	}
	id := newID()
	var scoped any
	if scope != "" {
		scoped = scope
	}
	_, err := s.Pool.Exec(ctx, `INSERT INTO internal.porting_run(id,migration_version,actor,mode,owner_scope,status,options) VALUES($1,$2,$3,$4,$5,'queued',$6)`, id, Version, actor, mode, scoped, opts)
	return id, err
}

func (s *Service) ResumeRun(ctx context.Context, id string) error {
	var mode string
	if err := s.Pool.QueryRow(ctx, `SELECT mode FROM internal.porting_run WHERE id=$1 AND migration_version=$2`, id, Version).Scan(&mode); err != nil {
		return err
	}
	if mode == "apply" && !s.AllowApply {
		return ErrApplyDisabled
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE internal.porting_run SET status='queued',finished_at=NULL,error=NULL,lease_token=NULL,lease_until=NULL
 WHERE id=$1 AND migration_version=$2 AND (status IN ('failed','incomplete') OR (status='running' AND lease_until<now()))`, id, Version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotResumable
	}
	return nil
}

func (s *Service) GetRun(ctx context.Context, id string) (Run, error) {
	var r Run
	err := s.Pool.QueryRow(ctx, `SELECT id::text,actor,mode,owner_scope,status,options,created_at,started_at,finished_at,lease_until,error FROM internal.porting_run WHERE id=$1 AND migration_version=$2`, id, Version).Scan(&r.ID, &r.Actor, &r.Mode, &r.Scope, &r.Status, &r.Options, &r.CreatedAt, &r.StartedAt, &r.FinishedAt, &r.LeaseUntil, &r.Error)
	if err != nil {
		return r, err
	}
	r.Options = r.Options.normalized()
	r.Owners = []OwnerReport{}
	rows, err := s.Pool.Query(ctx, `SELECT result FROM internal.porting_owner WHERE run_id=$1 ORDER BY owner_key`, id)
	if err != nil {
		return r, err
	}
	defer rows.Close()
	for rows.Next() {
		var owner OwnerReport
		if err := rows.Scan(&owner); err != nil {
			return r, err
		}
		r.Owners = append(r.Owners, owner)
	}
	return r, rows.Err()
}

func (s *Service) RecentRuns(ctx context.Context) ([]Run, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id::text,actor,mode,owner_scope,status,options,created_at,started_at,finished_at,lease_until,error FROM internal.porting_run WHERE migration_version=$1 ORDER BY created_at DESC LIMIT 20`, Version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Run{}
	for rows.Next() {
		var r Run
		if err := rows.Scan(&r.ID, &r.Actor, &r.Mode, &r.Scope, &r.Status, &r.Options, &r.CreatedAt, &r.StartedAt, &r.FinishedAt, &r.LeaseUntil, &r.Error); err != nil {
			return nil, err
		}
		r.Options = r.Options.normalized()
		out = append(out, r)
	}
	return out, rows.Err()
}

// WorkOne claims a queued or expired job. A detached runner calls this repeatedly;
// HTTP requests only enqueue jobs. A lease prevents a crashed runner hiding a job.
func (s *Service) WorkOne(ctx context.Context, id string) (bool, error) {
	token := newID()
	var runID string
	err := s.Pool.QueryRow(ctx, `WITH candidate AS (
 SELECT id FROM internal.porting_run WHERE migration_version=$1
 AND (status='queued' OR (status='running' AND lease_until<now()))
 AND ($2::text='' OR id::text=$2) AND (mode='audit' OR $4)
 ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1
) UPDATE internal.porting_run r SET status='running',lease_token=$3,lease_until=now()+interval '45 seconds',heartbeat_at=now(),started_at=coalesce(started_at,now()),finished_at=NULL,error=NULL
 FROM candidate c WHERE r.id=c.id RETURNING r.id::text`, Version, id, token, s.AllowApply).Scan(&runID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	run, err := s.GetRun(ctx, runID)
	if err != nil {
		return true, err
	}
	jobCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		timer := time.NewTicker(10 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-jobCtx.Done():
				return
			case <-timer.C:
				hbCtx, stop := context.WithTimeout(jobCtx, 5*time.Second)
				tag, e := s.Pool.Exec(hbCtx, `UPDATE internal.porting_run SET lease_until=now()+interval '45 seconds',heartbeat_at=now() WHERE id=$1 AND lease_token=$2 AND status='running'`, runID, token)
				stop()
				if e != nil || tag.RowsAffected() != 1 {
					cancel()
					return
				}
			}
		}
	}()
	executeErr := s.execute(jobCtx, run, token)
	cancel()
	<-heartbeatDone
	// A cancelled worker leaves an expiring lease; another runner can retry.
	if ctx.Err() != nil {
		return true, ctx.Err()
	}
	finishCtx, stop := context.WithTimeout(ctx, 10*time.Second)
	defer stop()
	if executeErr != nil {
		_, err = s.Pool.Exec(finishCtx, `UPDATE internal.porting_run SET status='failed',error=$3,finished_at=now(),lease_until=NULL WHERE id=$1 AND lease_token=$2`, runID, token, executeErr.Error())
		return true, errors.Join(executeErr, err)
	}
	final, err := s.GetRun(finishCtx, runID)
	if err != nil {
		return true, err
	}
	status := "completed"
	for _, owner := range final.Owners {
		if !owner.Ready || owner.Error != "" {
			status = "incomplete"
		}
	}
	_, err = s.Pool.Exec(finishCtx, `UPDATE internal.porting_run SET status=$3,result=$4,finished_at=now(),lease_until=NULL WHERE id=$1 AND lease_token=$2`, runID, token, status, final.Owners)
	return true, err
}

func (s *Service) execute(ctx context.Context, run Run, token string) error {
	all, err := s.readSource(ctx)
	if err != nil {
		return err
	}
	scope := ""
	if run.Scope != nil {
		scope = *run.Scope
	}
	ownerKeys := owners(all, scope)
	if scope != "" && len(ownerKeys) == 0 {
		return errors.New("source account not found")
	}
	for _, owner := range ownerKeys {
		if err := ctx.Err(); err != nil {
			return err
		}
		var report OwnerReport
		if run.Mode == "apply" {
			report, err = s.ImportOwner(ctx, all, owner, run.Options)
		} else {
			tx, e := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
			if e != nil {
				return e
			}
			state, e := loadState(ctx, tx, owner)
			if e == nil {
				p := buildPlan(all, owner, state, run.Options)
				report = p.OwnerReport
			}
			closeErr := tx.Rollback(context.Background())
			_ = closeErr
			err = e
		}
		if err != nil {
			report.Owner = owner
			report.Ready = false
			report.Error = err.Error()
		}
		if err = s.saveReport(ctx, run.ID, token, report); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) saveReport(ctx context.Context, id, token string, report OwnerReport) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	var live bool
	if err = tx.QueryRow(ctx, `SELECT lease_token=$2 AND lease_until>now() FROM internal.porting_run WHERE id=$1 FOR UPDATE`, id, token).Scan(&live); err != nil {
		return err
	}
	if !live {
		return fmt.Errorf("worker lease lost")
	}
	status := "completed"
	if !report.Ready {
		status = "incomplete"
	}
	if report.Error != "" {
		status = "failed"
	}
	if _, err = tx.Exec(ctx, `INSERT INTO internal.porting_owner(run_id,owner_key,status,result) VALUES($1,$2,$3,$4) ON CONFLICT(run_id,owner_key) DO UPDATE SET status=excluded.status,result=excluded.result,updated_at=now()`, id, report.Owner, status, report); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM internal.porting_issue WHERE run_id=$1 AND owner_key=$2`, id, report.Owner); err != nil {
		return err
	}
	for _, issue := range report.Issues {
		// Cross-owner issues may be visible to both owners; the stored source key is exact.
		if _, err = tx.Exec(ctx, `INSERT INTO internal.porting_issue(run_id,owner_key,source_table,source_key,code,severity,details,source_data) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(run_id,source_table,source_key,code) DO UPDATE SET details=excluded.details`, id, report.Owner, issue.Table, issue.Key, issue.Code, issue.Severity, issue.Details, issue.Source); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// JSONReport provides the same machine-readable report to operational callers.
func JSONReport(v any) ([]byte, error) { return json.MarshalIndent(v, "", "  ") }
