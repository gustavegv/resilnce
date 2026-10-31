package migrations

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Status struct {
	Ready      bool              `json:"ready"`
	CanApply   bool              `json:"canApply"`
	Migrations []MigrationStatus `json:"migrations"`
	Issues     []string          `json:"issues"`
}

type MigrationStatus struct {
	Version string `json:"version"`
	Applied bool   `json:"applied"`
}

// LayoutError means additive migrations cannot safely repair the current layout.
// It is distinct from an unavailable database or insufficient database privileges.
type LayoutError struct{ Issues []string }

func (e *LayoutError) Error() string {
	return "Database layout needs attention: " + strings.Join(e.Issues, "; ")
}

// Inspect is read-only and works before the bookkeeping schema exists.
func Inspect(ctx context.Context, pool *pgxpool.Pool) (Status, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Status{}, err
	}
	defer tx.Rollback(context.Background())
	status, err := inspect(ctx, tx)
	if err != nil {
		return status, err
	}
	return status, tx.Commit(ctx)
}

type tableLayout struct {
	name string
	// PostgreSQL format_type spelling, with underscores standing for spaces.
	// A trailing ! requires NOT NULL. Extra columns are permitted.
	columns     string
	fks, unique int
}

var legacyLayout = []tableLayout{
	{"User", "mail:character_varying(255)! signup_date:timestamp_without_time_zone name:character_varying(255) active_session:text", 0, 0},
	{"Exercise", "ex_id:integer! ex_name:character_varying(255) auto_inc:numeric(6,2) mail:character_varying(255)", 1, 0},
	{"Session", "ses_id:integer! ses_name:character_varying(255) date_last_ran:timestamp_without_time_zone mail:character_varying(255)", 1, 0},
	{"SesExercise", "ses_id:integer! ex_id:integer! rep_threshold:smallint set_count:smallint rep_per_set:smallint[] weight_per_set:real[] rest_seconds:smallint order:smallint finished:boolean!", 2, 0},
	{"History", "ex_id:integer! save_date:timestamp_without_time_zone! avg_rep:integer avg_weight:numeric(6,2) no_of_sets:integer", 1, 0},
	{"Set", "ses_id:integer! ex_id:integer! set_count:integer! weight:numeric(6,2) rep:integer", 1, 0},
}

var domainLayout = []tableLayout{
	{"exercise", "id:uuid! owner_mail:character_varying(255)! name:character_varying(100)! muscle_group_codes:text[]! archived_at:timestamp_with_time_zone revision:bigint! created_at:timestamp_with_time_zone! updated_at:timestamp_with_time_zone!", 1, 1},
	{"exercise_variant", "id:uuid! owner_mail:character_varying(255)! exercise_id:uuid! name:character_varying(100)! progression_mode:text! rep_threshold:integer! auto_increment:numeric(8,2)! rest_seconds:integer! archived_at:timestamp_with_time_zone revision:bigint! created_at:timestamp_with_time_zone! updated_at:timestamp_with_time_zone!", 1, 1},
	{"variant_target_set", "id:uuid! owner_mail:character_varying(255)! variant_id:uuid! position:integer! target_reps:integer! target_weight:numeric(8,2)!", 1, 2},
	{"training_session", "id:uuid! owner_mail:character_varying(255)! name:character_varying(100)! revision:bigint! created_at:timestamp_with_time_zone! updated_at:timestamp_with_time_zone!", 1, 1},
	{"session_variant", "id:uuid! owner_mail:character_varying(255)! session_id:uuid! variant_id:uuid! position:integer!", 2, 2},
	{"workout", "id:uuid! owner_mail:character_varying(255)! source_session_id:uuid origin:text! status:text! revision:bigint! started_at:timestamp_with_time_zone! ended_at:timestamp_with_time_zone", 2, 1},
	{"workout_exercise", "id:uuid! owner_mail:character_varying(255)! workout_id:uuid! variant_id:uuid! position:integer! state:text! origin:text! progression_mode:text! rep_threshold:integer! auto_increment:numeric(8,2)! config_revision:bigint! completed_at:timestamp_with_time_zone", 2, 2},
	{"workout_set", "id:uuid! owner_mail:character_varying(255)! workout_exercise_id:uuid! target_set_id:uuid position:integer! kind:text! state:text! target_reps:integer! target_weight:numeric(8,2)! recorded_reps:integer recorded_weight:numeric(8,2) recorded_at:timestamp_with_time_zone late_weight_override:boolean!", 2, 1},
	{"mutation_receipt", "owner_mail:character_varying(255)! operation_id:uuid! kind:text! target_id:uuid! payload_fingerprint:text! accepted_revision:bigint! committed_at:timestamp_with_time_zone!", 1, 0},
	{"legacy_history_entry", "id:uuid! owner_mail:character_varying(255)! exercise_id:uuid! variant_id:uuid source_ex_id:integer! source_save_date:timestamp_without_time_zone! occurred_at:timestamp_with_time_zone source_timezone:text avg_rep:integer avg_weight:numeric(6,2) no_of_sets:integer representation:text! source_data:jsonb! imported_at:timestamp_with_time_zone!", 2, 1},
	{"legacy_set_entry", "id:uuid! owner_mail:character_varying(255)! exercise_id:uuid! variant_id:uuid source_ses_id:integer! source_ex_id:integer! source_set_count:integer! weight:numeric(6,2) rep:integer source_data:jsonb! imported_at:timestamp_with_time_zone!", 2, 1},
}

