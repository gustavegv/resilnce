# Application transition planning

**Status:** Agreed application transition plan. Scope, implementation sequence, validation
approach, and later-stage allocation were confirmed in the planning discussion. Next synthesize
the detailed stage 1 implementation spec, then create implementation tickets. This document is
not itself a published spec or ticket breakdown.

The new database schema is in place. Application operations still use the legacy model.
The [product requirements](decisions.md) describe the destination; the delivery stages below
determine when those capabilities become available. Deferring a capability to a later stage
does not remove it from the product requirements.

## Confirmed delivery stages

1. **Adapt existing workflows.** Make the existing app features work with the exercise-centric
   model. Keep the current styles and minimize visible frontend changes. Required changes include
   displaying variants and adapting session creation to reuse exercises/variants. Persist active
   workout drafts to the server while online, including unfinished sets; support online resume and
   safe completion retries.
2. **Extend those workflows.** Add features to the adapted app, including persisted RPE, notes,
   bonus-set creation, per-set weight editing, progression-mode selection, and durable offline
   recording/synchronization with conflict handling. Include separate library management,
   archive/merge/delete, muscle-group selection, and detailed history interfaces.
3. **Add free mode.** Implement standalone workouts without a saved session after the earlier
   stages.

Stage 1 preserves familiar user workflows while adopting the new domain model. A separate
library-management screen is deferred beyond stage 1; creating and selecting variants remains
available within session creation/editing.

## Development and merge

- Work takes place on a feature branch.
- The user permits development and testing against the live database.
- The branch is merged during a service window when ready.
- The user controls branch, merge, and release timing; planning does not prescribe a release
  or service window per stage.
- Add an env-controlled development mode that forces application use of a dedicated test account.
  Enabling it signs the app into that account without Google login and creates the account if
  missing. The account starts with an empty library, retains test data across restarts, and is
  visibly identified in the app. Disabling devmode restores normal login. Backend-supplied
  identity is consistent across account, workout, and AI operations.
- Existing automated migration/import tests retain their explicitly isolated database setup.

## Frontend direction

Preserve the existing styles. Stage 1 should make only the interface changes required by the
new model; later stages introduce their corresponding controls and flows. Detailed navigation,
styling adjustments follow the existing app rather than introducing a visual redesign.

### Stage 1 session creation and editing

Keep the current layout and replace the exercise-name input with search/autocomplete. Selecting
an existing exercise offers its existing variants or creation of a new variant. Creating a new
exercise also creates its Standard variant. Adding the same variant to multiple sessions reuses
its shared configuration and progression.

Keep these capabilities within session creation/editing for stage 1. A separate library-management
screen is a later-stage capability.

AI Quick fill and Program creation remain available in stage 1. Their drafts pass through a
review screen that resolves exercise/variant reuse before saving. When suggested settings differ
from an existing variant, the user chooses between reusing that variant's current configuration
and creating a new variant from the suggestion. Generated plans never silently overwrite shared
targets.

### Shared configuration editing

Keep the existing exercise-edit workflow, showing exercise and variant names and a small warning
that configuration changes affect every session using the variant. Target-weight edits preserve
already recorded actual weights. Users needing independent settings create a separate variant
through session creation/editing.

### Set recording and weights

Use tap-to-complete for sets and persist the performed state. Loading or displaying target reps
never records them as actual reps. Pending, performed (including explicit zero reps), and skipped
sets remain distinct. Completing an exercise must resolve pending sets explicitly rather than
submitting their target values as performance; skipped prescribed sets prevent automatic weight
progression.

Stage 1 keeps common-weight editing but displays and preserves each set's individual target
weight, including unequal imported targets. Only an explicit "Apply weight to all sets" action
makes existing weights equal. Changing set count preserves retained sets' weights. Per-set weight
editing is deferred to stage 2.

**Confirmed stage 1 exception to the full-product volume-editing rule:** reducing prescribed
set count removes excess sets from the unfinished attempt, including their already recorded
reps/weights. Do not convert or retain them as bonus work in stage 1. Stage 2 introduces preserving
such work as bonus sets, along with manual bonus-set creation. Retained prescribed sets keep
their individual weights. This applies to editing an unfinished attempt; completed performance
remains immutable.

### Progression

Stage 1 retains average-reps progression and the existing reset formula, calculated by the server
from actual performed sets. The UI exposes no new progression-mode selector. Stage 2 adds the
All sets and Off choices.

### Statistics and history

Stage 1 retains the current Stats layout and charts, adds variant labels, and clearly identifies
imported history. New chart entries derive from recorded performance. Imported aggregate values
remain unchanged; legacy history without reliable variant attribution stays associated with its
parent exercise. Detailed workout/per-set history screens are deferred beyond stage 1.

For new entries, total volume is the sum of each performed set's recorded weight multiplied by
its recorded reps. Keep average weight and the existing average-based estimated-1RM method for
continuity, using actual recorded performance and without inventing an estimate when no reps
were performed. Imported entries retain their original aggregate values; their volume is an
explicitly identified estimate derived from those aggregates. Do not round new average reps to
an integer merely to imitate legacy storage.

### Workout ending

In stage 1, completing the final listed exercise automatically ends the workout, preserving the
current interaction. Introduce explicit ending in stage 2 alongside bonus work. Quitting early
retains completed attempts and discards unfinished attempts. Closing the app leaves the workout
active and resumable.

### Online saving and failures

In stage 1, show Saving, Saved, or Not saved with retry. A failed save retains the just-entered
values on screen and pauses further recording. Require successful saves before exercise
completion or automatic workout ending, and prevent in-app navigation from silently discarding
unsaved input. Only server-confirmed progress is durable across reload at this stage. Stage 2
adds durable local storage and offline replay.

