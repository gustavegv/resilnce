# Exercise-centric workouts

**Status:** Product requirements consolidated. The muscle-group catalog, additive database
migrations, and initial shared importer with CLI, worker, and allowlisted internal page are
implemented. The new application behavior and production cutover remain to be implemented.

Build a personal exercise library whose variants carry training configuration and progression across
saved sessions and free-mode workouts. Preserve actual performance independently from future
targets, and make active workouts durable across app closure, devices, and offline use.

## Read the specification

| Document                                          | Purpose                                                                                   |
| ------------------------------------------------- | ----------------------------------------------------------------------------------------- |
| [Product requirements](decisions.md)              | Agreed behavior, progression rules, editing, history, and deletion                        |
| [Database design](schema.md)                      | Proposed tables, relationships, ownership, and integrity constraints                      |
| [Implementation investigation](implementation.md) | Findings from the current code, transaction boundaries, offline sync, and delivery phases |
| [Legacy porting plan](migration.md)               | Proposed table mapping, preservation rules, rehearsal, cutover, and rollback              |
| [Porting tool design](porting-tool.md)            | Shared importer, automatic audit, allowlisted overview, and resumable jobs                |
| [Porting operations](porting-operations.md)       | Implemented commands, setup, access policy, limitations, and verification                 |
| [Acceptance criteria](acceptance.md)              | Scenarios the implementation must satisfy                                                 |
| [Remaining decisions](questions.md)               | Unresolved product choices and explicit design defaults                                   |

Product requirements reflect the latest answers in the design discussion. Database layout, API
shape, and implementation choices are proposals unless identified as existing behavior. Remaining
decisions are collected in one place rather than repeated throughout the specification.

## Terminology

| Term            | Meaning                                                                                                                                |
| --------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| Exercise        | A user's named lift, such as Bench Press, with shared muscle groups                                                                    |
| Variant         | A configuration and progression track, such as Heavy or High Reps                                                                      |
| Session         | A saved, ordered workout plan referencing variants                                                                                     |
| Session variant | A `session_variant` linking row that selects a reusable variant and its order in a saved session; it does not own or copy that variant |
| Workout         | One actual training occasion, either based on a session or started in free mode                                                        |
| Workout exercise | One attempt at a variant within a workout; starts as a draft and becomes part of exercise history on completion                       |
| Prescribed set  | A set expected by the variant's current configuration                                                                                  |
| Bonus set       | Work recorded for this workout without changing prescribed volume or progression eligibility                                           |
| Target          | What to aim for; separate from what was actually performed                                                                             |

## Delivery order

1. Settle the schema and remaining behavior decisions.
2. Implement and validate the new application behavior against the new schema.
3. Design, rehearse, and execute legacy data porting, then cut over existing users.

The [current database schema](../database-schema.md) remains the reference for legacy tables.
Migration risks are investigated now so the new design can accommodate them; data-porting rules and
execution remain a later phase.

## Implementation handoff

The agreed product model is sufficient to begin implementation. Keep the table name
`session_variant`: it links a session to a reusable exercise variant, whose ownership remains with
the parent exercise. Several sessions can reference the same variant and share its progression.

Read product requirements first, then the database design and acceptance criteria. Use the
implementation investigation for current code paths and the porting-tool/legacy-porting documents
for importer behavior. Explicit user decisions take precedence over proposed technical defaults;
ordinary database and code-organization choices can be resolved during implementation and recorded
in the spec.

The initial deliverable includes executable, additive SQL migrations and meaningful database tests for the
new domain, legacy compatibility storage, and private import bookkeeping. Use an explicitly
configured isolated test database for migrations and write tests. Do not copy the schema-inspection
test's fallback to the application's `DATABASE_URL` or `.env` into tests that write data.

Next implement library/variant/session/workout operations and the remaining application and
offline behavior. The initial audit/apply importer and allowlisted interface are developed against
fixtures while application work continues;
existing users cut over only after the new application and importer are validated together.

Migration `003_exercise_rpe.sql` adds optional exercise-attempt RPE for history and last reported
RPE/time on each variant. Slider persistence and transactional completion/cache updates remain part
of the new application implementation; RPE-based exercise adjustment is deferred.

Optional notes are specified on variants (reusable instructions), workouts (occasion context), and
workout exercises (attempt observations). See [requirements](decisions.md#notes) and
[storage/lifecycle](schema.md#notes-storage-and-lifecycle). Notes storage migrations and application
behavior remain to be implemented; the current executable migrations have no note fields.

The muscle-group catalog is available in
[`predefined/muscle_groups.go`](../../apps/server/predefined/muscle_groups.go). Internal access is
restricted in code to `gustavengvall@yahoo.se`, with verified email and a live login session required.
The importer defaults missing rest to zero and blank source timezone to Stockholm; optional
unused-exercise deletion is described in [operations](porting-operations.md).
Before a production import, resolve remaining audit-detected source exceptions, precision
interpretation, and active legacy-workout handling. Those do not block schema/code development, but
the importer must report affected accounts as incomplete instead of silently choosing destructive
repairs.
