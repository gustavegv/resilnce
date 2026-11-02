# Stage 1 Spec errors

This collection records requirements that added unnecessary complexity, what was
removed, and the exact wording in the documents and issues that described it.
Use these examples to refine the spec-making process as patterns emerge.
Quoted passages are historical evidence of withdrawn or revised wording.

## Unnecessary “Discard local input” button

Recorded: 2026-11-02.

### What was removed

Removed the standalone **Discard local input** button from the workout screen and
its handler, which cleared local input and reloaded the workout. The user judged
this action unnecessary. Also removed navigation guidance directing users to it:

> Input is not saved. Stay and retry, or explicitly discard local input before leaving.

The conflict-resolution button previously labeled **Accept server state and
discard local input** was renamed **Accept server state**. Accepting reviewed
server state remains available, as do saving, retrying and canceling an edit.

Definitively rejected completion or quit commands now release their pending
intent automatically while retaining the error. This keeps existing editing
flows available without requiring a separate discard action. Uncertain commands
still retain their input and require retry or reviewed conflict resolution.

### Spec-process observation

The spec turned the need to protect unresolved input into a requirement for an
additional discard action. Supporting documentation then described reloading
after discard, shared discard guards, and manual discard after a rejected
completion. Removing the button required removing these related expectations.

For future specs, establish the user need for a new recovery control and check
whether existing save, retry, cancel and conflict-resolution flows cover it
before prescribing another action.

### Documentation and issue references

#### `docs/exercise-centric/training-application.md`

Source: [Training application](exercise-centric/training-application.md).
The following are the exact former passages that were removed or rewritten.

**Explicit online set recording — navigation recovery and reload after discard:**

> Stay/retry or explicit local discard resolves the guard; discard
> after an uncertain response refetches authoritative state before recording can
> resume.

This was replaced with guidance to resolve input through the existing save,
retry, cancel or reviewed-conflict flow.

**Explicit online set recording — conflict acceptance wording:**

> Accept server state
> explicitly discards input.

This now describes using the reviewed state and clearing conflicting input;
the conflict-resolution action itself remains.

**Shared configuration editing during a workout — shared discard guards:**

> Configuration and recording share one immutable pending-command slot and the
> same Saving/Saved/Not saved, original-account/workout, navigation/unload,
> retry/discard and reviewed-conflict guards.

Removed the discard reference from the shared guards.

**Completion, average progression and automatic ending — blocking discard while uncertain:**

> An uncertain completion offers
> exact Retry and cannot be discarded or overtaken through navigation/account
> changes; resolving the same intent returns authoritative completion/ending.

Removed the discard-specific wording while retaining retry and navigation/account
guards for uncertain completion.

**Completion, average progression and automatic ending — manual discard after rejection:**

> Definitive validation rejection permits discarding that rejected intent and
> reviewing configuration.

Replaced this manual recovery step with automatic release of the rejected intent
while retaining the error so configuration can be reviewed.

**Quit partial workouts — discard in the uncertain-quit guard list:**

> An uncertain quit retains its exact identity/revision/operation ID,
> blocks edits/discard/navigation/unload, and offers identical Retry under the
> original account/workout.

Removed discard from the guard list; the remaining uncertain-quit protections
still apply.

#### GitHub issue #25 — Stage 1 specification

