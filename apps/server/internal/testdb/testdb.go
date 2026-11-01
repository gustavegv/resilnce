// Package testdb provisions disposable databases from an explicit isolated DSN.
package testdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func New(t *testing.T, legacyFixture string) (*pgxpool.Pool, context.Context) {
	t.Helper()
	dsn := os.Getenv("APPLICATION_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set APPLICATION_TEST_DATABASE_URL to an explicit isolated PostgreSQL _test database")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid isolated test DSN")
	}
	if !strings.HasSuffix(cfg.ConnConfig.Database, "_test") {
		t.Fatal("isolated test database must end in _test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	admin, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.Ping(ctx); err != nil {
		admin.Close()
		t.Fatal("isolated PostgreSQL is not ready: ", err)
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		t.Fatal(err)
	}
	name := "resilnce_application_" + hex.EncodeToString(id[:]) + "_test"
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	copy := cfg.Copy()
	copy.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, copy)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		clean, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if _, err := admin.Exec(clean, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)"); err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	fixture, err := os.ReadFile(legacyFixture)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(fixture)); err != nil {
		t.Fatal(err)
	}
	return pool, ctx
}
