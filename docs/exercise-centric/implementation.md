# Implementation investigation and delivery plan

This document records findings from the current repository and proposes how to build the
[product requirements](decisions.md). Findings come from source inspection and the existing
[schema snapshot](../database-schema.md), not a new production-data audit.

The initial additive migrations and importer have since been implemented against isolated fixtures;
see [porting operations](porting-operations.md). The investigation below describes the legacy
application paths that still need replacement.

## 1. Current architecture

The backend uses Go `net/http`, `pgxpool`, and direct parameterized SQL. Routes are registered in
[`api/routes.go`](../../apps/server/api/routes.go); database handlers and queries live in
`apps/server/db`. Authentication uses a signed cookie and Redis session lookup. Redis stores
authentication/rate-limit state, not workout set records.

Svelte tracker components call the Go API through
[`dbFetches.ts`](../../apps/web/src/routes/tracker/dbFetches.ts). The same frontend object currently
represents settings, targets, and performed reps. Splitting those meanings affects both the database
and the API contracts.

### Findings that shape the change

| Finding                                                                            | Evidence                                                                                                                                                                                                 | Consequence                                                                                                                               |
| ---------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Adding session exercises creates new library rows                                  | `NewSession` and `AddExercisesToSession` call `AddMultipleExercises` in [`writes.go`](../../apps/server/db/writes.go)                                                                                    | Add APIs for selecting existing variant IDs; creation must be explicit and atomic                                                         |
| Session links hold configuration and mutable completion state                      | `SessionExercises`, `UpdateExercise`, and `CompleteSession` in [`reads.go`](../../apps/server/db/reads.go) and [`writes.go`](../../apps/server/db/writes.go)                                             | Move settings to variants and state to actual workouts                                                                                    |
| Active state is a text pointer to a saved session                                  | `SetActiveSession` and `CheckIfActive` in the same files                                                                                                                                                 | Replace it for new behavior with an active workout query and database uniqueness                                                          |
| Progression is calculated before sending performance                               | `buildProgressUpdate` in [`saving.ts`](../../apps/web/src/routes/tracker/saving.ts), called by `submitExercise` in [`track.svelte`](../../apps/web/src/routes/tracker/track.svelte)                      | Send actual performance; calculate progression on the server                                                                              |
| History consumes the progressed update payload                                     | `CallUpdateExercise` in [`handler.go`](../../apps/server/db/handler.go) passes the same payload to `UpdateExercise` and `SaveHistory`                                                                    | On this path, qualifying history can record increased weight/reset reps rather than actual work; investigate legacy values before porting |
| Completion consists of separate writes                                             | `CallUpdateExercise` separately updates exercise state, history, and last-run date                                                                                                                       | A partial failure or retry can leave inconsistent state; replace with one transaction and an immutable completion transition              |
| Edits are also separate writes and flatten weights                                 | `EditExercise`, `updateSessionSetCount`, `updateSessionWeight` in [`writes.go`](../../apps/server/db/writes.go); `applyExerciseEdit` in [`edit.ts`](../../apps/web/src/routes/tracker/edit.ts)           | Use target-set IDs and atomically reconcile active drafts; preserve independent set weights                                               |
| Partial rep changes are only local in the inspected tracker path                   | `handleRepCountIncrementation` mutates component state; `SendUpdate` runs on exercise completion in [`track.svelte`](../../apps/web/src/routes/tracker/track.svelte)                                     | Existing completed progress persists, but new per-set drafts need explicit durable storage and sync                                       |
| Set completion is a UI-local boolean                                               | `checked` and `onMount` in [`SetBlock.svelte`](../../apps/web/src/components/SetBlock.svelte)                                                                                                            | Persist pending/performed/skipped states; do not infer performance from prefilled targets                                                 |
| Finishing or quitting resets reusable plan state                                   | `CompleteSession` in [`writes.go`](../../apps/server/db/writes.go); `quitSession` in [`track.svelte`](../../apps/web/src/routes/tracker/track.svelte)                                                    | End an actual workout instead; preserve its completed records                                                                             |
| History contains aggregates only                                                   | `SaveHistory` and `ExerciseHistory`; current `History` schema                                                                                                                                            | Introduce per-set workout history; do not manufacture missing legacy set details                                                          |
| RPE is UI-local and never included in completion                                    | `ExerciseTrackScreen.svelte` passes `RIRValue` to `RPESlider.svelte` without binding it back or including it in `onSubmit`                                                                                | Use nullable attempt-level RPE and variant last-RPE storage from migration 003; wire recording/completion in the new application             |
| A finished-exercise query references a missing column                              | `CheckFinishedExercises` selects `Session.finished_ex_ids`, absent from the captured schema                                                                                                              | Replace this path with workout attempt state; assess current callers during implementation                                                |
| History JSON shapes also differ                                                    | `HistoricData` has no JSON tags in [`reads.go`](../../apps/server/db/reads.go), while `GetExerciseHistory` expects snake_case fields in [`dbFetches.ts`](../../apps/web/src/routes/tracker/dbFetches.ts) | Define and test a single explicit response contract during the rewrite                                                                    |
| REST data mixes numeric strings and numbers                                        | `ExInfo`, `ExerciseEdit`, and `ExerciseInfo` in backend/frontend                                                                                                                                         | Introduce explicit exercise/variant/workout IDs, numeric fields, targets, and recorded values                                             |
| No versioned SQL migrations or domain tests were found in the inspected repository | File inventory; the added schema introspection test is the only Go test found                                                                                                                            | Establish migration execution and meaningful domain/integration tests before cutover                                                      |

