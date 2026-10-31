//go:build integration

package db_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

// TestDatabaseSchema inspects metadata only. It never reads application rows.
// Run with -count=1 so Go cannot return a cached schema snapshot.
func TestDatabaseSchema(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		values, err := godotenv.Read(filepath.Join("..", ".env"))
		if err != nil {
			t.Fatal("set DATABASE_URL or provide apps/server/.env")
		}
		dsn = values["DATABASE_URL"]
	}
	if dsn == "" {
		t.Fatal("DATABASE_URL is empty")
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		// Parse errors can contain the password-bearing connection string.
		t.Fatal("invalid DATABASE_URL")
	}
	config.ConnectTimeout = 15 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		t.Fatal("database connection failed (connection details omitted)")
	}
	defer conn.Close(context.Background())
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		t.Fatalf("begin read-only schema snapshot: %v", err)
	}
	defer tx.Rollback(context.Background())

	var doc strings.Builder
	doc.WriteString("# Database schema\n\n")
	doc.WriteString("Application tables in the `public` schema. Refresh with `make db-schema` using `DATABASE_URL` from your environment or `apps/server/.env`. The integration test reads metadata only.\n\n")
	doc.WriteString("`User.active_session` holds a session ID as text; the application links it to `Session.ses_id`, but there is no foreign key.\n\n")
	for _, section := range schemaSections {
		rows, err := tx.Query(ctx, section.query)
		if err != nil {
			t.Fatalf("inspect %s: %v", section.title, err)
		}
		fmt.Fprintf(&doc, "## %s\n\n", section.title)
		fields := rows.FieldDescriptions()
		for _, field := range fields {
			fmt.Fprintf(&doc, "| %s ", field.Name)
		}
		doc.WriteString("|\n")
		for range fields {
			doc.WriteString("| --- ")
		}
		doc.WriteString("|\n")
		count := 0
		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				rows.Close()
				t.Fatalf("decode %s: %v", section.title, err)
			}
			for _, value := range values {
				cell := "NULL"
				if value != nil {
					cell = fmt.Sprint(value)
				}
				cell = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "|", "&#124;", "\r", "", "\n", "<br>").Replace(cell)
				fmt.Fprintf(&doc, "| %s ", cell)
			}
			doc.WriteString("|\n")
			count++
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("read %s: %v", section.title, err)
		}
		doc.WriteString("\n")
		t.Logf("%s: %d objects/rows", section.title, count)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("finish snapshot: %v", err)
	}
	if output := os.Getenv("SCHEMA_OUTPUT"); output != "" {
		if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(output, []byte(doc.String()), 0644); err != nil {
			t.Fatal(err)
		}
		t.Logf("schema written to %s", output)
	} else {
		t.Log("\n" + doc.String())
	}
}

// Every query uses pg_catalog metadata; none selects application table contents.
// Keep ordering explicit so snapshots can be reviewed with ordinary git diffs.
var schemaSections = []struct{ title, query string }{
	{"Tables", `SELECT c.relname AS table,
 (SELECT string_agg(a.attname, ', ' ORDER BY k.position)
  FROM pg_constraint pk CROSS JOIN LATERAL unnest(pk.conkey) WITH ORDINALITY AS k(attnum, position)
  JOIN pg_attribute a ON a.attrelid = pk.conrelid AND a.attnum = k.attnum
  WHERE pk.conrelid = c.oid AND pk.contype = 'p') AS primary_key
 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
 WHERE n.nspname = 'public' AND c.relkind IN ('r','p') ORDER BY c.relname`},
	{"Columns", `SELECT c.relname AS table, a.attname AS column,
 format_type(a.atttypid, a.atttypmod) AS type,
 CASE WHEN a.attnotnull THEN 'No' ELSE 'Yes' END AS nullable
 FROM pg_attribute a JOIN pg_class c ON c.oid = a.attrelid
 JOIN pg_namespace n ON n.oid = c.relnamespace
 WHERE n.nspname = 'public' AND c.relkind IN ('r','p')
 AND a.attnum > 0 AND NOT a.attisdropped ORDER BY c.relname, a.attnum`},
	{"Relationships", `SELECT c.relname AS table, pg_get_constraintdef(co.oid, true) AS links_to
 FROM pg_constraint co JOIN pg_class c ON c.oid = co.conrelid
 JOIN pg_namespace n ON n.oid = c.relnamespace
 WHERE n.nspname = 'public' AND co.contype = 'f' ORDER BY c.relname, co.conname`},
}
