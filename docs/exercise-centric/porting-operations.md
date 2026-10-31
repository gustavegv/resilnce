# Running the initial porting tool

The additive migrations, shared Go importer, CLI, worker, protected API, and
`/internal/porting` page are implemented. The existing tracker still uses the legacy
tables. Application behavior, merge/deletion integration, and production cutover
remain separate work. Actual imports are enabled by default for the allowlisted internal
tool and the CLI/worker; starting a process does not start a data import.

## Database setup and audit

Open `/internal/porting` on the frontend (normally
`http://localhost:5173/internal/porting` during development). Start the Go backend
with `go run .` from `apps/server` and sign in with the allowlisted account.
The page checks the database configured by the backend's `DATABASE_URL`, including
Supabase connections. When migrations are pending, click **Apply additive migrations**.
It applies them transactionally and refreshes the layout check. Schema setup does
not require the separate worker or enabling data imports.

The check works on the untouched six-table legacy layout. It checks the migration
ledger/checksums, table/column types and nullability, key counts, named indexes,
required triggers, RPE/notes constraint definitions and validation, and row security.
Altered or untracked destination objects are
listed for review; the tool does not rebuild existing tables automatically.
Audits and imports become available once the layout passes the check.

The bundled migrations create the initial domain and bookkeeping tables, then use
`003_exercise_rpe.sql` and `004_notes.sql` to ALTER existing tables. These add optional
RPE and nullable notes on variants, workouts, and exercise attempts, including the
0–10 RPE bounds, paired last-rating timestamp, and 4,000-character notes limits.
Existing/imported rows stay unrated and without notes; later retries preserve user
notes. Already applied migrations keep their checksums and are not replayed.

An apply run also applies pending migrations and validates the full layout before
queueing or resuming, and again before writing each account. This covers CLI imports
and worker execution as well as page-started imports. Failed upgrades roll back the
DDL transaction, and incompatible layouts block the import before domain writes.
Auditing never applies migrations; the page still offers explicit setup before its
audit/import controls become available.

The CLI remains available for operational use:

The CLI loads `.env` from the working directory just like `go run .`, preserving
exported environment variables. It uses `DATABASE_URL` by default; an optional
`PORTING_DATABASE_URL` takes precedence for a separate rehearsal database.
The database must contain the six legacy tables. Run from `apps/server`:

```sh
export PORTING_DATABASE_URL='postgres://.../isolated_rehearsal_database'
go run ./cmd/portdata migrate
go run ./cmd/portdata audit
```

Migrations are embedded, versioned, checksum-checked, transactional, and additive.
Starting the web server never applies them. PostgreSQL 17 is covered by the integration
suite. The backend and worker currently use the migration-owning database role, which
can bypass RLS on the new tables. Browser roles receive no grants or policies for
the new domain tables or private bookkeeping. Configure equivalent explicit privileges
before using a distinct runtime role.

Audit emits JSON and exits with status 2 when any account is incomplete. It writes
neither source data, destination data, nor bookkeeping. Source timestamps default
to `Europe/Stockholm`. Supply a different IANA timezone only when needed:

```sh
go run ./cmd/portdata audit --owner account@example.com
```

Blank timezone values also use Stockholm time; `--source-timezone UTC` explicitly
overrides it. Missing rest durations default to 0 seconds; negative durations remain
invalid. Ambiguous/nonexistent DST times remain issues. Numeric conversion accepts exact cents without rounding; null,
zero, and out-of-range increments require a reviewed source-data policy. Missing
owners, malformed/multidimensional arrays, name collisions, unconfigured exercises,
and active legacy workouts also block their affected accounts. Durable jobs preserve
raw source values and keys in issue records.

The page includes unchecked **Delete unused exercises** and **Also delete their
history** options. The CLI equivalents are `--delete-unused-exercises` and
`--delete-unused-history`; the history flag requires the exercise flag. An exercise
qualifies only when it has no session placements, no retained Set rows, and no prior
import mapping. History is preserved unless both options are selected. Already mapped
history prevents automatic cleanup.

Audit previews separate exercise and history deletion counts without changing source
data. Apply deletes the selected unused exercises and their history in the account
transaction, together with durable mappings of every deleted source key. Original
source data remains in private import audit records. Source exercise/history locks
and exact history-key comparisons prevent a stale scan from deleting newly added
history or a newly used exercise. A failed transaction restores both source records
and mappings. With the history option off, orphans with history remain setup issues.

## Apply and resume

Freeze legacy writes for the import scope. With the API and worker restarted on the
updated code, **Import pending records** is enabled after schema setup. The CLI can
also apply pending migrations and run a real import directly:

```sh
go run ./cmd/portdata run --actor operator@example.com --owner account@example.com
go run ./cmd/portdata status --id RUN_UUID
go run ./cmd/portdata resume --id RUN_UUID
```

