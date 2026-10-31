# Acceptance criteria

These scenarios define the intended verification work once implementation begins. They are not
claims that tests already exist or pass. Product rules are in [requirements](decisions.md);
scenarios depending on a proposed default or unresolved question are marked accordingly.

## Library and reuse

| ID  | Scenario                                                 | Expected result                                                                                                  |
| --- | -------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| L1  | Create Bench Press                                       | One personal exercise and its Standard variant are created together                                              |
| L2  | Create `bench press` for the same owner                  | Case-insensitive duplicate rejected; another user may use the same name                                          |
| L3  | Add existing Heavy Bench to Push and Full Body           | Both placements reference the same variant and share targets                                                     |
| L4  | Add Heavy Bench twice to one session                     | Duplicate rejected; Heavy Bench and Light Bench together are allowed                                             |
| L5  | Rename an exercise or variant after completing workouts  | Historical screens show its current name; recorded performance is unchanged                                      |
| L6  | Choose no muscle groups or several backend-defined codes | Store an empty code array or the selected valid codes on the exercise; reject duplicate, unknown, and null codes |
| L7  | Change a catalog display label/order in backend code     | Existing exercise code arrays remain valid; selection UI uses the backend catalog's updated labels/order         |

## Progression and recorded performance

| ID  | Scenario                                                                                 | Expected result                                                                                                        |
| --- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| P1  | With no late manual weight override, complete prescribed reps `12, 11, 10`, threshold 11 | Average mode qualifies; next reps become `6, 6, 6` and every prescribed weight increases once                          |
| P2  | Same performance with threshold 12                                                       | No increase; next reps are `12, 11, 10`                                                                                |
| P3  | Threshold 11 with reps `12, 11, 10` in all-sets mode                                     | No increase because one prescribed set is below threshold                                                              |
| P4  | With no late manual weight override, qualify with weights `40, 50, 50` and increment 2.5 | Next targets are `42.5, 52.5, 52.5`; workout history still shows `40, 50, 50`                                          |
| P5  | Add a bonus set with very high or very low reps                                          | It appears in history but does not change either progression check or prescribed volume                                |
| P6  | Perform two of three prescribed sets and explicitly finish                               | No increase regardless of reps; skipped set retains weight and gets reset reps; all three targets remain for next time |
| P7  | Complete an exercise in off mode                                                         | Performed reps become next targets without automatic increase/reset; manual target-weight edits persist                |
| P8  | Revisit a completed exercise after progression                                           | Display actual recorded work; do not show increased next weights as weights lifted                                     |
| P9  | Retry completion after its response is lost                                              | Exactly one completion and one progression update; no duplicate history                                                |
| P10 | End a workout with one completed and one partial exercise                                | Keep the completed history/progression; discard the unfinished exercise from history without progressing it            |
| P11 | Record zero reps, leave another set pending, skip another                                | These states remain distinct and progression eligibility is evaluated accordingly                                      |

## RPE recording and history

| ID | Scenario | Expected result |
| --- | --- | --- |
| R1 | Complete an exercise with RPE 8 | Attempt history stores 8; variant last RPE is 8 with that attempt's completion time, atomically with progression |
| R2 | Complete the same variant from another session with RPE 6 | Its last RPE becomes 6; the earlier attempt still shows 8 |
| R3 | Complete without choosing RPE, including when the slider displays its initial value | Attempt RPE is null; previous variant rating/time remain unchanged, or both stay null if never rated |
| R4 | Explicitly report 0 or 10; submit a fraction or a value outside 0–10 | Accept the integer endpoints; reject invalid API input rather than treating it as missing or rounding it |
| R5 | Select or clear draft RPE, close the app, resume, and sync offline work | Explicit nullable rating survives local/server durability; variant last rating changes only on accepted completion |
| R6 | Abandon a draft with RPE selected | No historical rating or variant last-RPE update is created |
| R7 | Retry a completion or send a delayed draft rating after completion | Return the committed result; never replace the frozen rating, advance its timestamp, or progress twice |
| R8 | Merge variants with rated history | Keep all attempt ratings; destination last rating/time reflect the latest rated completion in combined history |
| R9 | Import legacy data or apply migration 003 to existing data | RPE stays null; no rating is inferred and existing targets/performance remain unchanged |
| R10 | Complete otherwise identical performances with different RPE | Existing progression rules give the same targets; automatic RPE adjustment remains deferred |

## Notes

