# Product requirements

These requirements consolidate the agreed behavior. Technical mechanisms and implementation defaults
live in the [database design](schema.md) and [implementation plan](implementation.md).

These requirements describe the complete product. [Application transition planning](application-transition.md)
records stage allocation and confirmed temporary stage 1 behavior, including removing excess draft
sets when reducing prescribed volume rather than retaining them as bonus work.

## 1. Personal library and variants

- Exercises are private to a user. There is no shared exercise catalog.
- Exercise metadata consists of ID, name, and zero or more predefined muscle groups.
- The muscle-group catalog is predefined in backend code. Exercises store an array of its stable
  codes; display labels and list order come from that backend catalog. Muscle groups do not need
  their own database tables.
- Exercise names are unique within a user's library, ignoring capitalization. Variant names are
  unique within their exercise, also ignoring capitalization.
- Creating an exercise creates a `Standard` variant. Its label need not be shown when it is the only
  variant.
- Variants share the parent exercise's metadata but own prescribed sets, per-set rep and weight
  targets, repetition threshold, auto-increment amount, and progression mode. Variants may have
  identical settings.
- Each variant also retains its last reported RPE and the completion time of that rating, shared
  across all sessions using the variant.
- Variants have optional reusable notes, such as setup instructions or technique cues, shared
  across all sessions using that variant.
- Renaming an exercise or variant changes its displayed name everywhere, including historical
  workouts.
- Users add exercises through name autocomplete, library browsing, or creation. Selecting an
  existing exercise leads to selecting its variant.

## 2. Sessions and free mode

A session stores an ordered selection of variants. It contains no independent training-setting
overrides. Reusing a variant in several sessions shares its progression across all of them.

Each variant may appear once in a session. Different variants of the same exercise may appear
together, such as Heavy Bench and Light Bench. The same restriction applies to new workout contents
as a design default; history may contain multiple records after a merge.

A workout is a distinct training occasion. Users can start from a saved session or use free mode,
adding exercises as they go. Only one workout may be active per user.

Editing a saved session changes its planned structure for the next workout. An already active
workout keeps its own exercise list and order. Adding bonus work during a workout does not change
the saved structure. Temporary bonus or reduced volume does not change the variant's prescribed set
count.

## 3. Targets, performance, and immediate edits

Store targets and recorded performance independently. Starting a workout presents the variant
targets available at that time and empty performance inputs. Automatic progression changes future
targets without rewriting what the current workout displays as performed.

**Explicit configuration edits take effect immediately**, including in the unfinished exercise of
the active workout. This applies to prescribed set count, per-set target weights, auto-increment,
repetition threshold, and progression mode. The workout-start targets are a starting point, not a
barrier to intentional edits.

| Action                                              | Effect                                                                                        |
| --------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| Record reps or mark a set performed                 | Save actual work; do not update rep targets until exercise completion                         |
| Edit a prescribed set's weight                      | Immediately update that variant target and the unfinished workout's applicable target         |
| Change prescribed set count                         | Immediately change the variant and the unfinished exercise's expected volume                  |
| Reduce prescribed volume after recording extra sets | Keep the already recorded excess sets as bonus work                                           |
| Add a bonus set                                     | Record extra work without adding a future prescribed set                                      |
| Skip a prescribed set                               | Keep the future prescribed set; apply the reduced-volume rule on explicit exercise completion |
| Finish an exercise                                  | Persist performance and update future targets once                                            |

Already recorded performance is retained when targets change. If a set was performed at 50 kg and
its target is then explicitly changed to 45 kg, history remains 50 kg and the next target is based
on the latest 45 kg edit. Completion must not overwrite that edit with the older recorded weight.
Even if the exercise qualifies for auto-increase, that deliberate target remains exactly 45 kg.
Design policy: suppress automatic weight increase and its associated rep reset for this completion
when a target was manually edited after its performance was recorded. This preserves the rule that
automatic increases apply to all prescribed weights together; recorded reps still become future rep
targets.

