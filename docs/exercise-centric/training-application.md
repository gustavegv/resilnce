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
Lists sort dated performance descending with session-ID ties; undated plans
follow in session-ID order. Never-run plans have `lastPerformedAt: null`. A start or untouched active workout
does not fabricate performance. Completed attempts in an active workout count.

`lastPerformedStatus` is `dated`, `never`, or `unknown`. A completed attempt's
timestamp takes precedence. Without one, an imported session uses only its own
mapping's normalized timestamp. A preserved null source date means `never`;
an old nonnull date without proven provenance means `unknown`, with a null
timestamp. The browser labels this “Last run unavailable”, separately from
“Never run”. New imported Session mappings freeze normalized source timezone
and their uniquely resolved instant together with the graph transaction. Blank
import policy means Europe/Stockholm; explicit UTC remains UTC. DST gaps/folds
block an unmapped row for explicit resolution. Mapped rows are skipped before
validation and later runs never rewrite metadata. No mapping, source key,
timezone policy, raw data, or run bookkeeping is exposed to application clients.

## Saved-session maintenance

`POST /api/training/sessions/{uuid}/update` accepts
`{operationId,expectedRevision,name,items}`. The items are the same explicit
create/reuse intents as creation, plus `{kind:"retainPlacement",placementId}`.
The ordered array is the complete next composition. Retaining a placement keeps
its UUID; omission removes only that placement. Duplicate placements/variants,
other-session placements and other-owner references are rejected atomically.
The session revision advances once. Existing sessions may become empty; they
cannot start a workout until a selectable variant is added. Plan changes affect
future starts and never alter an active workout's captured list/order.

`POST /api/training/sessions/{uuid}/delete` accepts
`{operationId,expectedRevision}` and returns `{operation,sessionId,deleted:true}`.
Deletion is rejected with `409 active-source-session` while a workout sourced
from the plan is active. Otherwise only the plan and its placements are deleted;
surviving workout source references become null. Library configuration, targets,
completed performance and durable import mapping tombstones remain. An identical
delete replay returns the original acknowledgment even though the plan is gone.
Both commands check exact receipts before revision guards and use the shared
owner-row transaction lock. Changed operation intent is `operation-mismatch`;
stale plan/selection reviews are `stale-revision`. The editor freezes the exact
command for Retry, retains the user's ordered draft after rejection, and blocks
navigation while a save is unresolved. Discard/reload is an explicit action.

Sessions retain the existing name-based categories and inclusive 14-day Recent
boundary; older dated and undated plans appear under Older in server order.

### Historical import policy correction

Migration 005 deliberately leaves existing provenance null. A new run, source
History timezone, import timestamp, or latest account-wide option is never
evidence of an old Session mapping's policy. Unknown dates stay unknown until an
operator independently establishes that exact mapping's historical policy.
There is no automatic backfill or ordinary application metadata-write endpoint.

If correction is needed, use the existing privileged operator boundary for a
separately reviewed manifest containing exact owner, migration version, Session
source keys, intended timezone, and expected preserved source-data digest. First
preview each raw date, chosen zone, unique instant, and unresolved DST issue.
Only a deliberate transaction holding the same owner lock may populate both
new columns, and only where both are still null and the preserved data matches
the manifest. Reject known/conflicting provenance and stale manifests. Do not
change source data, keys, destinations, domain data, aggregates or import counts.
This describes the narrowly scoped recovery procedure; no production inspection,
correction, or new general correction UI is part of Stage 1.

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
`types.ts`. Quick fill/Program wiring and generated-output validation remain #35.

## Session-based workout start and resume

The concrete `training.Service` exposes `StartWorkout(ctx, owner, StartWorkout)`,
`ActiveWorkout(ctx, owner)` and `GetWorkout(ctx, owner, workoutID)`. HTTP adapters
only decode/authenticate/translate. SQL, snapshotting, ownership, locks and receipts
remain private to the application module.

`POST /api/training/workouts/start` accepts exactly:

```json
{"operationId":"90000000-0000-4000-8000-000000000001","sessionId":"<uuid>","expectedSessionRevision":"1"}
```

It returns `{operation, workout}`. After receipt replay, the owner-row command
lock resolves an existing active workout before requested-plan revision/archive
checks. The response always identifies the actual source; requesting another
plan never replaces active work. A new start requires an owned, nonempty plan at
the reviewed revision and owned, unarchived exercises/variants. It captures the
current configuration, target identities, ordered placements and individual set
weights. Attempts and sets receive stable new UUIDs, with draft/pending state and
null recorded reps, weight, timestamp, RPE and notes. It never writes legacy
tables or the old account active-session pointer.

`GET /api/training/workouts/active` returns `{workout:null}` when none is active.
`GET /api/training/workouts/{uuid}` returns `{workout}` for that owned workout,
including ended workouts; absent/other-owner IDs return 404. Both reads use a
consistent repeatable-read snapshot. Closure and reads perform no lifecycle write.

`Workout` has `id`, `status` (`active`/`ended`), opaque `revision`, `origin`, nullable
`sourceSession:{id,name}`, `startedAt`, nullable `endedAt`, and ordered `exercises`.
Each `WorkoutExercise` has stable `id`, `variantId`, `position`, `state`
(`draft`/`completed`), `origin`, captured `configRevision`, `progressionMode`,
`repThreshold`, `autoIncrement`, `restSeconds`, nullable `completedAt`,
`recordedRpe` and `notes`, current `exercise` metadata, separate `currentVariant`
with future target sets, and ordered `sets`. Current names follow library renames;
the workout list/order and captured performance do not follow session edits.

Each `WorkoutSet` has `id`, nullable `targetSetId` provenance, `position`, `kind`,
`state` (`pending`/`performed`/`skipped`), captured `targetReps`/`targetWeight`,
nullable `recordedReps`/`recordedWeight`/`recordedAt`, and `lateWeightOverride`.
A performed zero remains zero. Future targets must never replace displayed actuals.
The workout revision is the mutable aggregate guard for subsequent recording,
configuration reconciliation, completion and quit commands. Attempt
`configRevision` records the variant configuration used by its captured settings;
`currentVariant.revision` can be newer. Those later commands reuse private
`beginCommand`, resolve receipts first, lock the affected workout before sorted
variant rows, and return authoritative `Workout` plus acknowledgment. Recording,
completion, free-mode start and quitting are not exposed by the start slice.

Start retries return the original acknowledgment with current state of the
original workout, even if it ended or another workout has since started. New
operation IDs requesting an active workout receive an acknowledgment targeting
that existing workout. Changed intent with the same operation ID is 409
`operation-mismatch`. Stale plan intent is 409 `stale-revision`, archived references
409 `resource-unavailable`, empty plans 422 `invalid-field`, invalid shape 400;
shared identity/model/origin/maintenance policy applies to every route.

The shared browser client adds `startWorkout`, `activeWorkout`, and `readWorkout`.
`ResumeWorkout.svelte` supplies the home Continue link using the returned workout
UUID. `WorkoutStart.svelte` captures exact start intent and opaque revision,
guards unresolved navigation, freezes plan selection and retries identically.
After a definitive rejection it can explicitly re-read/review the plan before a
new intent. A different already-active source is identified before Continue.
`/workout/[id]` checks account model availability and renders `WorkoutScreen.svelte`
with stable attempt/set keys and distinct target/pending/actual displays. It sends
no completion or quit request on mount/unmount and shows no default durable rating.

Additive migration `006_workout_rest.sql` captures attempt rest settings; existing
attempts backfill the current variant rest because no earlier snapshot existed.
Its columns and nonnegative constraint are included in migration readiness checks.