These are implementation and data-porting concerns, not claims that every historical row is wrong.
The browser's local completed display can retain actual values even when the server received next
targets, so checking only the UI would miss the history issue.

## 2. Backend boundaries

Keep the current Go/PostgreSQL stack. Separate request decoding/authentication, workout/library
operations, and SQL persistence. Pure progression logic should be independently testable and reused
by every completion path. Backend code must validate ownership, archived state, set states, mode,
and limits even if the client has already validated them.

The current `getValidatedMail` flow supplies an authenticated owner from the cookie and verifies a
Redis session. Continue using that boundary, with ownership predicates and composite foreign keys on
new records. Do not trust client-provided owner IDs or a browser-computed increase.

### Proposed operations

Route spelling is implementation detail; these are the required boundaries.

| Area           | Operations and important behavior                                                                                                    |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Library        | Search/browse exercises, create exercise with Standard variant, create/configure variants, assign backend-defined muscle-group codes |
| Metadata       | Return the code-defined muscle-group catalog with display labels and ordering for frontend selection                                 |
| Session plans  | Create/edit ordered variant references, attach existing variants, remove placements, delete plan                                     |
| Active workout | Start from plan or free mode, fetch/resume current workout, add/remove workout-only work                                             |
| Set recording  | Persist actual values and set states, add bonus sets, skip prescribed sets                                                           |
| RPE recording  | Set/clear nullable exercise-attempt RPE on a draft; validate integer 0–10; return the variant's last rating/time separately           |
| Notes          | Set/clear independent variant, active-workout, and draft-attempt notes; validate text/length and return authoritative notes/revisions |
| Configuration  | Update variant targets/settings and reconcile its unfinished active attempt immediately                                              |
| Completion     | Complete one attempt once; freeze notes/RPE and update variant last rating/time atomically; return performance and next targets separately |
| Workout ending | Flush accepted completions, discard unfinished attempts, end only the specified workout                                              |
| History        | Filter completed attempts by workout, variant, or parent exercise; include set kind, actual values, RPE, and attempt/workout notes    |
| Lifecycle      | Archive/restore, preview deletion impact, confirm delete, transfer/merge                                                             |

Use separate commands for editing variant configuration, recording a set, skipping a set, and adding
bonus work. A single untyped `currentProgress` payload would make intent ambiguous again. Return
updated revisions and authoritative state; distinguish not-found, validation, archived-resource, and
conflict responses.

Note operations address the variant, workout, or workout-exercise ID explicitly. Treat omitted
`notes` in a partial update as unchanged and null as clearing it. Normalize blank notes to null,
preserve internal line breaks, and validate the proposed 4,000-character limit consistently with
PostgreSQL `char_length`. Return current variant instructions separately from attempt notes.
Changing notes must not trigger progression, RPE updates, or target reconciliation. Migration
`004_notes.sql` supplies nullable note storage and length constraints; note APIs and UI remain
to be implemented.

### Transactions and concurrency

For this application's single-active-workout model, a simple initial strategy is to serialize
mutations for one user by locking their existing `User` row, then locking affected workout/variant
rows in a consistent order. Keep transactions short; do not hold locks while waiting on clients or
network calls. This also makes start, merge, delete, and new-placement checks race-safe.

- **Create:** exercise, Standard variant, and initial targets succeed together. Name uniqueness
  handles concurrent creation.