Completed exercises cannot be edited or reopened. Editing their variant later changes future
configuration, not their historical performance. Direct configuration edits persist independently of
whether the current exercise is eventually completed.

### Notes

Support optional plain-text notes on these three records:

| Record | Meaning | Example |
| --- | --- | --- |
| Variant | Reusable instructions for this configuration | “Narrow grip; bench at 30 degrees” |
| Workout | Context for one training occasion, from a saved session or free mode | “Slept poorly; kept the workout light” |
| Workout exercise | Observations about this particular exercise attempt | “Last set felt difficult on the left side” |

Each field is independent. Show current variant instructions separately from an attempt's own
notes; starting a workout does not copy variant notes into attempt notes. Completing an attempt
does not copy its observations back to the variant. Editing variant notes changes the reusable
instructions without rewriting any workout or attempt notes, targets, performance, or RPE.

Save workout and draft-attempt notes while the workout is active, including through app closure
and offline synchronization. As an initial design default consistent with completion finality,
attempt notes become read-only when that exercise is completed, and workout notes become read-only
when the workout ends. Variant notes remain editable through ordinary variant editing. Clearing a
note is supported; empty or whitespace-only input represents no note. The proposed initial limit
is 4,000 characters per field, preserving internal line breaks.

Completed attempt notes appear in exercise/variant history. Workout notes appear with their
workout. Ending a workout retains its own notes and completed attempt notes; removing an unfinished
attempt also removes its draft notes. An ended workout with notes remains accessible even when it
has no completed exercise attempts. Notes do not affect progression or last RPE.

## 4. Progression rules

The existing average-based calculation is in
[`saving.ts`](../../apps/web/src/routes/tracker/saving.ts), in `CheckRepThresholdMet` and
`buildProgressUpdate`. Preserve the reset formula and extend the calculation to variants,
independent set weights, and selectable modes.

Each variant has one mode:

| Mode     | Eligibility for automatic weight increase                                                                                               |
| -------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| Average  | Every prescribed set is performed, and their average reps meet or exceed the threshold                                                  |
| All sets | Every prescribed set is performed and each reaches the threshold; use the same inclusive comparison as average mode as a design default |
| Off      | Never automatically increase weights or reset performed sets' rep targets                                                               |

Bonus sets are excluded from both checks. Skipping even one prescribed set blocks an automatic
increase for the whole variant. Do not round the average before comparing it to the threshold.

Use the existing reset calculation: `reset_reps = Math.round(rep_threshold / 1.7)`. The Go
implementation must produce the same result for valid positive thresholds.

| Outcome on explicit exercise completion                | Next targets                                                                                       |
| ------------------------------------------------------ | -------------------------------------------------------------------------------------------------- |
| Eligible increase, with no late manual weight override | Add the variant's increment to every prescribed set's weight and reset every prescribed rep target |
| No eligible increase, performed prescribed set         | Recorded reps become its next rep target; retain its current target weight, including manual edits |
| Skipped prescribed set                                 | Reset its rep target and keep its target weight unchanged                                          |
| Bonus set                                              | History only; no future target row and no effect on progression                                    |

Ordinarily target weight and performed weight agree because weight edits update the target directly.
A later explicit target edit takes precedence over an earlier performance value and over automatic
increase for that completion. Determine next targets once per completion; never repeatedly increment
an already progressed value on retries.

Examples:

- Reps `12, 11, 10` average 11. With threshold 11, increase weight and reset reps to `6, 6, 6`. With
  threshold 12, next reps remain `12, 11, 10`.
- Weights `40, 50, 50 kg` become `42.5, 52.5, 52.5 kg` after a legitimate 2.5 kg increase.
- With three prescribed sets and threshold 12, performing only two blocks progression. The third set
  keeps its weight and receives a rep target of 7.
- Off mode still updates performed sets' rep targets. It does not prevent manual weight edits.
  Skipped prescribed sets still receive reset rep targets.

### RPE recording

