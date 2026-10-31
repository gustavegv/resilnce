// Package migrations applies additive, versioned migrations explicitly. The server
// never migrates a database as a side effect of starting up.
package migrations

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed *.sql
var files embed.FS

func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(72643101)`); err != nil {
		return err
	}
	status, err := inspect(ctx, tx)
	if err != nil {
		return err
	}
	if len(status.Issues) > 0 {
		return &LayoutError{Issues: status.Issues}
	}
	if status.Ready {
		return tx.Commit(ctx)
	}
	if _, err = tx.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS internal;
REVOKE ALL ON SCHEMA internal FROM PUBLIC;
CREATE TABLE IF NOT EXISTS internal.schema_migration (
 version text PRIMARY KEY, checksum text NOT NULL, applied_at timestamptz NOT NULL DEFAULT now()
)`); err != nil {
		return err
	}
	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return err
	}
	for _, name := range names {
		sql, err := files.ReadFile(name)
		if err != nil {
			return err
		}
		sum := fmt.Sprintf("%x", sha256.Sum256(sql))
		var previous string
		err = tx.QueryRow(ctx, `SELECT checksum FROM internal.schema_migration WHERE version=$1`, name).Scan(&previous)
		if err == nil {
			if previous != sum {
				return fmt.Errorf("migration %s checksum changed", name)
			}
			continue
		}
		if err != pgx.ErrNoRows {
			return err
		}
		if _, err = tx.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO internal.schema_migration(version,checksum) VALUES($1,$2)`, name, sum); err != nil {
			return err
		}
	}
	status, err = inspect(ctx, tx)
	if err != nil {
		return err
	}
	if !status.Ready {
		return &LayoutError{Issues: status.Issues}
	}
	return tx.Commit(ctx)
}