## Allocation of remaining capabilities

The following allocation completes the agreed staged scope:

- **Stage 1:** preserve existing session rename, add/remove placement, and session deletion
  workflows. Removing a placement does not delete its reusable variant or history. Retain existing
  saved-session ordering behavior. Preserve/validate muscle-group codes without adding a new
  muscle-group selection interface.
- **Stage 2:** separate library management, muscle-group selection, archive/restore,
  move/merge/permanent deletion, and detailed workout/per-set history. Add workout-only variants
  to a session-based workout without modifying the saved plan. These extend the new model after
  the existing workflows work end to end.
- **Stage 3:** starting and using standalone free-mode workouts, reusing the same recording,
  completion, history, and synchronization implementation.

RPE-based automatic adjustment remains outside these stages as previously deferred in the
product requirements. Later-stage UI details can be designed when that stage is planned; they
do not block the stage 1 implementation spec.

## Implementation sequence for stage 1

Each numbered item should produce verifiable behavior through the relevant backend and frontend
layers; these are planning slices, not yet published tickets. Slice size and blocking edges will
be refined when making implementation tickets.

1. **Development account and identity.** Add env-controlled forced test identity, non-destructive
   account provisioning, visible account status, and normal authentication when devmode is off.
   Use the same identity for account reads, new application operations, and AI quota identity.
   Internal porting access retains its separate verified-user authorization; devmode does not
   grant importer administration. Establish authenticated ownership and the new model's request
   contracts without depending on a legacy session pointer.
2. **Create and maintain reusable session plans.** Connect manual creation and session editing
   to exercise/variant search, creation, and ordered references. Cover shared-configuration
   editing and its warning. Keep existing list, rename, remove-placement, and delete-session
   interactions. The first usable path creates an exercise/variant and saves a session referencing
   it; further work should stay small enough to verify that path independently.
3. **Start, record, resume, and finish a session-based workout.** Connect the tracker to actual
   workout and set IDs. Add explicit performed/skipped state, online draft saving, save status
   and retries, and authoritative server completion/progression. Starting from another device
   resolves the same single active workout. Complete the final attempt and end its workout in a
   safe transaction so a retry cannot progress twice or strand the active workout. Preserve the
   agreed stage 1 volume-reduction exception and independent weights.
4. **Adapt AI and Program creation.** Route both through variant/configuration review, using the
   same create/reuse operations as manual creation. Demonstrate that accepting a generated plan
   cannot silently overwrite a variant already used elsewhere.
5. **Restore statistics against the new model.** Connect the existing charts to completed
   recorded performance and compatible imported summaries, with variant labels and aggregate
   provenance. Verify exact new volume, unchanged imported values, and visibility of exercise-only
   legacy history.
6. **Verify normal-account operation and cutover readiness.** Exercise the same flows through
   normal sign-in, validate representative imported fixtures, and prevent stale legacy clients
   from writing the old model after the application switch. The user chooses merge/release timing.
   Production importing remains gated by the separate rehearsal, source-exception, backup, and
   write-freeze requirements in the porting plan.

## Backend modules and validation

Keep Go/PostgreSQL and Svelte. Put ownership, revision checks, transactions, completion, and
configuration reconciliation behind a small application-module interface. HTTP handlers decode
and authenticate requests; they do not orchestrate separate target/history writes. Keep
progression as a deterministic calculation used by completion. Keep SQL persistence inside the
application implementation rather than introducing an interface for every query.

Use an injected identity seam with normal-authentication and development-account adapters. The
frontend receives authoritative workout state and future targets separately, with stable IDs and
consistent decimal-safe weight contracts. Revision checks reject stale device edits; stable
operation IDs make retries safe. New application code writes the new model only; do not maintain
competing legacy/new target stores for the same workflow.

Verify behavior primarily through application HTTP operations backed by an explicitly isolated
PostgreSQL database. Test pure progression rules directly and use focused frontend checks for
tap-to-complete, saving/retry state, generated-plan review, and variant identity. Use the live-DB
development account for manual workflow checks; do not redirect fixture-creating automated tests
to the application's live database. Existing migration/import tests remain the prior art for
database setup.

Stage 1 acceptance must demonstrate:

- New exercises create a Standard variant; two sessions can reuse one variant without copying it.
- Manual, AI, and Program creation resolve variants explicitly and preserve reused configuration.
- The test account persists across restart; normal mode uses real authenticated ownership.
- Untouched targets never become actual reps; performed zero, pending, and skipped are distinct.
- Saved unfinished sets survive reload and resume on another device; failed saves remain visibly
  unsaved, pause recording, and can be retried without duplication.
- Unequal target weights survive loading, set-count changes, and progression unless explicitly
  replaced for all sets. Reducing prescribed count drops excess draft work in stage 1.
- Completion records actual values and advances future targets exactly once, even after a lost
  response. A skipped prescribed set blocks an increase; a late explicit target-weight override
  is respected according to the agreed progression policy.
- Quitting retains completed attempts and discards drafts; final-attempt completion ends the
  workout safely. Completed history is immutable.
- Stats show correct new performance and clearly identified imported aggregates, including history
  that has only a parent-exercise attribution.
- Cross-owner IDs, stale revisions, concurrent workout starts, and partial transaction failures
  cannot produce mixed ownership, duplicate progression, or partially committed completion.
- Cached legacy clients cannot mutate legacy targets after cutover to the new application.

The [implementation investigation](implementation.md) supplies further code findings and technical
proposals. [Acceptance criteria](acceptance.md) covers the complete product; use the stage-specific
rules here when defining stage 1 tests.
