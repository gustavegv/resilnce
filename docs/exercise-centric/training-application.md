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
surrounding whitespace. Additional explicit reuse/create-variant intents belong
to the following library-selection slice; a reuse item must never implicitly
override configuration.

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

Create has no expected existing revision because it depends only on new
aggregates. Subsequent reuse/edit operations must include the reviewed revisions
returned by reads. Unknown fields and unsupported deferred-feature controls are
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
has `createSession`, `listSessions`, `readSession` and `TrainingError` with status,
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