var bookkeepingLayout = []tableLayout{
	{"porting_run", "id:uuid! migration_version:text! actor:text! mode:text! owner_scope:text status:text! options:jsonb! result:jsonb error:text created_at:timestamp_with_time_zone! started_at:timestamp_with_time_zone finished_at:timestamp_with_time_zone lease_token:uuid lease_until:timestamp_with_time_zone heartbeat_at:timestamp_with_time_zone", 0, 0},
	{"porting_owner", "run_id:uuid! owner_key:text! status:text! result:jsonb! updated_at:timestamp_with_time_zone!", 1, 0},
	{"porting_group", "migration_version:text! owner_mail:text! normalized_name:text! exercise_id:uuid! next_variant_number:integer!", 0, 0},
	{"porting_map", "migration_version:text! source_table:text! source_key:jsonb! owner_key:text! destinations:jsonb! source_data:jsonb! imported_at:timestamp_with_time_zone!", 0, 0},
	{"porting_issue", "run_id:uuid! owner_key:text! source_table:text! source_key:jsonb! code:text! severity:text! details:text! source_data:jsonb! resolution_status:text!", 1, 0},
}

var migrationIndexes = map[string][]string{
	"001_exercise_centric.sql": {"exercise_owner_name", "variant_exercise_name", "variant_owner_exercise", "target_owner_variant", "session_variant_owner_variant", "single_active_workout", "workout_source_session", "attempt_variant_history", "workout_set_target", "legacy_history_exercise", "legacy_history_variant", "legacy_set_exercise", "legacy_set_variant"},
	"002_porting.sql":          {"porting_run_queue", "porting_map_owner"},
}

