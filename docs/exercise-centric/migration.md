# Legacy data porting plan

**Status:** Porting design with agreed automatic name grouping and skip-only reruns. The importer is
implemented in an initial version; see [operations](porting-operations.md). Build and validate the new
application, then rehearse the import on a copy of the
old database. The [internal tool design](porting-tool.md) describes execution and access policy.

The goal is to preserve current targets, saved sessions, and historical information while moving to
reusable variants. Automatically group duplicate names, preserve their separate progression tracks
as variants, and never manufacture missing performance.

## 1. Mapping the current tables

| Current table | Destination                                                 | Import rule                                                                                                          |
| ------------- | ----------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| `User`        | Existing `User`                                             | Retain account identities and ownership; replace active-session lookup in the new application                        |
| `Exercise`    | `exercise`                                                  | Automatically group trimmed, case-insensitive names per owner into a personal base exercise                          |
| `SesExercise` | `exercise_variant`, `variant_target_set`, `session_variant` | Preserve each placement's configuration/progression as an independent variant; split arrays into ordered target rows |
| `Session`     | `training_session`                                          | Preserve owner, name, and relative exercise order; keep legacy last-run metadata in the import record                |
| `History`     | Proposed `legacy_history_entry`                             | Preserve aggregates and source identity; do not invent individual sets or a workout association                      |
| `Set`         | Retained legacy source pending inspection                   | Preserve rows and source keys as legacy set data; do not invent a dated workout association                          |

Keep explicit old-to-new ID mappings. A source `Exercise` maps to a base exercise, while a source
`(ses_id, ex_id)` placement maps to its own variant and saved-session placement. New UUIDs are
allocated once and reused on retries.

## 2. Exercise grouping and variant creation

### Base exercise names

Group names automatically per owner using the same trimmed, case-insensitive normalization as the
new library. Different capitalization or surrounding whitespace does not create another base
exercise. Preserve original names and source IDs in mapping metadata. Do not fuzzy-match spelling or
punctuation. Initialize `muscle_group_codes` to an empty array unless reliable source data can be
mapped to codes in the backend catalog. Do not create database dictionary rows or infer groups from
exercise names during import.

Duplicate names are a normal transformation rather than an audit blocker. Their configurations
become distinct variants that users may rename or merge after import. Stable group mappings allow
reruns to find the same destination even after it has been renamed.

### Preserve progression independently

Proposed conservative default: create one variant per old `SesExercise` placement, even if two
placements happen to have identical settings today. They previously carried independent targets;
sharing a new variant would change future behavior.

Use `Standard` when a base exercise has only one imported configuration. Multiple initial
configurations are named `Variant 1`, `Variant 2`, etc., allocated deterministically and recorded in
the mapping. Users can rename or deliberately merge them afterwards. Later runs allocate new variant
names without renaming existing variants or resetting the numbering.

For example:

| Old data                                   | New representation      |
| ------------------------------------------ | ----------------------- |
| Push: Bench Press, three sets at 50 kg     | Bench Press → Variant 1 |
| Full Body: Bench Press, four sets at 40 kg | Bench Press → Variant 2 |

Both saved sessions continue using their prior configuration. Reusing or merging one variant later
is an explicit choice, rather than an irreversible migration guess.

An old `Exercise` with no session placement has no complete set configuration in that table. Retain
it in staging and resolve how to initialize its Standard variant; do not silently invent targets.
The importer optionally deletes unmapped exercises without placements or retained Set
records when **Delete unused exercises** is selected. If **Also delete their history**
is selected too, their unmapped History rows are deleted as well. Audit previews the
exercise/history counts; apply records each deleted source key and original data in
private bookkeeping in the same transaction. Without the history option, orphans with
history continue requiring setup. Already imported data is preserved.

## 3. Targets and settings

For every source placement:

1. Copy `Exercise.auto_inc`, `SesExercise.rep_threshold`, and `rest_seconds` into the variant;
   a missing rest duration defaults to zero.
2. Set progression mode to `average` for valid existing enabled progression.
3. Expand `rep_per_set` and `weight_per_set` by array position into prescribed target rows.
4. Preserve relative session ordering, resolving ties/null order deterministically by source IDs and
   recording a warning.
5. Leave the new variant and exercise unarchived unless a separate source indicates otherwise.

These arrays represent the stored current/next targets on the existing application path. Copy them
as targets without re-running progression, resetting reps again, or treating them as new historical
performance. Do not infer a new completed workout from `finished = true`.

Before importing, report missing owners/names, cross-owner links, null settings, invalid counts,
unequal array lengths, duplicate/null ordering, and values outside new validation limits. Reject
unresolved exceptions rather than truncating names, dropping array elements, or choosing silent
defaults. Review zero/null auto-increments explicitly: mapping them to off mode may change the old
rep-reset behavior. Review decimal conversion and source timestamp semantics before rounding or
timezone conversion. The implemented timezone default is `Europe/Stockholm`, with an explicit
IANA timezone override available; no separate timezone approval is required.