- **Start:** resume the active workout or create one; the partial unique index is the final guard
  against concurrent starts.
- **Edit:** update variant configuration and unfinished attempt snapshots in the same transaction,
  retaining actual performance.
- **Notes:** persist the field and its mutation receipt with the appropriate variant/workout
  revision check. Apply the draft/active editing boundaries without rewriting other note fields.
- **Complete:** validate the current revision and set states, calculate progression, update targets,
  freeze attempt notes and optional RPE, update the variant's last rating/time if supplied, and
  mark completed together. Read-only retries never increment again or replace committed notes/RPE.
- **End:** lock the specified active workout; keep completed attempts, remove unfinished drafts, and
  freeze workout notes while marking ended. Discard draft-attempt notes with their attempts.
  Do not reuse the old blanket reset of a saved session's `finished` flags.
- **Merge/delete:** perform ownership, active-workout, reference, and impact checks under the same
  lock used for mutation. Never move only part of a variant's history.

Use explicit set/attempt identity and optimistic revisions to reject stale writes from a second
device. Per-user serialization prevents concurrent database changes from interleaving, but revisions
are still needed to detect a request composed from stale state. Server responses must not silently
discard local work.

## 3. Offline recording and synchronization

Offline recording with later sync is required. Propose IndexedDB for the active workout snapshot and
pending operation queue; `localStorage` is synchronous and does not offer a transaction spanning
structured draft changes and their queued operations.

Persist each user action and its queue entry locally in one transaction before showing it as saved
on the device. Render from that durable local state. Keep client-generated UUIDs and stable
operation IDs so retries do not create duplicate sets or attempts.

### Operation flow

1. Load the active workout and available library data from the server while online.
2. Record changes locally immediately, including set values, optional exercise RPE, notes on
   variants/workouts/attempts, skips, bonus work, and allowed configuration edits.
3. Replay operations in dependency order when online, with their expected server revision and stable
   operation ID.
4. The server authorizes and applies each operation atomically with its receipt, returning the new
   revision and authoritative result.
5. Mark accepted operations acknowledged, then update the local snapshot without overwriting later
   local edits still queued.
6. Before exercise completion or workout ending, process preceding edits, including notes, in
   order. Retrying a completion after a lost response cannot progress twice.

Proposed initial boundary: starting a new workout and library creation/merge/delete require
connectivity. An already loaded active workout can be recorded offline, including adding cached
variants and bonus sets. This preserves server enforcement of one active workout; expanding offline
creation can be a later feature. Record this boundary in the UI, not as an unexplained failure.

Closing the browser must not clear the queue. Do not depend on a background sync event firing; also
replay on app startup, focus, and reconnection. Show local/pending/server-confirmed status and
errors. Another device sees server-confirmed changes after refetch; it cannot see work that has
never left the original device.

### Conflicts and finality

Use revision checks rather than last-write-wins for overlapping configuration or performance edits.
On conflict, retain the local pending operation, refetch server state, and require reconciliation
for conflicting intent. Do not replay an old end/completion over a newer workout or reopen an
attempt completed on another device.

A local completion is pending server synchronization until acknowledged. Preserve it locally and
prevent casual editing, but present synchronization conflicts clearly if the server rejects it. An
ended workout on the server rejects delayed draft mutations; recovery must preserve unsynced local
work for explicit resolution rather than silently resurrecting history.

Local data must be scoped to the signed-in account. Logging out or switching accounts must not
upload one user's queued work under another user's identity. Decide whether to retain or explicitly
discard unsynced work before clearing account storage.

## 4. Frontend impact

- Replace the shared target/performance arrays with explicit per-set targets, recorded values, kind,
  and state.
- Bind the RPE slider to nullable attempt state and persist explicit selection/clear actions.
  Keep its untouched default unreported. Include RPE in local durable drafts, sync, completion,
  and history; show the variant's previous rating/time separately. RPE-based adjustment is deferred.
- Add separate note inputs to variant editing, the active workout, and each draft attempt.
  Show saved variant instructions separately from the attempt's observations. Support clearing,
  multiline text, local durability, sync conflicts, and read-only historical notes after finalization.
  Keep workouts with notes accessible even when no exercise was completed.
- Replace index-based tracking identity with stable workout-exercise and set IDs. Variant ID
  identifies reusable configuration, not a particular performance.
