# Remaining decisions and design defaults

The core product rules are settled. Confirmed behavior is in [product requirements](decisions.md).
This document records the implemented reference catalog and proposed defaults for implementation
review.

## Muscle-group catalog: implemented

The supplied 26 muscle groups are defined in
[`predefined/muscle_groups.go`](../../apps/server/predefined/muscle_groups.go). The `predefined`
package also holds the exercise-name suggestions in `exercises.go`. `MuscleGroups()` returns a copy of
the list, preserving the supplied codes, labels, and `sort_order` values. Shared sort-order values
are intentional; ties retain the list's declared order.

Exercises select zero or more codes from this catalog and store them in an array. There is no
primary/secondary distinction, database seed data, or muscle-group table. The read-only metadata
endpoint and exercise validation are implemented.

## Exercise-name suggestions: implemented

`predefined.ExerciseNames()` returns ten common gym exercise names for autocomplete. The
read-only `/api/training/catalog` response exposes them as `exerciseNames`, alongside
`muscleGroups`. This metadata creates no database records. Users select a suggestion to
configure a private exercise and Standard variant, or select a matching personal exercise to
review and reuse its variants. Personal names take precedence over duplicate suggestions.

## Proposed implementation defaults

These choices make the specification concrete without reopening settled behavior. They remain
adjustable during technical review.

| Area                              | Proposed default                                                                                                                                                                                  |
| --------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Late manual target edit           | Keep the edited weight exactly. Suppress automatic weight increase and its associated rep reset for the whole completion, preserving all-or-nothing increases; performed reps still carry forward |
| Progression comparison            | Both average and all-sets modes use `>=`; average is the default mode                                                                                                                             |
| Initial targets                   | Reps start at the configured threshold; creation accepts a common initial weight and allows per-set adjustment                                                                                    |
| Free-mode targets                 | Capture current targets when adding the variant, then reconcile intentional configuration edits immediately                                                                                       |
| New workout contents              | A variant appears at most once in both saved-session and free-mode workouts                                                                                                                       |
| Added exercises                   | An added variant uses ordinary prescribed-set progression; only extra bonus sets are excluded                                                                                                     |
| Warm-up work                      | No special warm-up set type in this version; model extra non-progressing sets as bonus work                                                                                                       |
| Workout ending                    | Confirmed stage-specific behavior is recorded in [application transition planning](application-transition.md#workout-ending)                                                                     |
| Offline boundary                  | Recording an already loaded active workout works offline; starting a new workout and library creation/merge/delete require connectivity                                                           |
| Names                             | Trim surrounding whitespace, compare case-insensitively, reserve archived names, and permit renaming Standard                                                                                     |
| Variant archiving                 | Same hidden-from-selection and restore/remove behavior as archiving its parent; offer an archived-library filter                                                                                  |
| Archiving during active use       | Block archiving variants involved in the active workout until it ends, avoiding a new in-workout state transition                                                                                 |
| Saved-session deletion during use | Block deleting the source plan while its workout is active; later deletion retains workout history                                                                                                |
| Merge collisions                  | Block if a saved session contains both source and destination; remove one placement before retrying                                                                                               |
| Last variant                      | Deleting the final variant uses the parent exercise deletion flow; moving/merging it away removes the now-empty parent as part of that operation, disclosed in the preview                        |
| Rest duration                     | Keep an optional variant-level duration, defaulting to zero; no new timer behavior is required for this migration                                                                                 |
| Notes format and limit            | Optional plain text, maximum 4,000 characters per field; blank means null; preserve internal line breaks                                                                                             |
| Notes editing                     | Variant notes remain editable; attempt notes freeze on exercise completion and workout notes freeze on ending, consistent with existing finality                                                   |
| Notes independence                | Show reusable variant instructions separately; do not automatically copy or write back workout/attempt notes                                                                                        |
| Notes on merge                    | Keep destination variant notes, show source variant notes in the preview, and preserve each historical attempt's notes                                                                               |
| Empty workouts with notes         | Retain and keep accessible after ending even without completed attempts; do not count them as historical exercise entries                                                                           |
| History count for deletion        | Count completed exercise attempts, show recorded-set count separately, and later include legacy aggregates explicitly                                                                             |
| Confirmation flow                 | Show archive/merge alternatives and impact count first, then require typed exercise name and a final destructive-action confirmation                                                              |

## Resolved during this investigation

- Configuration edits take effect immediately in unfinished exercises.
- Prescribed-volume reduction follows the confirmed
  [stage-specific rule](application-transition.md#set-recording-and-weights).
- A later target-weight edit preserves the actual weight already recorded and keeps the deliberate
  new target exactly, even when reps qualify for automatic progression.
- Saved-session list/order edits apply to the next workout; active workout contents remain
  independent.
- Offline recording is required; IndexedDB and an ordered operation queue are the proposed
  mechanism.
- One workout is active per user; multiple devices resume that same workout.
- Bonus sets do not affect progression, and any skipped prescribed set blocks automatic increases.
- Saved-session merge collisions do not need further product exploration; use the conservative
  default above.

## Porting implementation decisions

Confirmed: duplicate names group automatically into base exercises with separate numbered variants;
the importer is repeatable and skips previously mapped source records. Imported old rows are assumed
not to change, so no changed-row synchronization is required. The internal tool uses existing
sign-in with a hardcoded server-side email allowlist. See
[the implementation design](porting-tool.md).

The configured internal allowlist is `gustavengvall@yahoo.se`. Source-data policies for
unconfigured exercises, invalid values, active legacy sessions, history attribution, and
precision conversion still need definition before production execution; audit detects these
automatically. Ordinary duplicate names no longer need manual review. See also
[legacy risks](implementation.md#6-legacy-porting-risks).

Implemented import defaults: missing rest duration is zero and omitted/blank source
timezone is `Europe/Stockholm`. The explicit unused-exercise checkbox/CLI option deletes
unmapped exercises without placements or retained Set rows. A separate explicit history
option also deletes their unmapped legacy History rows; without it, orphans with history
remain setup issues. Audits preview exercise/history deletion counts and preserve source rows.