Source: [Stage 1: adapt existing workflows to exercise-centric training](https://github.com/gustavegv/resilnce/issues/25),
decision **10. Online saving, conflicts, and safe retries**.

Exact wording replaced within the navigation requirement:

> Offer staying/retry or an explicit discard of local draft input; never silently leave. An uncertain completion must be resolved by retry/authoritative state before a discard or new workout can be treated as safe.

The replacement uses existing save, retry, cancel and reviewed conflict resolution
to resolve input. Uncertain completion must still be resolved before leaving or
starting a new workout.

#### GitHub issue #33 — Completion implementation note

Source: [Implementation comment on Stage 1: complete attempts and progress targets exactly once](https://github.com/gustavegv/resilnce/issues/33#issuecomment-6556764380).

Exact wording replaced in the implementation note:

> retains uncertain completion for identical retry without discard/navigation/account overtaking.

Removed `discard/`; the note still records protection against navigation and
account changes overtaking an uncertain completion.

## Stage 1 frontend audit

Recorded: 2026-11-02. The [original frontend report](exercise-centric/stage-1-problems.md)
is preserved separately. These entries cover the problems fixed throughout the chat,
including subsequent creation, navigation and generated-review tweaks.

Audited the exercise-centric docs, searched other repository Markdown, and checked all 33 GitHub
issues (open and closed), including their comments. The relevant Stage 1 specification
is [#25](https://github.com/gustavegv/resilnce/issues/25), with implementation tickets
#26–#37. Earlier tickets were also checked for conflicting instructions.

**How to read this record:** quotations are exact former wording, retained here as
historical evidence after removal or replacement in the active source. They must
not be implemented. “No wording removed” means the audit found no instruction for
the unwanted behavior; the fix corrected implementation drift or filled a design
gap. Existing preservation, validation, ownership, save-status and retry requirements
remain valid. A matching passage documents a requirement or description, not proof
that it caused the implementation.

The separately recorded removal of the **Discard local input** control is covered
in [the existing entry above](#unnecessary-discard-local-input-button); its evidence
and the related edits are preserved there rather than duplicated here.

### 1. Workout recording replaced the established tracking UI

Removed the replacement workout layout and restored the existing tracking components,
set counters, tap/stepper interactions, exercise navigation, information menu and
progress presentation, connected to workout/attempt/set identities. Related repairs
kept dialogs usable on small screens, dropdowns above surrounding content and the
development-account indicator clear of workout controls.

**Finding:** implementation drift. The spec already required preserving the frontend.

#### `docs/exercise-centric/application-transition.md`

[Frontend direction](exercise-centric/application-transition.md#frontend-direction) already required
preserving styles and minimizing visible changes. No wording removed. Added an
explicit instruction to reuse tracking and other existing components, and a link
to this historical audit.

#### GitHub issue #25 — Stage 1 specification

[Solution](https://github.com/gustavegv/resilnce/issues/25) already required keeping
the existing Svelte interface and styles; visual redesign was out of scope. No
wording removed. Made component reuse explicit. A schema/API adaptation must not
be treated as authorization to replace the surrounding screen.

### 2. The exercise finish screen was replaced

Removed the replacement finish presentation and restored the existing completion
cards and visual progression feedback. Completed actuals remain separate from
current future targets.

**Finding:** implementation drift, not a requirement to redesign completion.

#### `docs/exercise-centric/application-transition.md`

[Frontend direction](exercise-centric/application-transition.md#frontend-direction) now names the
existing finish component among the components to reuse. No wording removed.

#### GitHub issue #25 — Stage 1 specification

[#25](https://github.com/gustavegv/resilnce/issues/25) story 49 requires frozen actuals
and separate next targets. No wording removed; explicitly preserved the finish component.

#### GitHub issue #33 — Completion

[#33](https://github.com/gustavegv/resilnce/issues/33) requires correct completed
performance and progression, not a new layout. No wording removed.

### 3. Continue workout lost its styled card and home state

Removed the unstyled replacement Continue presentation. Restored the shared resume
card and active-workout home presentation while using the actual workout UUID.

**Finding:** implementation drift. The home Continue action itself is required.

#### `docs/exercise-centric/training-application.md`

[Session-based workout start and resume](exercise-centric/training-application.md#session-based-workout-start-and-resume)
now explicitly describes the existing styled resume card. No instruction prescribed
an unstyled link. The start-flow paragraph replacement is quoted under problem 7.

#### GitHub issue #25 — Stage 1 specification

[#25](https://github.com/gustavegv/resilnce/issues/25) preserves styles; it now also
names the resume-card component. No wording removed.

#### GitHub issue #30 — Start and resume

[#30](https://github.com/gustavegv/resilnce/issues/30) requires home Continue to
resolve a workout ID. This requirement is retained. No wording removed.

### 4. Tracker edit/delete menus and the session edit screen disappeared

Removed the replacement session-list/edit presentation. Restored the three-dot,
slide-open edit/delete menu and existing edit dialog, retaining rename, ordering,
add/remove placement and variant reuse with the new operations.

**Finding:** implementation drift. Required existing interactions were lost.

#### `docs/exercise-centric/application-transition.md`

[Frontend direction](exercise-centric/application-transition.md#frontend-direction) and
[stage allocation](exercise-centric/application-transition.md#allocation-of-remaining-capabilities)
already preserved session maintenance. No wording removed. Explicitly retained the
slide-open menu and existing edit dialogs.

#### GitHub issue #25 — Stage 1 specification

[#25](https://github.com/gustavegv/resilnce/issues/25) stories 14–18 preserve session
maintenance. No wording removed; explicitly named the slide-open menu and edit components.

#### GitHub issue #29 — Session maintenance

[#29](https://github.com/gustavegv/resilnce/issues/29) requires the existing session
maintenance capabilities. No wording removed; new operations should use those interactions.

### 5. Shared-configuration editing lost the established form design

Removed the stripped replacement edit presentation. Restored existing styled inputs
and undo interactions while keeping distinct exercise/variant names, the compact
shared-configuration warning, mixed weights and explicit Apply weight to all sets.

**Finding:** implementation drift. Shared settings needed new semantics, not a new form style.

#### `docs/exercise-centric/application-transition.md`

[Shared configuration editing](exercise-centric/application-transition.md#shared-configuration-editing)
already required the existing edit workflow and a small warning. No wording removed;
the frontend guidance now also explicitly requires reusing edit components.

#### GitHub issue #25 — Stage 1 specification

[#25](https://github.com/gustavegv/resilnce/issues/25) decision 8 already required
the existing edit workflow and compact warning. No wording removed.

#### GitHub issue #32 — Shared configuration

[#32](https://github.com/gustavegv/resilnce/issues/32) requires that workflow and
intentional weight equalization. No wording removed.

### 6. Workout Back navigated to the nonexistent `/workout` page

Removed the broken parent-path navigation. Back from a workout, including the start
route, returns to `/tracker`.

**Finding:** routing defect. No audited spec or ticket instructed Back to visit `/workout`.

#### `docs/exercise-centric/training-application.md`

[Start and resume](exercise-centric/training-application.md#session-based-workout-start-and-resume)
now states the Back destination. No incorrect route instruction was removed.

#### `docs/exercise-centric/application-transition.md`

[Workout entry and navigation](exercise-centric/application-transition.md#workout-entry-and-navigation)
now records the destination. No wording removed.

#### GitHub issue #25 — Stage 1 specification

[#25](https://github.com/gustavegv/resilnce/issues/25) now records the destination.
No wording removed. Route names being an implementation detail did not define a
working user navigation path; that path is now explicit.

### 7. Starting a workout exposed a preview and several intermediate action screens

Removed tracker-side plan loading/preview, the extra Start action, and the successful
Saved/ready/Continue step. A tracker selection immediately opens the workout start
route, loads/starts there with one loading state, then enters the resolved workout.
Uncertain-request retries still preserve exact intent; home Continue remains available.

**Finding:** stale implementation documentation endorsed an extra continuation step.
The source-plan revision and retry requirements themselves did not require that UI.

#### `docs/exercise-centric/training-application.md`

[Start and resume](exercise-centric/training-application.md#session-based-workout-start-and-resume).
Exact former paragraph replaced:

> The shared browser client adds `startWorkout`, `activeWorkout`, and `readWorkout`.
> `ResumeWorkout.svelte` supplies the home Continue link using the returned workout
> UUID. `WorkoutStart.svelte` captures exact start intent and opaque revision,
> guards unresolved navigation, freezes plan selection and retries identically.
> After a definitive rejection it can explicitly re-read/review the plan before a
> new intent. A different already-active source is identified before Continue.

Replaced with immediate route entry, loading/starting at the destination, automatic
successful entry and the tracker Back destination. Source identification and safe
retry/review after a definitive rejection remain.

#### GitHub issue #27 — Manual-session implementation note

[Implementation comment](https://github.com/gustavegv/resilnce/issues/27#issuecomment-6553272576).
Exact former sentence replaced:

> The UI retains failed drafts and supports retry, save state, stable draft ordering and current-plan previews.

Removed the current-plan preview reference. Retry, save-state and ordering guidance
remain. The note labels this UI correction separately from its original test results.

#### `docs/exercise-centric/application-transition.md`

[Workout entry and navigation](exercise-centric/application-transition.md#workout-entry-and-navigation)
now states immediate entry. No wording removed.

#### GitHub issue #25 — Stage 1 specification

[#25](https://github.com/gustavegv/resilnce/issues/25) now states immediate entry.
No extra start-step requirement was present or removed here.

#### GitHub issue #30 — Start and resume

[#30](https://github.com/gustavegv/resilnce/issues/30) requires correct active-workout
identity and source identification, which do not require another Continue button.
No wording removed.

### 8. Set options duplicated the main screen's recording controls

Removed the dropdown's duplicate reps field and Record/Save correction controls.
The main counters/steppers handle recording and corrections; the menu retains Skip,
Reset and recorded details.

**Finding:** redundant UI, with an implementation description that named extra actions
without distinguishing them from the existing controls. No source required duplicating
them inside a dropdown.

#### `docs/exercise-centric/training-application.md`

[Explicit online set recording](exercise-centric/training-application.md#explicit-online-set-recording).
Exact former wording replaced:

> Explicit
> Record set, Save reps correction, Skip set and Reset to pending use stable IDs.

The replacement assigns recording/correction to the main set controls and describes
only the additional menu actions. Stable identities and all recording commands remain.

#### `docs/exercise-centric/application-transition.md`

[Set recording](exercise-centric/application-transition.md#set-recording-and-weights)
now clarifies the control placement. No wording removed.

#### GitHub issue #25 — Stage 1 specification

[#25](https://github.com/gustavegv/resilnce/issues/25) now clarifies the control
placement. No duplicate-control requirement was present or removed here.

#### GitHub issue #31 — Set recording

[#31](https://github.com/gustavegv/resilnce/issues/31) requires tap-to-complete,
correction, skip and reset behavior, not duplicate inputs. No wording removed.

### 9. Exercise matches were loose buttons instead of autocomplete

Removed the ad hoc matching buttons. Added a real results dropdown and muted inline
completion for the best prefix match, with pointer and keyboard selection. The ghost
suffix does not change the draft until accepted.

**Finding:** an underspecified interaction. Autocomplete was already required; the
sources did not prescribe the unwanted button layout.

#### `docs/exercise-centric/application-transition.md`

[Session creation and editing](exercise-centric/application-transition.md#stage-1-session-creation-and-editing)
already required replacing the name input with search/autocomplete. No wording removed;
clarified dropdown and muted-suffix behavior.

#### `docs/exercise-centric/training-application.md`

[Personal library selection](exercise-centric/training-application.md#personal-library-selection-and-mixed-drafts)
now records the dropdown, prefix hint, keyboard selection and explicit custom-name path.
No button-layout instruction was removed.

#### GitHub issue #25 — Stage 1 specification

[#25](https://github.com/gustavegv/resilnce/issues/25) now clarifies the interaction.
Its removed personal-library-only restriction is quoted under problem 10.

#### GitHub issue #28 — Exercise selection

[#28](https://github.com/gustavegv/resilnce/issues/28) now clarifies the interaction.
Its removed personal-library-only criterion is quoted under problem 10.

### 10. Common exercise-name suggestions were missing and explicitly excluded

Removed the deferral and personal-library-only autocomplete restriction. The backend
metadata now supplies ten common gym exercise names. A suggestion configures a new
private exercise with Standard; records are created only on session save. A matching
personal exercise takes precedence and keeps its existing variants/settings.

**Finding:** a planned catalog was deferred, and the Stage 1 spec narrowed autocomplete
to private records. This chat superseded that scope restriction. Name-only suggestions
do not introduce shared exercises, seeded user records or shared progression.

#### `docs/exercise-centric/questions.md`

[Catalog decisions](exercise-centric/questions.md#exercise-name-suggestions-implemented).
Exact former deferral replaced:

> The `predefined`
> package can also hold the planned exercise-name catalog later.

Replaced with the implemented name-suggestion catalog and its private-creation semantics.

#### `docs/exercise-centric/implementation.md`

[Application operations](exercise-centric/implementation.md#proposed-operations).
Exact former Metadata operation description replaced:

> Return the code-defined muscle-group catalog with display labels and ordering for frontend selection

The metadata operation now includes name-only suggestions alongside muscle groups.

#### `docs/exercise-centric/training-application.md`

[Quick fill and Program review](exercise-centric/training-application.md#quick-fill-and-program-review).
Exact former incomplete catalog-response description replaced:

> `GET /api/training/catalog` returns `{muscleGroups:[{code,label,sort_order}]}`

The response description now includes `exerciseNames`. The personal-selection section
also documents private-match precedence and no implicit database seeding.

#### GitHub issue #25 — Stage 1 specification

[Decision 5](https://github.com/gustavegv/resilnce/issues/25).
Exact former restriction replaced:

> Autocomplete is confined to the authenticated user's library and excludes archived exercises/variants from ordinary selection.

The replacement combines private selectable records and name-only suggestions while
retaining archived-resource and ownership protections.

#### GitHub issue #28 — Search and reuse acceptance criteria

[First autocomplete criterion](https://github.com/gustavegv/resilnce/issues/28).
Exact former criterion replaced:

> - [ ] Exercise autocomplete searches only the current personal library and offers existing variants or explicit named-variant creation; a new exercise still creates Standard.

The replacement includes the ten suggestions, dropdown/inline hint and private-match
precedence. A second criterion records variant cards and mutually exclusive creation.

#### `docs/exercise-centric/decisions.md`

[Personal library requirements](exercise-centric/decisions.md#1-personal-library-and-variants)
now distinguish name suggestions from domain records. No wording removed. The existing
rule that exercises are private and there is no shared exercise catalog remains valid:
metadata suggestions are not shared exercise entities.

#### `docs/exercise-centric/acceptance.md`

[L8–L9](exercise-centric/acceptance.md#library-and-reuse) cover creation timing and
private-match precedence. No wording removed; these are new acceptance cases.

### 11. Reuse and Create named variant were oversized action buttons

Removed the oversized reuse/create actions. Existing variants are selectable cards
with compact configuration summaries; the creation choice is a ghost outline card
with a plus and Create new variant. Reuse still accepts the reviewed settings unchanged.

**Finding:** a design gap. Choosing reuse or independent configuration is required;
the specs did not require large buttons for those choices.

#### `docs/exercise-centric/application-transition.md`

[Session creation](exercise-centric/application-transition.md#stage-1-session-creation-and-editing)
now describes cards and the ghost creation action. No wording removed.

#### `docs/exercise-centric/training-application.md`

[The shared resolver contract](exercise-centric/training-application.md#personal-library-selection-and-mixed-drafts)
now describes cards and the ghost creation action. No wording removed. The existing
requirement for explicit reviewed reuse remains.

#### GitHub issue #25 — Stage 1 specification

[#25](https://github.com/gustavegv/resilnce/issues/25) now describes the cards.
No oversized-button instruction was found or removed.

#### GitHub issue #28 — Variant selection

[#28](https://github.com/gustavegv/resilnce/issues/28) now describes the cards.
Explicit reuse/create choices and protection of shared targets remain.

#### GitHub issue #35 — Generated review

[#35](https://github.com/gustavegv/resilnce/issues/35) retains explicit reuse/create
choices. No oversized-button instruction was found or removed.

### 12. Generated-session review exposed the core JSON for editing

Removed Edit suggested draft, the raw JSON textarea/correction flow and payload-focused
recovery messages. Invalid output offers regeneration or manual creation. Valid
suggestions remain editable through the ordinary exercise/variant forms, and no
invalid output creates domain records.

**Finding:** explicit overengineering in the implementation documentation, reinforced
by ambiguous editable-draft language upstream. Requiring validation and recovery
did not justify exposing a serialization format to users.

#### `docs/exercise-centric/training-application.md`

[Quick fill and Program review](exercise-centric/training-application.md#quick-fill-and-program-review).
Exact former requirement replaced:

> Malformed/invalid output remains visible in an editable JSON
> field with a correction action; no domain mutation is issued. Applying a
> corrected draft explicitly restarts review.

Replaced with ordinary form editing for valid suggestions and an error/recovery path
for invalid output, without showing the payload or a raw editor.

#### GitHub issue #25 — Stage 1 specification

[Decision 6](https://github.com/gustavegv/resilnce/issues/25).
Exact former ambiguous requirement replaced:

> Invalid/failed AI output must leave an editable/error state, not a stored session.

Now distinguishes normal field editing from malformed-output recovery and explicitly
excludes raw JSON editing.

#### GitHub issue #35 — Generated-review acceptance criteria

[Acceptance criteria](https://github.com/gustavegv/resilnce/issues/35).
Exact former criterion replaced:

> - [ ] Repeated suggested names, invalid numbers/names/muscle codes, duplicate variant placements and concurrent library/review changes yield editable errors or re-review; they never silently merge distinct lifts or save partial plans.

The replacement separates invalid-payload recovery from ordinary field errors,
duplicate/review conflicts and protection against partial plans.

#### GitHub issue #35 — Implementation note

[Implementation comment](https://github.com/gustavegv/resilnce/issues/35#issuecomment-6556231474).
Exact former wording replaced:

> Invalid drafts remain editable, cancel saves nothing,

The comment now distinguishes valid form edits from invalid-output recovery and labels
the UI correction separately from the original integration's validation results.

#### `docs/exercise-centric/application-transition.md`

[Session creation and editing](exercise-centric/application-transition.md#stage-1-session-creation-and-editing)
now makes the recovery boundary explicit. No raw-editor instruction was present or
removed here. The existing requirement to review variants before saving remains.

### 13. Generated review used giant secondary and removal buttons

Removed the oversized Cancel generated session and Remove this suggestion presentation.
Cancel is a quiet secondary header action. During configuration, Add to session is
the larger action on the same row as a compact trash icon; removing a suggestion
retains accepted draft work. Edit suggested draft was removed entirely under problem 12.

**Finding:** a design gap. Cancellation and per-suggestion removal are useful behaviors;
their oversized presentation was never required.

#### `docs/exercise-centric/training-application.md`

[Quick fill and Program review](exercise-centric/training-application.md#quick-fill-and-program-review)
now specifies the compact action row and secondary cancellation. No giant-button
instruction was removed. The valid ability to remove a suggestion remains.

#### `docs/exercise-centric/application-transition.md`

[Creation guidance](exercise-centric/application-transition.md#stage-1-session-creation-and-editing)
now describes the action hierarchy. No wording prescribed the previous button sizes.

#### GitHub issue #35 — Generated review

[#35](https://github.com/gustavegv/resilnce/issues/35) now describes the action hierarchy.
No giant-button instruction was removed. Cancellation/retry safety remains.

### 14. Variant cards remained visible while creating a new variant

Removed the simultaneous picker and named-variant form. Create new variant switches
to its form; Back to variants returns to the cards. Entered values survive returning
and reopening, and keyboard focus returns to the picker.

**Finding:** an unspecified state transition. The sources listed choices without
defining when each view should be visible. None required showing both at once.

#### `docs/exercise-centric/training-application.md`

[Shared resolver contract](exercise-centric/training-application.md#personal-library-selection-and-mixed-drafts)
now states mutually exclusive views and value preservation. No wording removed.

#### `docs/exercise-centric/application-transition.md`

[Session creation and editing](exercise-centric/application-transition.md#stage-1-session-creation-and-editing)
now describes the transition and return path. No wording removed.

#### GitHub issue #25 — Stage 1 specification

[#25](https://github.com/gustavegv/resilnce/issues/25) now makes the state transition
explicit. No simultaneous-view requirement was removed.

#### GitHub issue #28 — Variant selection

[#28](https://github.com/gustavegv/resilnce/issues/28) now makes the state transition
explicit. The existing reuse/create alternatives remain; no wording required both views.

#### GitHub issue #35 — Generated review

[#35](https://github.com/gustavegv/resilnce/issues/35) now makes the state transition
explicit. No simultaneous-view requirement was removed. Define the transition and
return behavior when specifying alternative UI choices.
