# Database design

**Status:** Proposed physical model implementing the [product requirements](decisions.md). This is a
design document; the initial executable migrations are in `apps/server/migrations`. New tables are
additive so implementation can precede
legacy data porting.

## 1. Design principles

- Keep the existing `public."User"` account and email ownership model for this change; replacing
  authentication identity is a separate project.
- Use separate records for library configuration, saved-session composition, active work, and
  completed performance.
- Persist each target set and workout set as a row. Arrays cannot clearly express skipped sets,
  bonus sets, stable set identity, and independent target edits.
- Derive prescribed set count from target-set rows, avoiding a second count that can drift.
- Use one workout/performance model for active drafts and new completed history, distinguished by
  explicit state. Legacy aggregate-only entries need a separate compatibility representation, as
  proposed in the [porting plan](migration.md#4-preserve-legacy-history-honestly).
- Let the server own progression, ownership validation, and transaction completion. Clients send
  recorded work and edit intent, not precomputed future targets.

## 2. Conventions

Proposed new tables use lowercase snake_case names. Existing quoted legacy tables remain during
implementation and porting.

| Concern                | Proposed representation                                                                              |
| ---------------------- | ---------------------------------------------------------------------------------------------------- |
| New entity IDs         | UUID; clients may generate IDs for offline-created workout items                                     |
| Ownership              | `owner_mail varchar(255)` referencing `public."User"(mail)`                                          |
| Time                   | `timestamptz`; server-assigned lifecycle timestamps, optional client recording time for offline work |
| Weight                 | `numeric(8,2)` in kilograms; fixed-point or decimal-safe arithmetic in Go                            |
| Counts and order       | Integer; nonnegative reps, positive set positions, nonnegative weights                               |
| RPE                    | Nullable `smallint`, checked 0–10 inclusive; integer scale matching the current slider               |
| Notes                  | Nullable `text` on variants, workouts, and attempts; proposed maximum 4,000 characters per field      |
| Modes and states       | Text with CHECK constraints; no unconstrained free-text states                                       |
| Optimistic concurrency | Integer `revision` on mutable aggregates, checked on write                                           |
| Archive                | Nullable `archived_at`; archiving retains relationships and history                                  |

The ID and type choices are technical proposals. Existing numeric IDs will need explicit mapping
during legacy porting. Keep decimal weights exact across JSON and Go; do not let binary
floating-point rounding decide future weights.

## 3. Library tables

| Table                | Main columns                                                                                                                                            | Purpose and constraints                                                           |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| `exercise`           | `id`, `owner_mail`, `name`, `muscle_group_codes`, `archived_at`, `revision`, `created_at`, `updated_at`                                                 | Personal base exercise; unique owner + case-insensitive name                      |
| `exercise_variant`   | `id`, `owner_mail`, `exercise_id`, `name`, nullable `notes`, `progression_mode`, `rep_threshold`, `auto_increment`, `rest_seconds`, nullable `last_rpe`, nullable `last_rpe_at`, `archived_at`, `revision`, timestamps | Shared configuration/instructions, progression, and last reported effort; unique exercise + case-insensitive variant name |
| `variant_target_set` | `id`, `owner_mail`, `variant_id`, `position`, `target_reps`, `target_weight`                                                                            | Stable prescribed-set identity; unique variant + position                         |

`progression_mode` is `average`, `all_sets`, or `off`. A variant must have at least one prescribed
set. Exercise creation and its initial `Standard` variant/target rows are one transaction. Deleting
the final variant uses the parent exercise deletion flow as a design default. Moving or merging the
final variant away removes the empty source parent in that same transaction, disclosed before the
operation; no historical performance is deleted by that cleanup.

Reuse existing limits initially: 1–20 prescribed sets, threshold 1–99, target weight 0–9999 kg,
enabled increment 0.25–99 kg, and names up to 100 characters. Off mode bypasses progression rather
than encoding it as a zero increment. Bonus volume needs a separate total-workout bound; it must not
silently consume the prescribed-set limit.

Stable target-set IDs allow adding/removing/reordering configuration without mistaking a recorded
set for another position. Deleting a target-set row must never cascade to recorded workout sets.

### Muscle-group codes

The predefined catalog is implemented in
[`predefined/muscle_groups.go`](../../apps/server/predefined/muscle_groups.go). Each entry has
`code`, `label`, and `sort_order`; shared sort-order values retain their declared list order. The
frontend will obtain this catalog through a read-only metadata response so it shares the backend's
codes, labels, and ordering.

Store selected codes directly on `exercise.muscle_group_codes` as `text[] NOT NULL DEFAULT '{}'`.
For example, an exercise can store `['chest', 'triceps']`; no selection is an empty array. Variants
inherit these groups through their parent exercise. There are no muscle-group dictionary or join
tables, database seed rows, or foreign keys for individual muscle-group codes.

The API represents the field as `muscleGroupCodes: string[]`. Validate every exercise creation/edit
and import against the backend catalog, rejecting unknown codes, duplicates, and null elements.
Treat the array as a selection set without primary/secondary ranking; return it in catalog order.
Catalog membership is a backend validation rule, not a database enum or duplicated hardcoded SQL
list. Any other application writer must use the same validation.

Labels and catalog order can change without rewriting stored exercise codes. Codes remain stable;
removing or renaming a code already in use requires an explicit compatibility/data-migration policy.
Adding a catalog entry requires a backend code change, not a schema migration.

## 4. Saved sessions

| Table              | Main columns                                               | Purpose and constraints                                                                  |
| ------------------ | ---------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `training_session` | `id`, `owner_mail`, `name`, `revision`, timestamps         | Reusable named plan                                                                      |
| `session_variant`  | `id`, `owner_mail`, `session_id`, `variant_id`, `position` | Ordered placement; unique `(session_id, variant_id)` and unique `(session_id, position)` |

Configuration is read through the referenced variant. There are no weights, rep targets, set counts,
or completion flags on `session_variant`. Update ordering transactionally, using a deferrable
position constraint or temporary positions during reordering.

## 5. Workouts and set records

| Table              | Main columns                                                                                                                                                                                                                           | Purpose and constraints                                                                                   |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| `workout`          | `id`, `owner_mail`, nullable `source_session_id`, nullable `notes`, `origin`, `status`, `revision`, `started_at`, nullable `ended_at`                                                                                                                    | Actual training occasion and its notes; `origin` is `session` or `free`; `status` is `active` or `ended` |
| `workout_exercise` | `id`, `owner_mail`, `workout_id`, `variant_id`, `position`, `state`, `origin`, nullable `notes`, captured progression settings, `config_revision`, nullable `recorded_rpe`, nullable `completed_at`                                                                               | Draft or completed attempt, including its own notes and effort; `origin` distinguishes planned and added work |
| `workout_set`      | `id`, `owner_mail`, `workout_exercise_id`, nullable `target_set_id`, `position`, `kind`, `state`, `target_reps`, `target_weight`, nullable `recorded_reps`, nullable `recorded_weight`, nullable `recorded_at`, `late_weight_override` | Independent target and actual values, with explicit prescribed/bonus and pending/performed/skipped states |

A partial unique index on `workout(owner_mail)` where `status = 'active'` enforces one active
workout. `source_session_id` may be null for free mode or after a saved session is deleted; `origin`
retains how the workout started. Session list/order edits affect future workouts only; active
workout contents are independent.

### Notes storage and lifecycle

Add a nullable `notes text` column to `exercise_variant`, `workout`, and `workout_exercise`.
Each row owns one optional plain-text note. A separate notes table is unnecessary for this model.
New fields default to null, including on existing rows and imported variants. These note columns
are specified here but are not yet present in executable migrations; add a new additive migration
without changing checksums of already versioned migrations.

Proposed initial validation: at most 4,000 characters per field, using PostgreSQL `char_length`
for a matching database CHECK constraint. Normalize empty or whitespace-only input to null in the
API; preserve internal line breaks and reject oversized input rather than truncating it. Render
notes as plain text. In partial updates, omitted `notes` means unchanged and explicit null clears
the field.

Variant notes are current reusable instructions. Workout notes describe the occasion. Attempt
notes describe its recorded experience and remain with that attempt in history. Keep these fields
independent: no automatic copying, reconciliation, or write-back between them. Historical views
must distinguish any current variant instructions from the attempt's own historical note.

Use authenticated ownership and the containing aggregate's revision/operation receipt for note
writes, just as for other edits. Variant notes can be edited through variant operations; workout
notes can be edited while active; attempt notes can be edited while draft. The proposed finality
policy freezes attempt notes on completion and workout notes on ending. Synchronize preceding
note edits before these transitions, reject delayed writes to finalized records, and preserve
unsynced input for conflict resolution.

Removing a draft attempt removes its note. Completion preserves the attempt note, and ending
preserves workout notes alongside completed history. Archiving retains notes. A variant move
retains its note; a merge retains destination variant notes and moves each source attempt's note
with its history. Deleting an attempt or variant removes its owned notes with its row; surrounding
workout notes remain. Expose source variant notes in the merge preview before source removal.

### Exercise-level RPE

`workout_exercise.recorded_rpe` stores the attempt's optional integer 0–10 rating. It may be set or
cleared while the attempt is a draft and becomes immutable on completion. `NULL` is unreported;
zero is a real rating. Do not populate it from the slider's initial value or a variant's previous
rating. Completed attempts are the source of truth for historical RPE.

`exercise_variant.last_rpe` and `last_rpe_at` cache the latest rated completed attempt's rating and
server-assigned `completed_at`, respectively. Both are null until a rated completion exists; a
CHECK constraint requires them to be null or present together. A draft rating never changes them.
An unrated completion retains both previous values so callers can distinguish an older rating
using its timestamp. Do not copy this cache into a new attempt's recorded performance.

The completion transaction maintains the cache; the SQL migration only supplies storage and
constraints. Select the latest rated completed attempt by `completed_at DESC, id DESC` when
rebuilding it after a history-preserving merge. Clear both fields if no rated completed history
exists. Moving a variant keeps its cache. The existing variant/completion-time history index
supports this lookup. RPE-based target adjustment remains future application work.

Migration `003_exercise_rpe.sql` adds these nullable fields without changing earlier migration
checksums. Existing attempts and imported variants start with no RPE; legacy tables stay unchanged.

### Attempt and set states

`workout_exercise.state` is `draft` or `completed`. History queries include only completed attempts,
even while their surrounding workout remains active. Ending a workout removes unfinished attempts
and their draft sets while retaining completed attempts.

`workout_set.kind` is `prescribed` or `bonus`; `state` is `pending`, `performed`, or `skipped`. Zero
recorded reps are a valid recorded value, not a substitute for missing input. Performed rows require
actual reps and weight. Skipped rows retain their captured target but have no invented actual
performance.

Store captured progression mode, threshold, and increment on the attempt. They track intentional
configuration edits until completion, then explain the rule used for the recorded workout. Automatic
progression never rewrites completed set values. Current names are joined through the
variant/exercise, not frozen as historical labels.

`late_weight_override` is a boolean set by an explicit target-weight edit after that set has
recorded performance. Its proposed effect is to suppress automatic increase/reset for the attempt
while retaining the deliberate target. Do not infer this flag merely from target and recorded weight
differing.

Target-set references are optional provenance. Copy target values into workout rows and explicitly
retain `kind`; a deleted configuration row must not accidentally turn historical prescribed work
into bonus work. If a merge removes source target rows, clear the provenance reference while
preserving the workout's set values and kind.

### Uniqueness and merges

For new workout contents, reject a variant already present, including one already completed in the
same active workout. Perform this check under the workout/owner lock. Do not add a blanket unique
`(workout_id, variant_id)` constraint: a later history-preserving merge may legitimately produce two
historical attempts of the destination variant in one workout.

Saved sessions do enforce uniqueness. The merge default is to reject collisions before any changes,
letting the user remove a conflicting placement first.

## 6. Ownership and database integrity

Carry the owner through library, session, workout, and child rows. Use unique `(owner_mail, id)`
parent keys and matching composite foreign keys so cross-owner references cannot be inserted
accidentally. Resolve the authenticated owner server-side; never accept it as authoritative from the
request body.

Validate source target-set references against the attempt's variant inside the transaction. Every
merge, configuration edit, draft write, completion, and deletion must enforce the same ownership
rules. Inspect and deliberately configure privileges/RLS for the actual backend role during
implementation; do not assume signed-cookie identity automatically becomes a Supabase JWT claim.

Useful indexes include owner/name for library search, variant/position for targets, session/position
for plans, workout/position for attempts, and variant/completion time for history. Add indexes to
referencing foreign keys used by delete/merge checks; primary keys alone do not cover these access
paths.

## 7. Immediate configuration edits

In one transaction, update the variant and its target rows, then reconcile any unfinished active
attempt using it. Update its captured progression settings and applicable target values immediately.
Already recorded actual values remain intact.

- Adding prescribed sets adds required pending sets to the unfinished attempt.
- Removing prescribed sets removes unrecorded excess rows; recorded excess rows become bonus work
  and stay in history on completion.
- Weight edits update future targets immediately. If the set has already been performed, preserve
  its recorded weight and mark a late weight override on the active set. Keep the edited target
  exactly; proposed all-or-nothing policy suppresses automatic increase/reset for that completion.
  Weight edits made before recording do not trigger this exception.
- Threshold, increment, and mode edits become the settings used by that unfinished attempt's
  completion.
- Completed attempts are never reconciled to later configuration.

Do not identify these edits only by array position: stable set IDs prevent moving or deleting one
set from editing another's actual performance. Configuration edits are persistent even if the
current exercise is subsequently abandoned.

## 8. Completion and ending

Exercise completion is one transaction that validates set states, retains performance, calculates
next prescribed targets, freezes its notes and optional RPE, updates the variant's last reported
RPE/time when present, and marks the attempt completed. Bonus sets never participate in progression.
Skipping any prescribed set blocks increases; skipped targets keep weight and reset
reps. Reject ambiguous pending prescribed rows or require explicit skip marking before completion.

Completion retries return the committed result without applying progression again. A late draft
update or configuration reconciliation cannot reopen a completed attempt. End-workout retries
similarly cannot reset another workout's state.
Retries also cannot change committed notes, RPE, or the last-rating timestamp.

Ending changes the workout to `ended`, freezes its notes, removes unfinished drafts and their notes,
and preserves completed history. Closing the app changes no server lifecycle state. Empty ended
workouts without notes can be retained and hidden from history as a default. Workouts with notes
remain accessible even without completed attempts; they do not count as historical exercise entries.

## 9. Offline operation receipts

Propose a small `mutation_receipt` table with `(owner_mail, operation_id)` as its key, operation
kind, target ID, payload fingerprint, accepted revision, and commit time. It makes replayed offline
writes idempotent. Apply the domain mutation and receipt in the same transaction.

Use the same operation ID on retries. Reject reuse of an ID with a different payload. Store minimal
receipt metadata, not duplicated set performance; deleting history must not leave a second copy in
receipt payloads. Retention must outlast the supported offline retry window; expiry must return an
explicit recovery outcome rather than replay old work as new.

## 10. Deletion and merge boundaries

| Operation                        | Database effect                                                                                                                        |
| -------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| Archive exercise or variant      | Set archive timestamp; retain related rows                                                                                             |
| Remove saved-session placement   | Delete placement only                                                                                                                  |
| Delete saved session             | Delete its placements; preserve workouts with source reference cleared; active-session policy is listed in remaining decisions         |
| Delete variant                   | After reference/active-workout checks and confirmation, delete its targets, attempts, and their sets                                   |
| Delete base exercise             | Apply variant deletion to all children, then delete the parent, including its muscle-group code array                                  |
| Move variant to another exercise | Change its parent while retaining variant ID, configuration, history, and session links; resolve destination name conflicts explicitly |
| Merge variant into another       | Move history and saved-session references; preserve destination targets; remove source configuration after moving references           |

Use RESTRICT for saved-session-to-variant references. Cascade from an explicitly deleted
exercise/variant to its configuration and affected history, and from workout attempts to sets. Never
cascade from configuration target-set deletion to workout performance or from library deletion to
the surrounding workout.

Recheck active-workout and saved-session references in the final deletion/merge transaction.
Deletion impact is the number of completed exercise attempts plus any later-supported legacy
aggregate entries, with recorded set count separately available. A stale preview must not authorize
deleting newly added history without showing the changed count.

Legacy aggregates, ID mapping, and cutover mechanics belong to the later porting phase described in
[implementation](implementation.md#6-legacy-porting-risks).

## 11. Porting bookkeeping

The [porting tool](porting-tool.md#4-durable-bookkeeping) adds private run, account-progress,
source-group, source-key mapping, and issue records. These are independent of user-editable names
and survive destination renames, merges, and deletion, preventing reruns from duplicating or
resurrecting imported data. Do not expose this metadata to ordinary browser database clients.