Import is enabled when `PORTING_APPLY_ENABLED` is unset, blank, or `true`. Set it to
`false` to disable actual imports in either process; other unrecognized nonblank
values also disable them. Both the API and worker must permit apply jobs. Existing
deployments with an explicit disabling value must unset it or set it to `true` and
restart. The allowlist, verified sign-in, Origin checks, and per-account source
validation still apply. Deletion remains separately opt-in through the page's
checkboxes or the CLI flags described above.

`run` creates a durable job and executes it using the shared worker implementation.
`resume` requeues an incomplete/failed job or an expired lease. Job options and the
original actor remain recorded. A run ending incomplete exits with status 2. Empty
account scope includes all owners and unowned rows; broken accounts do not prevent
valid accounts from importing. Reports distinguish source rows, skipped mappings,
newly imported rows, pending work, retained legacy rows, and created destination objects.
Deletion candidates and committed source deletions have separate counts. Retry uses
the run's saved options; start a new audit/import to change either deletion option or
explicit timezone. Existing runs without a timezone use the Stockholm default.

The account graph commits atomically with its mappings. History and Set batches of at
most 500 source rows then commit with their mappings. Retries skip committed source
keys. Duplicate names use PostgreSQL `lower(btrim(name))`; independent placements
become stable variants even when their settings match. New placements added after an
initial import append after current destination placements, preserving user edits.

History retains original aggregates, naive timestamps, source keys, and provenance,
plus an explicitly normalized occurrence time. Variant attribution requires a single
source placement. Undated Set rows use separate compatibility storage. Neither
creates workouts or individual historical workout sets.

Mappings and source groups survive domain deletion. New dependent rows targeting a
deleted destination become issues. Future move/merge operations must transactionally
update destination redirects in both `internal.porting_map` and
`internal.porting_group`; that application integration remains pending.

## Internal page and independent worker

Schema checking and setup are available immediately through the internal API:
`GET /internal/porting/schema` and `POST /internal/porting/schema/apply`.
Both require the allowlisted verified identity; applying also checks the mutation
Origin. Setup calls the same migration runner as the CLI through the backend's
existing database pool. Only the bundled additive migrations can be applied.

For queued audit/import jobs, start a dedicated long-lived worker from `apps/server`.
It automatically loads the same `.env` and `DATABASE_URL` as the API:

```sh
go run ./cmd/portdata worker
```

Already queued audits are picked up automatically. If `PORTING_DATABASE_URL` is
configured, it overrides `DATABASE_URL`; point it at the API's database to process
jobs created through the page.

The Docker image includes `/portdata`; override its entrypoint with `/portdata` and
pass `worker` for a separate worker deployment. The worker needs CPU while no HTTP
requests are active, such as an always-running process or equivalent worker hosting.
API and worker both enable imports by default and honor `PORTING_APPLY_ENABLED=false`
as an explicit override. Audit jobs work with apply disabled.

Open `/internal/porting` on the frontend. Sign in again to populate trusted
`email_verified` in the live Redis session. The server authorizes only
`gustavengvall@yahoo.se`, binds the cookie identity to the session email, and checks
the configured frontend Origin for every mutation. The page shows inventory, audit
issues, variant assignments, recent runs, per-owner results, and retry controls.

HTTP start requests enqueue work promptly. Jobs use renewable 45-second leases and
10-second heartbeats; stopped workers leave reclaimable jobs. Per-owner database
locks and exact source-key mappings protect concurrent runs. Closing the browser does
not cancel work. The initial worker loads the source inventory into memory; very
large databases may need paginated scans and smaller graph transactions before cutover.

## Verification

Unit tests need no services:

```sh
go test ./...
go vet ./...
```

Integration tests require an explicit isolated test DSN whose database name ends in
`_test`; the role needs `CREATEDB`. Each test creates and removes its own uniquely
named database and never reads the application `.env`:

```sh
PORTING_TEST_DATABASE_URL='postgres://postgres:local-test-password@127.0.0.1:55432/resilnce_porting_test?sslmode=disable' go test -tags=integration ./porting -count=1
```

Coverage includes migration reruns, grouping, ordering, unchanged targets, timestamp
precision, aggregate/Set preservation, independent broken accounts, graph and batch
rollback, resume, concurrent runs, expired leases, skipped user edits, new source
keys, deletion tombstones, ownership, and target-count constraints. API tests cover
authorization, mutation origins, schema setup without bookkeeping, repeated/concurrent
setup requests, source preservation, and detection of changed layouts/checksums.
Upgrade coverage includes a populated pre-notes layout, RPE/notes constraint drift,
Unicode note length and multiline storage, default-enabled import setup, pre-import
schema rejection, and preservation of variant notes on retries.
The option/default suite also covers unused-exercise selection, explicit history
cleanup, deletion rollback, exact history-key/reference rechecks, missing rest,
default Stockholm timestamps, explicit timezone overrides, and deletion-aware inventory.

Existing users should cut over after the new application is validated, source
exceptions are resolved, and the gates in [migration.md](migration.md) are satisfied.
The tool does not freeze legacy writes or switch the application itself.
