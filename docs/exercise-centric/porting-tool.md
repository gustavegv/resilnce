# Porting importer and internal tool

**Status:** Initial additive schema, shared audit/apply importer, CLI, worker, and allowlisted
internal page implemented against fixtures. See [operations](porting-operations.md) for setup and
remaining limits. Production import and the new application cutover remain pending.

## 1. Agreed scope

- One porting program transfers old application data into the new model.
- Audit and predictable transformations happen automatically; duplicate exercise names do not
  require individual manual review.
- For one owner, names equal after trimming and case normalization become one base exercise.
  Independent legacy configurations become variants named `Variant 1`, `Variant 2`, and so on.
- Users can rename or merge those variants later.
- A protected internal page provides an overview, audit results, and controls to run the importer.
- Access uses normal sign-in plus a hardcoded email allowlist, enforced server-side.
- Runs can be repeated; previously imported source records are skipped.
- Already imported old rows will not change. Do not build changed-row synchronization, conflict
  comparison, or overwrite logic for this importer.

## 2. One implementation, two entry points

Implement the transformation and database work once in a Go package, proposed as
`apps/server/porting`. Expose it through:

| Entry point                     | Responsibility                                                          |
| ------------------------------- | ----------------------------------------------------------------------- |
| `apps/server/cmd/portdata`      | Operational command with audit/run/resume modes, using the same service |
| Internal Go API and Svelte page | Start an audit or import, inspect status, and resume an interrupted run |

The web endpoint calls the service through a durable job, rather than shelling out to an arbitrary
command. The command-line tool is the script-like entry point; it does not contain a second set of
mapping rules. Source tables are preserved by default. The explicit **Delete unused
exercises** option permits deletion only of unmapped exercises without session
placements or retained Set rows. Historical entries are preserved unless the additional
**Also delete their history** option is selected. Audits only preview those deletions.

Proposed service boundaries are `Audit`, `StartRun`, `ResumeRun`, `GetRun`, and `ImportOwner`. Audit
and apply use the same deterministic transformation functions so their interpretation of the source
cannot drift.

Starting an apply run performs the audit/classification automatically before each import unit; there
is no manual duplicate-name approval step. A separate audit-only mode serves the overview and
rehearsals. Unresolved issues block only their affected unit/account from being marked ready, while
independent valid accounts can continue.

## 3. Automatic mapping

### Exercise grouping

Use the same owner and database name-normalization rule as the new exercise library, initially
`lower(btrim(name))`. Group by owner plus normalized source name. Do not fuzzy-match punctuation,
spelling, or similar lift names. Retain each original exercise ID and name in import provenance.

The group record points to the created base exercise. Future runs resolve through this durable
record even if the user renames that exercise; never rediscover an imported base by its current
editable name. For a source group not yet mapped, a pre-existing destination name collision is
reported for that account rather than silently attaching to unrelated user-created data.

### Variants and stable names

Preserve one progression track per old `SesExercise(ses_id, ex_id)` placement. Matching settings are
not a reason to collapse variants. Sort unmapped placements deterministically by source IDs and
allocate variant numbers under the group lock.

- A group containing multiple configurations initially gets `Variant 1`, `Variant 2`, etc.
- A single initial configuration can use `Standard`, consistent with the product requirement.
- If more source configurations are discovered on a later run, allocate new numbered names without
  renaming Standard or any existing user-renamed variant.
- Store the next variant number with the group. Do not recompute names from the number of current
  variants or reuse an imported record's name/identity after deletion.
- Check the destination's case-insensitive name constraint, skipping any occupied generated name.

Create target-set rows from the old arrays by position, copying current targets without advancing
progression. Copy threshold, increment, and rest settings; missing rest defaults to zero.
Source timestamps default to `Europe/Stockholm` unless an explicit timezone is supplied.
Create the new saved-session placement
with the mapped variant ID and deterministic relative order. Where old order values tie or are null,
automatically use source exercise ID as a tie-breaker, assign fresh sequential positions, and record
a warning; preserve any valid relative order.

### Other source data

Import session definitions through their source IDs, history through its full composite primary key,
and set rows through their full composite primary key. Do not use a date watermark, row count, or
destination name to decide whether a row was already imported.