- Show per-set target weights; the current exercise screen and edit UI assume one weight.
- Replace singular/free-text muscle-group fields with `muscleGroupCodes` arrays validated against
  the backend's code-defined catalog. Use its metadata response for selection labels/order; review
  AI-generated exercise drafts and existing program labels through the same code mapping. Store the
  selected codes on the exercise rather than in dictionary or join tables.
- Make prescribed set editing, bonus-set addition, and skipping visually distinct commands because
  their progression effects differ.
- Introduce library search and variant selection across manual creation, AI-generated plan review,
  saved-session editing, and free mode. Generated names must not silently create duplicates or merge
  distinct exercises.
- Replace client-authoritative `buildProgressUpdate` payloads with actual records and completion
  intent. A local preview may use equivalent rules, but the server result is authoritative.
- Revisit automatic workout ending: `submitExercise` currently ends the session when every listed
  exercise is finished. Explicit ending is the proposed default so users can add more free-mode or
  bonus work.

## 5. Delivery phases

### Phase 1: finalize the design

Resolve the meaningful choices in [remaining decisions](questions.md), review the proposed tables
and constraints, and establish the API payloads. Keep the product rules and acceptance scenarios as
the source for tests.

### Phase 2: implement against the new schema

1. Use the versioned additive migrations and isolated development/test database workflow;
   migrations 003/004 supply RPE and nullable notes on variants, workouts, and attempts.
2. Implement library/variant/target-set operations and owner/name constraints.
3. Implement saved-session variant references and the new workout lifecycle.
4. Implement transactional recording, immediate configuration edits, progression, completion, and
   ending.
5. Implement durable local drafts, idempotent sync, revision conflicts, and resume behavior.
6. Update tracker, creation flows, library UI, history, archive/delete, and merge operations.
7. Validate the [acceptance criteria](acceptance.md), including retry and rollback scenarios, before
   routing existing users into the new model.

The old schema and endpoints must not continue writing conflicting state for an account already
using the new model. During implementation, use new test accounts/data or an explicit version gate.
Do not silently maintain two competing target stores.

### Phase 3: port legacy data and cut over

Design and rehearse the shared importer after the new implementation is validated. Plan the
snapshot, durable ID mapping, automatic transformations, count reconciliation, and rollback boundary
before applying it. The [internal tool design](porting-tool.md) defines audit/run/resume entry
points and the hardcoded email access policy. Preserve the original data through validation; do not
combine importing and dropping legacy tables into one step.

## 6. Legacy porting risks

| Legacy data                           | Proposed interpretation / challenge                                                                                                                                                                                                                                                                             |
| ------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Exercise`                            | Parent metadata and auto-increment exist, but repeated creation may have produced duplicate names. Trimmed, case-insensitive names are now grouped automatically per owner, with separate numbered variants preserving progression.                                                                             |
| `SesExercise`                         | Carries targets/settings and completion state. The same old exercise can have different session-specific settings, so mapping one old exercise to one variant is not always sufficient.                                                                                                                         |
| `Session`                             | Becomes a saved plan; `date_last_ran` is not a workout record.                                                                                                                                                                                                                                                  |
| `User.active_session`                 | Identifies a plan, not a distinct attempt. Decide how to close or carry active legacy work at cutover.                                                                                                                                                                                                          |
| `History`                             | Only timestamp and aggregates, with no session link or per-set records. Some writes may contain progressed targets. Preserve provenance; do not invent sets, workouts, or corrected historical values. If one old exercise becomes several variants, its historical variant attribution may also be unknowable. |
| `Set`                                 | Exists in the schema but was not referenced by the inspected Go queries. Inspect actual use and data during the porting phase before assuming it is empty or reusable.                                                                                                                                          |
| Old timestamps                        | `timestamp without time zone` does not establish the intended timezone. Resolve source semantics before converting to `timestamptz`.                                                                                                                                                                            |
| Missing/null values and array lengths | Existing columns allow nulls and uneven arrays; new set-row constraints require validation and an explicit repair policy.                                                                                                                                                                                       |

The [legacy porting plan](migration.md) proposes a compatibility representation for aggregate
history, linked to the base exercise and to a variant only when attribution is clear. It also
details conservative variant mapping, rehearsal, cutover, and rollback. Exact source-data repair
policies remain to be reviewed before execution; do not force aggregates into fabricated
`workout_set` rows.

## 7. Verification scope

This investigation changed documentation only. Source findings were traced through handlers, SQL
functions, tracker submission, and editing code. The live database was not modified or queried again
for this review. Formatting, relative links, and consistency with the agreed decisions should be
checked before handing off the spec.