RPE is an optional effort rating for the whole exercise attempt, rather than each set. Store it
independently of reps, weights, and future targets. The initial scale is integer 0–10, matching the
current `RPESlider.svelte`; `null` means not reported, and an explicit zero is a valid rating.
The slider's initial display value is not a reported rating unless the user selects or confirms it.

Persist a selected rating with the unfinished attempt so it survives resume and offline sync.
Explicit exercise completion freezes that attempt's RPE in history and, when a rating is present,
updates the variant's last RPE and rating completion time in the same transaction as progression.
Completing without a rating leaves the previous last RPE and its timestamp unchanged. Abandoning
an unfinished attempt never updates the variant's last RPE. Completed ratings cannot be edited or
reopened, consistent with other recorded performance.

The last rating is available for later RPE-based adjustment logic. Recording it does not change
any progression rule or automatically adjust the next exercise in this version.

## 5. Workout lifecycle and durability

1. Start or resume the user's single active workout.
2. Persist set performance and explicit configuration edits while the workout is active.
3. Explicitly complete an exercise to retain its performance and update its variant's next targets.
4. Continue showing actual recorded performance when revisiting the completed exercise.
5. End the workout, retaining completed exercises and discarding unfinished exercises from history.

Completing an exercise with deliberately reduced volume is a valid completion. Leaving a partially
recorded exercise unfinished and ending the workout does not create history or apply
completion-based progression for that exercise. Completed exercises still count even when the rest
of the workout is abandoned.

Closing the app is not ending the workout. Active progress must survive app closure, be resumable
across devices after sync, and support offline recording followed by later synchronization. The UI
must distinguish changes saved on the device from changes confirmed by the server; another device
cannot receive unsynced local work.

## 6. History

History stores individual recorded sets and their workout context. It supports reviewing a workout,
tracking one variant, and collecting progression across variants of a base exercise. Bonus work is
included as recorded performance but remains distinguishable from prescribed work.

Each completed attempt includes its optional exercise-level RPE. An unrated attempt remains
unrated even if its variant has a last RPE from an earlier completion. Legacy data has no recorded
RPE; never infer a rating from reps, weights, a slider default, or a later attempt.

Completed workout performance is independent of future target updates and configuration edits.
Current exercise/variant names are resolved for display, including after merges. Draft exercise
records are excluded from history until explicitly completed.

## 7. Archiving and deletion

Exercises and individual variants can be archived. Archived exercises disappear from autocomplete.
An archived exercise referenced by a saved session has a greyed-out screen offering unarchiving or
removal from that session. Apply the same behavior to individually archived variants as a design
default.

An exercise referenced by a saved session cannot be permanently deleted. Removing a session
placement does not delete the library exercise. Merging or deleting variants involved in an active
workout is prohibited; ordinary configuration edits remain available.

Historical data does not prohibit permanent deletion. Encourage archive or merge first. Deletion
requires multiple explicit confirmations, displays the number of historical entries affected, and
requires typing the exercise name. Explain the affected variants when deleting a parent exercise.

Deleting a variant removes its history. Deleting a base exercise removes all its variants and their
history. Surrounding workouts and their other exercises remain. The technical design defines the
count as completed exercise records, with set counts shown separately for clarity.

## 8. Transfer and merge

Users can move exercise data under another exercise as a variant, and merge into an existing
variant. All operations remain within the user's library.

On merge, the destination retains its configuration and targets. Source history and saved-session
references move to it. Distinct historical performances remain distinct, even when they occurred
within the same workout. Names shown in historical workouts become the destination's current names.

Keep each attempt's RPE on merge. Recompute the destination's last RPE and timestamp from its
combined rated completed history; these are recorded performance summaries, not configuration.
Attempt notes move with their history and remain distinct. The destination keeps its variant
notes as part of its configuration; show source notes in the merge preview before removing the
source variant. Moving a variant to another exercise retains its notes. Workout notes stay on the
surrounding workout through variant moves, merges, or deletion.

For saved sessions containing both source and destination, the design default is to block the merge
until one placement is removed. No automatic choice of placement or ordering is made. This resolves
the previously deferred collision case without expanding merge behavior.