Keep aggregate history separate from new set-level workout history, as described in the
[history mapping](migration.md#4-preserve-legacy-history-honestly). Preserve old Set rows as a
labeled legacy representation until their meaning is established; never discard them merely because
current Go queries do not reference them.

An orphan exercise has no reliable target configuration. The automatic audit classifies it as
requiring setup; do not fabricate user performance or pretend its configuration was imported.
When exercise deletion is selected, an orphan with no history and no prior mapping is
instead a deletion candidate. Selecting history deletion too includes its unmapped
legacy History rows. Apply locks/rechecks exact source keys and commits source deletion
with durable tombstones for every removed row; audit leaves source data untouched.
An orphan with history remains a setup issue when history deletion is off. Existing
imported data and retained Set rows are preserved.
Missing owners, cross-owner references, and irreconcilable arrays are also issues, not silent
repairs. Preserve unresolved source records in issue/staging data and identify the affected account
as incomplete. Routine duplicate names are a normal automatic transformation, not an issue.

## 4. Durable bookkeeping

Keep porting metadata in a private database schema, proposed as `internal`. Only the backend and
runner database role can read/write it; browser clients do not access it directly.

| Record          | Purpose                                                                                                 |
| --------------- | ------------------------------------------------------------------------------------------------------- |
| `porting_run`   | Requested actor, audit/apply mode, scope, status, counts, start/finish time, and worker lease           |
| `porting_owner` | Account-level progress and result within a run                                                          |
| `porting_group` | Stable owner + normalized source-name group → destination exercise, with naming counter                 |
| `porting_map`   | Migration version + source table + exact source primary key → destination identities and import time    |
| `porting_issue` | Source key, issue code, explanatory details, and resolution status for records that could not be ported |

`porting_map` must support one source row producing multiple destination objects. For example, a
SesExercise source key maps to a variant, its target rows, and a session placement. Use typed
mapping relations or structured destination references; the public domain schema must not depend on
the representation chosen for importer bookkeeping.

Use unique source keys as the correctness boundary. A persistent scan cursor is only an efficiency
aid; it is never proof that earlier source rows have been imported. Every run discovers unmapped
rows, including new history entries for accounts seen on a previous run.

### Atomicity and retries

Write each destination unit and its mapping in the same transaction. A crash before commit leaves
neither; a crash after commit leaves both. On retry, the unique source mapping causes a skip. Never
insert the mapping as successful before its destination writes commit.

Import the account's exercise/session graph as a transaction first, then append history/set records
in bounded batches with atomic mappings. Large accounts can require smaller dependency-aware graph
units; track readiness explicitly so partial progress is not mistaken for a fully ported account.

On duplicate-key races, load the committed mapping rather than generate another destination. A
failed owner must not erase successful work for other owners, and a resumed run must retry only
failed/unmapped work.

### Destination edits, merges, and deletion

Mappings survive ordinary destination edits and deletion. They cannot be removed by cascading
foreign keys. A mapped source row never becomes new again merely because its destination was
renamed, merged, archived, or deleted.

Merge operations update mapping redirects so later unmapped dependent records attach to the
surviving destination. Deletion leaves a tombstone rather than an invitation to resurrect data. If
an unmapped dependent source row points to a deleted destination, report it for explicit resolution
instead of recreating the deleted exercise. This preserves the agreed skip-only rerun behavior
without introducing source-change synchronization.

## 5. Execution and deployment

The internal start endpoint creates a durable queued job and returns its ID promptly. A runner
claims the job using a database lease, renews its heartbeat, and persists progress. The existing
server has a 60-second write timeout, so a whole import must not depend on one open HTTP request.

Run the worker independently of the browser request. A disconnected browser does not cancel the
import. If the worker stops, the lease expires and another worker can resume from committed
mappings. Bound concurrent imports for the same migration/scope using database locking, not a
process-local mutex.

The current Dockerfile builds only the main server binary. Implementation must add the porting
command/runner to an appropriate build/deployment target; the existing container cannot be assumed
to contain an arbitrary Go script or shell.

The imported source is assumed immutable, per the agreed scope. Initial audit/rehearsal and final
cutover are separate operations. Before enabling accounts on the new application, prevent legacy
writes for that cutover scope and verify all required source rows have an imported mapping or an
explicitly resolved retained representation. Repeated imports are not a replacement for the
application cutover gate.

## 6. Internal page and API

Proposed page: `/internal/porting`. Display:

- Database layout status and an apply-migrations button when additive migrations are pending.
  This works before import bookkeeping exists and uses the backend's configured database.
- Overall and per-owner counts for legacy accounts, exercises, placements, history, and Set rows.
- Imported, not-yet-imported, issue, and retained-legacy counts by source type.
- Automatically grouped duplicate names and their generated variant assignments.
- Audit warnings, source keys needing attention, run progress, and recent failure details.
- Controls for audit, run all pending records, retry/resume, and an initial single-account scope.

Each apply run reports new source rows processed, destination objects created, mapped rows skipped,
issues, and outstanding work. Keep source-row counts separate from destination-row counts: grouping
names and splitting arrays change cardinality.

Suggested endpoints are `GET /internal/porting/overview`, `POST /internal/porting/audits`,
`POST /internal/porting/runs`, `GET /internal/porting/runs/{id}`, and
`POST /internal/porting/runs/{id}/resume`. The UI may poll run status; no streaming system is needed
for the first version. There is no UI for arbitrary SQL, shell commands, or silently overwriting
already imported destination data.

Schema setup is implemented through `GET /internal/porting/schema` and
`POST /internal/porting/schema/apply`, protected by the same verified-email allowlist and
mutation-origin checks. It applies only bundled missing migrations in one transaction and does
not enqueue an import or require the worker. The overview returns layout status even before
bookkeeping exists, and import controls remain unavailable until the layout check passes.

## 7. Hardcoded access policy

Use one source-controlled allowlist of email addresses for viewing and running the tool. The
configured member is `gustavengvall@yahoo.se`. Changing membership requires changing/deploying
backend code; do not add a roles table or permissions-management UI for this scope.

Every internal API request must validate the signed cookie, load the live Redis login session, bind
its identity to that cookie, and compare the authenticated email against the allowlist. Reject
unauthenticated requests and deny authenticated nonmembers before querying private overview or job
data. Hiding a navigation link or guarding only the Svelte page is insufficient.

The current Google login verifies an ID token but does not extract `email_verified`. Before relying
on email-based internal access, establish verified email in the trusted login/session identity.
Require renewed sign-in where necessary; never accept verification status or admin membership from
the browser. Cookie-authenticated job-start/resume requests also need the normal mutation
origin/CSRF protection.

Record the initiating admin identity, scope, operation mode, and results in the run metadata. A
runner receives a fixed job specification, not browser-provided credentials or executable code.
Administrative authorization does not grant ordinary users cross-account data access.

## 8. First implementation increments

1. Define mapping/staging tables and deterministic transformation functions against fixtures.
2. Implement the shared audit service and CLI report before any real import writes.
3. Add transactional apply, durable mappings, and a second-run/no-op test.
4. Implement allowlisted internal authentication and read-only overview.
5. Add job creation, runner leases, progress polling, and retry/resume controls.
6. Rehearse a whole import and cutover on a database copy, then enable real runs through the same
   service once the new schema and application are ready.

## 9. Acceptance checks

- A second run with no new source rows creates no domain records and changes no user edits.
- Adding one new source history row imports it exactly once without reimporting its exercise.
- Duplicate names automatically produce one parent with stable, separately named variants.
- Crash/restart and concurrent start requests cannot produce duplicate domain records or mappings.
- Renaming, merging, or deleting imported data does not trigger resurrection on a later run.
- A broken owner does not prevent independent valid owners from importing; the final result names
  incomplete accounts and never reports them as fully ported.
- Ordinary signed-in users cannot read overview data, start a job, inspect its status, or resume it.
- Allowlisted admins can run audit/apply through the UI and inspect the same results as the CLI.
- Closing the internal page leaves the durable job running or safely resumable.
- Every legacy record is accounted for as imported, retained legacy, or a visible issue; no row is
  silently omitted and no aggregate history is fabricated as individual sets.