func inspect(ctx context.Context, tx pgx.Tx) (Status, error) {
	s := Status{Migrations: []MigrationStatus{}, Issues: []string{}}
	type column struct {
		typ     string
		notNull bool
	}
	type table struct {
		columns         map[string]column
		rls             bool
		pk, fks, unique int
	}
	tables := map[string]table{}
	rows, err := tx.Query(ctx, `SELECT n.nspname||'.'||c.relname,a.attname,format_type(a.atttypid,a.atttypmod),a.attnotnull,c.relrowsecurity,
 (SELECT count(*) FROM pg_constraint WHERE conrelid=c.oid AND contype='p'),
 (SELECT count(*) FROM pg_constraint WHERE conrelid=c.oid AND contype='f' AND convalidated),
 (SELECT count(*) FROM pg_constraint WHERE conrelid=c.oid AND contype='u')
 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
 JOIN pg_attribute a ON a.attrelid=c.oid AND a.attnum>0 AND NOT a.attisdropped
 WHERE n.nspname IN ('public','internal') AND c.relkind IN ('r','p')`)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var name, field, typ string
		var notNull, rls bool
		var pk, fks, unique int
		if err = rows.Scan(&name, &field, &typ, &notNull, &rls, &pk, &fks, &unique); err != nil {
			rows.Close()
			return s, err
		}
		t := tables[name]
		if t.columns == nil {
			t.columns = map[string]column{}
		}
		t.columns[field] = column{typ, notNull}
		t.rls = rls
		t.pk = pk
		t.fks = fks
		t.unique = unique
		tables[name] = t
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return s, err
	}
	checkTable := func(schema string, expected tableLayout, checkRelations bool) {
		name := schema + "." + expected.name
		actual, exists := tables[name]
		if !exists {
			s.Issues = append(s.Issues, "Missing table "+name)
			return
		}
		for _, spec := range strings.Fields(expected.columns) {
			parts := strings.SplitN(spec, ":", 2)
			required := strings.HasSuffix(parts[1], "!")
			typ := strings.ReplaceAll(strings.TrimSuffix(parts[1], "!"), "_", " ")
			col, ok := actual.columns[parts[0]]
			if !ok || col.typ != typ || col.notNull != required {
				s.Issues = append(s.Issues, "Unexpected or missing column "+name+"."+parts[0])
			}
		}
		if checkRelations && (actual.pk != 1 || actual.fks != expected.fks || actual.unique != expected.unique) {
			s.Issues = append(s.Issues, "Unexpected primary key, foreign keys, or unique constraints on "+name)
		}
	}
	for _, t := range legacyLayout {
		checkTable("public", t, true)
	}
	applied := map[string]string{}
	if _, exists := tables["internal.schema_migration"]; exists {
		before := len(s.Issues)
		checkTable("internal", tableLayout{"schema_migration", "version:text! checksum:text! applied_at:timestamp_with_time_zone!", 0, 0}, true)
		if len(s.Issues) == before {
			rows, err = tx.Query(ctx, `SELECT version,checksum FROM internal.schema_migration ORDER BY version`)
			if err != nil {
				return s, err
			}
			for rows.Next() {
				var name, sum string
				if err = rows.Scan(&name, &sum); err != nil {
					rows.Close()
					return s, err
				}
				applied[name] = sum
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return s, err
			}
		}
	}
	// Named indexes and constraint triggers supplement the column checks. Check
	// actual catalog objects as well as the ledger, so a dropped table is not ready.
	objects := map[string]bool{}
	rows, err = tx.Query(ctx, `SELECT n.nspname||'.'||c.relname FROM pg_index i JOIN pg_class c ON c.oid=i.indexrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('public','internal') AND i.indisvalid
 UNION ALL SELECT n.nspname||'.'||c.relname||'.'||t.tgname FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND t.tgenabled IN ('O','A')`)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			rows.Close()
			return s, err
		}
		objects[name] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return s, err
	}
	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return s, err
	}
	pending := 0
	known := map[string]bool{}
	for _, name := range names {
		known[name] = true
		previous, done := applied[name]
		s.Migrations = append(s.Migrations, MigrationStatus{name, done})
		if !done {
			pending++
		} else {
			sql, err := files.ReadFile(name)
			if err != nil {
				return s, err
			}
			if previous != fmt.Sprintf("%x", sha256.Sum256(sql)) {
				s.Issues = append(s.Issues, "Migration checksum differs: "+name)
			}
		}
		// Later ALTER migrations validate only their added columns, and only once
		// applied. Existing domain tables are expected while these are pending.
		if name == "003_exercise_rpe.sql" {
			if done {
				checkTable("public", tableLayout{name: "workout_exercise", columns: "recorded_rpe:smallint"}, false)
				checkTable("public", tableLayout{name: "exercise_variant", columns: "last_rpe:smallint last_rpe_at:timestamp_with_time_zone"}, false)
			}
			continue
		}
		layouts := domainLayout
		schema := "public"
		if name == "002_porting.sql" {
			layouts = bookkeepingLayout
			schema = "internal"
		}
		for _, t := range layouts {
			if done {
				checkTable(schema, t, true)
			} else if _, exists := tables[schema+"."+t.name]; exists {
				s.Issues = append(s.Issues, "Untracked table already exists: "+schema+"."+t.name)
			}
		}
		if done {
			for _, index := range migrationIndexes[name] {
				if !objects[schema+"."+index] {
					s.Issues = append(s.Issues, "Missing or invalid index "+schema+"."+index)
				}
			}
		}
	}
	for name := range applied {
		if !known[name] {
			s.Issues = append(s.Issues, "Database contains a migration unavailable in this server: "+name)
		}
	}
	if _, done := applied["001_exercise_centric.sql"]; done {
		for _, name := range []string{"public.exercise_variant.variant_requires_targets", "public.variant_target_set.target_count"} {
			if !objects[name] {
				s.Issues = append(s.Issues, "Missing or disabled trigger "+name)
			}
		}
	}
	if _, done := applied["002_porting.sql"]; done {
		for _, t := range domainLayout {
			if !tables["public."+t.name].rls {
				s.Issues = append(s.Issues, "Row level security is disabled on public."+t.name)
			}
		}
	}
	s.Ready = pending == 0 && len(s.Issues) == 0
	s.CanApply = pending > 0 && len(s.Issues) == 0
	return s, nil
}