## 4. Preserve legacy history honestly

The old `History` stores a timestamp, average reps, average weight, set count, and exercise ID. It
cannot establish the individual sets or which saved session produced an entry. If one old exercise
becomes several variants, the historical variant can also be ambiguous.

Neither the legacy schema nor the current slider persists RPE. Leave the new variant's `last_rpe`
and `last_rpe_at` null on import, and never manufacture historical exercise ratings. New attempt
RPE starts null as well; migration `003_exercise_rpe.sql` performs no inferred backfill.

The legacy schema has no notes either. When the proposed notes columns are implemented, initialize
variant notes to null on import and leave existing workout/attempt notes null when adding the
columns. Do not infer notes from names or import bookkeeping. Skip-only importer reruns must
preserve any notes the user has subsequently entered on imported variants.

Propose a `legacy_history_entry` compatibility table containing:

- Owner and mapped base exercise, with a nullable variant reference only when attribution is clear.
- Original source exercise ID and timestamp, forming a unique source key for idempotent import.
- Original aggregate values, with an imported-aggregate marker and import provenance.
- A normalized occurrence timestamp using `Europe/Stockholm` by default or the supplied source
  timezone; retain the original timestamp value as well.

Expose these entries in exercise history as imported summaries. Do not label them as per-set workout
records or include them in a particular session's history without evidence. Include them in deletion
impact counts and the corresponding archive/merge/delete behavior.

The inspected browser submission path can send already increased weights and reset reps to
`SaveHistory`. Preserve those legacy values with provenance; do not attempt to reverse the formula
to guess actual performance. New workouts use the accurate set-level model going forward.

## 5. Import execution

### Audit and rehearse

Create a consistent backup and restore it into an isolated rehearsal database. Inventory every
application table and generate a proposed mapping/exception report. The report should show source
counts, duplicate-name groups, variant assignments, unassigned history, invalid values, and active
legacy sessions. No production data is changed by this review.

Use one shared Go importer with audit and apply modes, durable ID mappings, and per-owner
transactions. The CLI and protected internal UI call that same implementation. Reruns skip source
keys already mapped and import only unmapped records. Imported source rows are assumed immutable;
changed-row synchronization is outside scope. Write destinations and mappings in the same
transaction so interrupted runs resume without duplicates.

### Cut over

The simplest initial approach is a controlled write-maintenance window; its duration depends on the
audited data volume and rehearsal results.

1. Have users finish active legacy workouts before the write freeze where possible. Report any
   remaining active pointers; do not pretend the old data can reconstruct a complete new draft.
2. Block legacy and new workout writes, including requests from cached old clients, and take the
   final consistent backup/snapshot.
3. Run the reviewed import and validation against that frozen source.
4. Switch the application/API version to the new model while writes remain blocked.
5. Smoke-test reads, targets, session composition, imported history, and new-workout initialization.
6. Reopen writes only after validation. Require stale clients to refresh before allowing mutations.

Avoid dual-writing both target models. Keep legacy tables read-only through the retention period;
remove them in a separate later change, after every row has a destination or an explicit retained
legacy representation.

## 6. Validation and rollback

Before cutover, reconcile:

- Every source account and owned record, including definitions not attached to a session.
- Every session placement and its mapped variant, order, rep/weight targets, and settings.
- Every source history row exactly once, with aggregate values unchanged and attribution recorded.
- No unresolved name conflicts, orphan references, cross-owner links, or constraint violations.
- Decimal/timestamp conversions against their reviewed policies.
- Idempotency by rerunning the importer on the rehearsal database and observing no new records.

Test representative accounts through the new API as well as comparing database counts. Merging
duplicate base names means raw exercise counts need not match; the ID mapping must explain every
source row instead.

Before new writes reopen, rollback can switch back to the untouched old application/tables. After
users create new workouts or change targets, switching back alone would lose those new changes. At
that point prefer a forward fix, or a separately designed reverse migration that preserves all
post-cutover writes. A backup by itself does not solve that second case.

## 7. Decisions needed before execution

Name grouping and numbered variants are automatic. Missing rest defaults to zero, source timezone
defaults to Stockholm, and unused-exercise deletion is an explicit option. Before real execution,
resolve remaining unconfigured exercises excluded from cleanup, invalid settings, zero increments, active
legacy workouts, precision conversion, and ambiguous legacy history attribution. Audit
reports these exceptions automatically; they do not require reviewing ordinary duplicate names. No
production audit has yet established how many records, if any, need each policy. See the
[tool and bookkeeping design](porting-tool.md) for run status, resume behavior, and the email
allowlist.
