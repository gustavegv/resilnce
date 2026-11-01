# Training application operations

The `training.Service` application boundary owns validation, ownership, revisions,
receipts and SQL transactions. Its HTTP adapter uses the shared application
identity/model/origin boundary. The browser cannot choose an owner. All writes
use the additive exercise-centric tables; they do not update legacy targets or
the legacy active-session pointer.

## Manual session creation

`POST /api/training/sessions/create` accepts one JSON command:

```json
{
  "operationId": "10000000-0000-4000-8000-000000000001",
  "name": "Upper body",
  "items": [
    {
      "kind": "createExercise",
      "name": "Bench press",
      "muscleGroupCodes": [],
      "configuration": {
        "prescribedSets": 3,
        "initialWeight": "40.50",
        "repThreshold": 12,
        "autoIncrement": "2.50",
        "restSeconds": 0
      }
    }
  ]
}
```

The tagged item explicitly creates a new exercise, its Standard variant, ordered
target sets and an ordered session placement in the same transaction as the
session. Existing-name collisions are conflicts, including case differences and
surrounding whitespace. Mixed drafts can also explicitly reuse an existing variant or create a named
variant under an owned exercise, using the item shapes documented below. Reuse
never overrides configuration.

Names are trimmed and contain 1–100 Unicode characters. New sessions contain
1–100 items. Prescribed sets are 1–20, threshold 1–99, initial weight 0–9999 kg,
increment 0.25–99 kg and rest nonnegative whole seconds within PostgreSQL integer
range. Omitted threshold/increment/rest default to 12/2.50/0. New variants use
average progression and initialize each target's reps to the threshold.

Weights and increments are decimal strings with at most two decimal places;
numeric JSON, excess precision, nonfinite/negative values and out-of-range values
are rejected. Responses return canonical two-decimal strings. Muscle groups may
be omitted or empty; explicit null, null/unknown/duplicate codes are invalid.
Responses use backend catalog order. No muscle-group picker is introduced.

Creation returns `{operation, session}`. `operation` contains `operationId`,
`acceptedRevision` and `committedAt`. Entity IDs are UUID strings, revisions are
opaque positive integer strings and timestamps have an explicit RFC 3339 offset.
The `session` contains its identity/name/revision/creation time,
`lastPerformedAt` and `placements`. Each placement exposes `id`, `variantId`,
`position`, separate `exercise` metadata and resolved `variant` configuration.
Variants include their own ID/name/revision/archive status, progression mode,
threshold, increment, rest, prescribed count and ordered target sets. Each target
has its own ID, position, target reps and target weight.

Creating entirely new exercises has no expected existing revision. Reuse and
named-variant creation include the reviewed revisions returned by reads. Unknown fields and unsupported deferred-feature controls are
rejected rather than silently ignored.

## Reads and retries

`GET /api/training/sessions` returns `{sessions: []}` or owned resolved sessions.
`GET /api/training/sessions/{uuid}` returns `{session}` and returns 404 for absent
or other-owner IDs. Both operations use a consistent repeatable-read snapshot.
Lists sort completed performance descending, then deterministic creation/ID ties;
never-run plans have `lastPerformedAt: null`. A start or untouched active workout
does not fabricate performance. Completed attempts in an active workout count.