Editing boundaries, text length, and empty-workout visibility below use the proposed defaults in
[remaining decisions](questions.md#proposed-implementation-defaults).

| ID | Scenario | Expected result |
| --- | --- | --- |
| N1 | Reuse a variant with notes in saved-session and free-mode workouts | Show the same reusable instructions; each new workout and attempt starts with its own null notes |
| N2 | Set distinct notes on a variant, a workout, and one draft attempt | Save all three independently; note changes do not alter targets, performance, RPE, or progression |
| N3 | Edit variant notes after completing an attempt | Current instructions change; saved workout and attempt notes remain unchanged |
| N4 | Complete an attempt with notes, then end the workout with notes | Attempt notes freeze at completion; workout notes freeze at ending; both remain available in history |
| N5 | End a workout containing an unfinished attempt with notes | Remove its draft notes with the attempt; retain workout notes and completed attempt notes |
| N6 | End a workout with notes but no completed attempts | Retain the workout and keep its notes accessible without inventing an exercise history entry |
| N7 | Edit or clear notes offline, close the app, resume, and reconnect | Local notes/queue survive; accepted updates sync once; stale conflicts preserve input for resolution |
| N8 | Retry a note update or send a late note update after finalization | Idempotent retries return the committed result; reject new edits to completed attempts or ended workouts |
| N9 | Clear notes, omit notes from a partial update, or enter multiline/oversized text | Null/blank clears; omission preserves; internal line breaks survive; reject over 4,000 characters without truncation |
| N10 | Move, merge, archive, or delete variants with notes | Move/archive retain notes; merge keeps destination variant notes and all attempt notes; deletion removes owned notes while retaining surrounding workout notes |
| N11 | Submit another owner's IDs for note reads or writes | Reject access consistently for variant, workout, and attempt notes |
| N12 | Import legacy data or later add note columns to existing rows | Notes start null; no reusable instructions or historical observations are inferred |

## Editing and temporary volume

| ID  | Scenario                                                                          | Expected result                                                                                                                                                    |
| --- | --------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| E1  | Change weight before recording a prescribed set                                   | Variant target and unfinished exercise target update immediately                                                                                                   |
| E2  | Record 50 kg, then deliberately change that set's target to 45 kg                 | Actual performance stays 50; target stays exactly 45 even if threshold qualifies; the proposed all-or-nothing policy suppresses increase/reset for this completion |
| E3  | Increase prescribed count from three to four during the exercise                  | Fourth set becomes required immediately; future workouts also prescribe four                                                                                       |
| E4  | Reduce four prescribed sets to two after recording three                          | Third recorded set becomes bonus work; fourth pending set is removed; future volume is two                                                                         |
| E5  | Add a bonus set or skip a set without editing configuration                       | Saved session and variant set count remain unchanged                                                                                                               |
| E6  | Change threshold, increment, or mode during an unfinished exercise                | Completion uses the edited settings                                                                                                                                |
| E7  | Edit a variant after its workout exercise is completed                            | Future configuration changes; completed performance remains immutable                                                                                              |
| E8  | Add bonus work during a saved-session workout, or edit the source plan separately | Bonus work does not change the plan; plan list/order edits affect the next workout and leave active contents intact                                                |

## Persistence, offline use, and concurrency

| ID  | Scenario                                                               | Expected result                                                                                                          |
| --- | ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| S1  | Start a workout from two devices concurrently                          | Only one active workout is created; both clients can resolve the active workout                                          |
| S2  | Record sets, close the app, reopen it                                  | Saved active draft and local pending operations survive                                                                  |
| S3  | Record offline, reconnect, resume on another device                    | Accepted sets/configuration synchronize once and are then visible on the other device                                    |
| S4  | Replay a request with the same operation ID                            | No duplicate set, edit, completion, or increase; a different payload with that ID is rejected                            |
| S5  | Queue a configuration edit, recording, completion, then ending offline | Replay preserves dependency order; ending cannot discard work that should have completed first                           |
| S6  | Two devices edit from different revisions                              | Stale conflicting request is surfaced without silently overwriting accepted work; local input is retained for resolution |
| S7  | Receive a late draft write after exercise completion or workout ending | Reject it; never reopen completed work or mutate a new workout                                                           |
| S8  | Server mutation fails midway                                           | Transaction leaves no partial progression/history change or successful receipt                                           |
| S9  | Switch accounts with unsynced work                                     | Never upload one user's queue as another user; preserve or explicitly discard pending work                               |
| S10 | End a partial workout while offline                                    | Local completion/ending remains visibly pending until accepted; app closure does not lose the queue                      |

## Archive, deletion, and merge

| ID  | Scenario                                                             | Expected result                                                                                                                   |
| --- | -------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| D1  | Archive an exercise used by a session                                | Hide from autocomplete; session shows restore/remove choice                                                                       |
| D2  | Archive only one variant                                             | Other variants remain selectable; archived variant follows the stated default                                                     |
| D3  | Delete an exercise referenced by a saved session                     | Reject until references are removed; archive remains available                                                                    |
| D4  | Delete an unreferenced exercise with history                         | Show alternatives and impact, require confirmations/name entry, then remove its variants/history while keeping other workout data |
| D5  | History changes between deletion preview and confirmation            | Recalculate impact and require the updated count to be acknowledged                                                               |
| D6  | Merge source into destination                                        | Keep destination configuration, move source history/references, preserve recorded performances, display current destination names |
| D7  | Merge variants with separate historical attempts in the same workout | Both attempts remain distinct; no historical performance is collapsed                                                             |
| D8  | Merge variants both referenced by one saved session                  | Reject the whole merge under the proposed collision policy; no partial changes                                                    |
| D9  | Merge/delete a variant involved in an active workout                 | Reject; ordinary configuration edits remain available                                                                             |
| D10 | Supply another user's exercise, variant, target-set, or workout ID   | Reject across reads and writes, including nested requests and merges                                                              |

## Test layers

- Pure Go progression tests cover modes, equality, reset rounding, bonus exclusion, skipped sets,
  off mode, manual edits, and independent weights.
- Database integration tests verify ownership/uniqueness constraints, transactional rollback,
  completion idempotency, immediate configuration reconciliation, merge/delete effects, and
  concurrent workout starts.
- Frontend tests verify target-versus-actual rendering, explicit set states, prescribed-versus-bonus
  controls, and current names in history.
- Persistence tests reload IndexedDB drafts, replay interrupted queues, simulate lost responses, and
  exercise cross-device conflicts and account changes.
- End-to-end scenarios cover saved-session and free-mode workouts, offline recording, resuming,
  ending partially completed work, and archive/delete confirmations.

The initial legacy import suite in `apps/server/porting` covers mapping, name conflicts, aggregates,
provenance, timestamps, retries, and count reconciliation against isolated PostgreSQL fixtures.
See [operations](porting-operations.md) for execution and remaining cutover work. Do not report
fabricated per-set history as a successful conversion of aggregate-only data.
