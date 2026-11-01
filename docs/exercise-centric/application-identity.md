# Application identity and model selection

The backend selects the owner for `/api/me`, training, Stats, and AI. Request query
parameters and bodies cannot select an account. Normal identity requires both a
valid signed cookie and a live server session whose email matches the cookie.
The session supplies the display name.

| Backend setting | Default | Behavior |
| --- | --- | --- |
| `DEVELOPMENT_ACCOUNT_ENABLED` | `false` | Forces the dedicated development identity without OAuth configuration or provider discovery. |
| `DEVELOPMENT_ACCOUNT_EMAIL` | `development@resilnce.invalid` | Dedicated account email; must be a plain `.invalid` address to keep real accounts and importer administration separate. |
| `APPLICATION_MODEL` | `legacy` | `legacy` or `exercise-centric`; invalid values stop startup. Applies to normal accounts. The development account always selects `exercise-centric`. |
| `TRAINING_WRITE_MAINTENANCE` | `false` | Rejects training mutations in either model; reads and separately authorized internal setup/import access remain available. |

Development startup inserts the account only if missing. It never updates existing
account/training data, creates a login session, imports legacy data, or migrates
the database. Turning development mode off restores normal authentication without
deleting the account. Database schema setup remains an explicit operation.
Redis is needed for AI quota accounting and for normal/importer login sessions;
development account/training access itself can start without Redis. With Redis
unconfigured, AI returns `503 ai-unavailable`.

`GET /api/me` returns `id` (the account email), `email`, `name`,
`developmentMode`, and `capabilities` containing `applicationModel`,
`trainingWrites`, and `writeMaintenance`. This response is not cached. The shared
frontend account store supplies the persistent development badge and account page.
AI quota keys are derived from the account email, independent of provider subjects,
login-session expiry, or application restart. The allowance is 25 calls; Redis must
retain its data to retain usage across Redis restarts.

`application.Boundary.Handler` authenticates a request and binds its identity to
context. New handlers read it with `application.FromContext`, use
`ExerciseCentric` model policy, and mark training mutations with `write=true`.
Normal-account new operations on the legacy model return `409 model-unavailable`.
All legacy training mutations return `426 upgrade-required` before decoding or
writing for switched accounts, including requests without a version header. The
frontend displays a persistent Refresh action when that response is received.
Maintenance returns `503 write-maintenance`. Mutations require the configured
application Origin; CORS alone does not authorize writes. Other shared HTTP
errors contain `code`, `message`, and optional `details`; wrong methods return 405.

The internal porting API always performs its own signed-cookie, live-session,
verified-email, allowlist, and mutation-origin checks. Forced development identity
is never passed to that administration policy.

For isolated HTTP integration tests, `internal/testdb.New` requires an explicit
`APPLICATION_TEST_DATABASE_URL` whose database name ends in `_test`, checks
readiness, and creates/drops a unique disposable database per case. It never loads
`.env` or falls back to `DATABASE_URL`. Run only the intended integration packages:

```sh
cd apps/server
APPLICATION_TEST_DATABASE_URL='postgres://postgres:local-test-password@127.0.0.1:55432/resilnce_application_test?sslmode=disable' go test -tags=integration ./api -count=1
```

The frontend rendered test seam is `npm run test --workspace web`, using Vitest
and jsdom with Svelte's real browser renderer. Production build/check commands
still require their normal `PUBLIC_BACKEND_BASE_URL` setting.