Imported last-run fallback remains in the session-maintenance slice (#29). The
existing import map has a naive legacy timestamp without reliable per-session
source-timezone metadata; this slice does not reinterpret that timestamp as UTC.

The package-private command transaction locks the existing `User` account row to
serialize mutations per owner. Affected workout/variant rows must then be locked
in a consistent order. It checks
owner-plus-operation receipts before revision/finality checks, and stores kind,
target, fingerprint and acknowledgment in the mutation transaction. Identical
create replay resolves the existing identities and returns current authoritative
state with the original acknowledgment. Current revisions can therefore be
newer than `acceptedRevision`. Changed intent with the same operation ID is an
`operation-mismatch` conflict. There is no receipt expiry. Failed transactions
leave neither new records nor a successful receipt.

Errors use `{code, message, details?}`: malformed shape 400, invalid fields 422,
unauthenticated 401, unavailable owned resource 404, name/operation conflicts 409,
legacy-model unavailability 409, stale-client refresh 426, method 405 and shared
maintenance policy 503. Storage failures return a retryable 500 without SQL or
other-owner details.

Frontend contracts live in `apps/web/src/lib/training/types.ts`; the shared client
has `createSession`, `listSessions`, `readSession`, `searchExercises`,
`readExercise` and `TrainingError` with status,
code and optional details. The manual editor captures one operation ID and exact
intent, pauses editing while unresolved, retains the ordered draft on failure,
and retries the same command. A definitive rejection permits explicit return to
editing with a new operation ID. Unresolved saves guard in-app navigation and
request a browser unload warning. Only the server acknowledgment means Saved.

## Isolated verification

Use the existing guarded `internal/testdb` fixture. Integration runs require an
explicit `APPLICATION_TEST_DATABASE_URL` whose database name ends `_test` and
create/drop a unique disposable database per test. Missing configuration is
reported as skipped coverage. Never load `.env` or use `DATABASE_URL` in tests.

```sh
APPLICATION_TEST_DATABASE_URL='<explicit isolated PostgreSQL _test DSN>' go test -tags=integration ./api -count=1 -v
go test ./...
go vet ./...
npm run test --workspace web
PUBLIC_BACKEND_BASE_URL=https://backend.example npm run check --workspace web
PUBLIC_BACKEND_BASE_URL=https://backend.example npm run build --workspace web
```

## Personal library selection and mixed drafts

`GET /api/training/exercises?q=Bench` returns `{exercises: [...]}` with
`{exercise, variants}` entries. Search trims input and matches literal,
case-insensitive substrings within this owner's exercise names; empty input
lists the selectable library. `%` and `_` have no wildcard meaning. Exercises
and variants are sorted by name/ID and archived resources are excluded.
`GET /api/training/exercises/{uuid}` returns one owned selectable
`{exercise, variants}` for current review; other-owner/absent/archived parents
return 404. Both operations use a repeatable-read snapshot.

The same atomic session create command accepts three tagged item shapes:

```json
{"kind":"createExercise","name":"Squat","muscleGroupCodes":[],"configuration":{"prescribedSets":3,"initialWeight":"60"}}
{"kind":"createVariant","exerciseId":"<uuid>","expectedExerciseRevision":"2","name":"Heavy","configuration":{"prescribedSets":3,"initialWeight":"60"}}
{"kind":"reuseVariant","exerciseId":"<uuid>","variantId":"<uuid>","expectedVariantRevision":"5"}
```

Reuse references the existing variant and target IDs without copying or editing
settings. Configuration, suggested values, names and parent revisions cannot be
attached to reuse. Create-variant requires an owned, unarchived parent at its
reviewed revision. Named variants may deliberately have identical settings.
The exercise revision advances once when a command adds named variants beneath
it; variant configuration revisions remain independent. A command can create
multiple differently named variants under the same reviewed parent revision.

After the private owner-row command lock, existing exercises then variants are
locked in sorted ID order and all review guards run before inserts. A variant
must belong to its stated parent. Duplicate variant placements are invalid;
different variants of the same exercise are allowed. Trimmed case-insensitive
names remain reserved even when archived. Rejection rolls back every new
exercise, variant, target, placement, session and receipt. The original command
can replay after its own parent revision advanced. Canonical receipt fingerprints
preserve JSON field presence, so adding even a forbidden empty tagged field
cannot replay an accepted command as identical intent.

New conflict codes are `stale-revision` (409, re-review current settings) and
`resource-unavailable` (409, an archived reference). Name collisions keep
`name-conflict`; missing/other-owner/wrong-parent references keep `not-found`.

`DraftItemResolver.svelte` is the shared rendered review component for manual
and generated drafts. It takes optional `ExerciseSuggestion{name,configuration}`,
`onresolve(DraftSessionItem): boolean`, and `disabled`. A resolved item contains
one `SessionItem` intent plus optional display-only `exercise` and
`reviewedVariant` data; only `intent` goes into the server command. The component
shows all current per-set targets and progression/rest settings beside the
suggestion. Reuse emits only reviewed IDs/revision. Explicit creation emits the
reviewed configuration. The callback returns false to preserve a choice rejected
by composition validation (for example a duplicate variant).

The manual form also uses `hasInput()`, `add()` and async `edit(item)` to capture
unadded explicit creation on Finish and re-review an existing item. Editing reuse
fetches current settings but requires another explicit Reuse unchanged choice;
it never silently rebases a stale draft. Saving/retry still freezes one exact
ordered command. `searchExercises` and `readExercise` are in the shared client;
`ExerciseSelection`, `ExerciseSuggestion`, and the three intent types are in
`types.ts`.

## Quick fill and Program review

Both existing producer entry points still hand an exercise suggestion array to
`/create/manual?quickload=1` through the tab's `workoutPayload`. This handoff is
untrusted review data, not `SessionItem` intent. Quick fill keeps its existing AI
options; Program keeps its previews and Edit and add action. No generated item is
inserted directly into the saved draft.

The common review validates the full array before resolving any item: 1–100
objects, trimmed nonempty names up to 100 Unicode characters, whole set counts
1–20, numeric weights 0–9999 with at most two decimal places, whole thresholds
1–99, and numeric increments 0.25–99 with at most two decimal places. Omitted
threshold and increment retain 12/2.50 defaults; null and unsupported fields are
errors. The producer shape remains `{name,sets,weight,repThreshold?,autoIncrease?,
muscleGroupCodes?}`. Malformed/invalid output remains visible in an editable JSON
field with a correction action; no domain mutation is issued. Applying a
corrected draft explicitly restarts review. A suggestion can also be removed.

`GET /api/training/catalog` returns `{muscleGroups:[{code,label,sort_order}]}`
from the backend's predefined catalog in declared order, under the shared
authenticated/model/method policy. Generated nonempty muscle code arrays are
checked against this catalog before review. Null, duplicate, unknown or nonstring
codes are errors. The resolver displays and carries valid codes only when
creating a new exercise; reuse/named-variant creation preserve existing exercise
groups. The save command independently validates catalog membership atomically.
There is no muscle-group picker or inference from marketing labels/free text.

Each suggestion has a stable local review ID and is resolved independently in
source order through `DraftItemResolver`, using its suggestion prop and boolean
`onresolve` contract. Current per-set targets/settings appear beside suggestions.
The explicit choice is reviewed reuse unchanged, a named variant with reviewed
settings, or a new exercise/Standard. Duplicate variants and normalized creation
names retain the current choice with an editable error. Different variants of
one exercise remain distinct. The existing sortable cards retain final order.

Finish is blocked until every retained suggestion is resolved. It uses the same
single atomic create command and immutable operation ID as manual creation;
neither producer orchestrates separate library writes. Network/server failure
retains the draft, pauses edits, protects navigation, and retries identical JSON.
Definitive rejection permits editing with a new operation ID, and stale reuse
requires an explicit current-settings review. Cancel before saving removes the
tab handoff and leaves no library/session records; unresolved saves must first
be resolved. Only acknowledged success clears the handoff as Saved.

`POST /ai/quick` continues to use the backend-selected account/quota key and
25-call limit. Generation only returns text for review. Provider errors return
`ai-generation-failed` rather than a successful empty draft. The Quick fill UI
recognizes `ai-quota-exceeded` from the authoritative 403 response; network and
other failed generations remain errors without creating a session.
