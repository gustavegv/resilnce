package training

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

// A command owns one transaction. Callers defer rollback, check replay before
// revision/finality guards, read authoritative results through tx, then commit.
// Owner serialization also makes independent-device commands and receipt
// publication one ordered sequence. Every training write must use this wrapper.
type command struct {
	tx                       pgx.Tx
	replay                   bool
	targetID                 string
	operation                Acknowledgment
	owner, kind, fingerprint string
}

func (s *Service) beginCommand(ctx context.Context, owner, operationID, kind, targetID string, payload any) (*command, error) {
	if !validID(operationID) {
		return nil, invalid("operationId", "Use a UUID operation ID")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(raw)
	c := &command{owner: owner, kind: kind, targetID: targetID, fingerprint: hex.EncodeToString(digest[:]), operation: Acknowledgment{OperationID: operationID}}
	c.tx, err = s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	failed := true
	defer func() {
		if failed {
			c.tx.Rollback(ctx)
		}
	}()
	if _, err = c.tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 127427))`, owner); err != nil {
		return nil, err
	}
	var existingKind, existingFingerprint string
	err = c.tx.QueryRow(ctx, `SELECT kind,target_id::text,payload_fingerprint,accepted_revision::text,committed_at FROM public.mutation_receipt WHERE owner_mail=$1 AND operation_id=$2`, owner, operationID).Scan(&existingKind, &c.targetID, &existingFingerprint, &c.operation.AcceptedRevision, &c.operation.CommittedAt)
	if err == nil {
		if existingKind != kind || existingFingerprint != c.fingerprint || (targetID != "" && targetID != c.targetID) {
			return nil, conflict("operation-mismatch", "This operation ID was already used for another intent. Review and use a new operation ID")
		}
		c.replay = true
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	failed = false
	return c, nil
}
func (c *command) rollback(ctx context.Context) { c.tx.Rollback(ctx) }
func (c *command) commit(ctx context.Context, targetID, revision string) error {
	if !c.replay {
		c.targetID = targetID
		err := c.tx.QueryRow(ctx, `INSERT INTO public.mutation_receipt(owner_mail,operation_id,kind,target_id,payload_fingerprint,accepted_revision) VALUES($1,$2,$3,$4,$5,$6) RETURNING accepted_revision::text,committed_at`, c.owner, c.operation.OperationID, c.kind, targetID, c.fingerprint, revision).Scan(&c.operation.AcceptedRevision, &c.operation.CommittedAt)
		if err != nil {
			return err
		}
	}
	return c.tx.Commit(ctx)
}
