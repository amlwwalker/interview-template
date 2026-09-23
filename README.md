# Go + chi + Postgres + React starter

A complete vertical slice with no domain attached, plus the process for adding
to it. Go REST API over Postgres, React frontend that is literally a CRUD
console — one tab per operation, showing the exact request sent and the raw
response received.

The resource is deliberately anonymous: a row with a `name`, a `description`
and an `active` flag. Enough to exercise every verb, and nothing to unlearn.

The frontend runs on a different origin to the API, so CORS is exercised for
real rather than hidden behind a dev proxy.

```
migrations/       the schema — ONE copy, shared by both backends
scripts/          initdb.sh: creates the database, applies migrations
backend/          Go — chi, pgx, graceful shutdown
backend-ts/       TypeScript — Fastify, pg, same API
frontend/         Vite, React, TypeScript, Vitest
.claude/          the development process: config + skills
.github/          issue and PR templates, CI
CLAUDE.md         the rules — read this first
```

**Two interchangeable backends.** Identical API, identical error bodies, one
schema between them. Run whichever the conversation calls for; the frontend
does not know or care which is answering. That is not an assertion — the
frontend's 29-check browser suite passes unmodified against both, which is what
makes the claim worth anything.

## Requirements

Go 1.24+, Node 18+, and a local Postgres (Homebrew or Postgres.app).

## Run it

```bash
make setup          # both backends, frontend, and both databases
make dev-backend    # :8080   Go        (one terminal)
make dev-frontend   # :5173             (another)
```

For the TypeScript API instead, stop the Go one and run `make dev-backend-ts`.
Both bind :8080, so exactly one at a time.

Open http://localhost:5173. Three seeded rows appear at the bottom.

If `make setup` cannot find Postgres:

```bash
brew install postgresql@16 && brew services start postgresql@16
```

## Commands

`make` on its own lists everything. You should never need to ask Claude to run
a test — that is slow and burns tokens for something a shell does instantly.

```
make test                  every unit suite
make test-backend          Go only, no database
make test-backend-ts       TypeScript API only
make test-frontend         frontend only
make test-integration      Go against the test database
make test-integration-ts   TypeScript against the test database
make test-all              everything
make watch-frontend        watch mode — the TDD loop
make lint                  vet, gofmt, tsc --noEmit
make ci / make ci-ts       exactly what GitHub runs
```

CI calls these same targets. If they ever disagree with your machine, the
Makefile is wrong, not the test.

**Two databases.** `crud_dev` is what the app runs against; `crud_test` is what
the integration suites use. They truncate before every test, so pointing them at
`crud_dev` would silently wipe your seed data every time you ran them — and then
the app looks broken for no visible reason. `make setup` creates both.

## Tests

| Suite | Count | What it covers |
|---|---|---|
| Go unit | 15 | Every handler and status code, against an in-memory fake store |
| Go integration | 12 | The actual SQL, against a real Postgres |
| TypeScript API unit | 28 | The same cases as the Go suite, plus CORS preflights |
| TypeScript API integration | 14 | The same SQL, plus pool recovery after a connection drop |
| Frontend (Vitest) | 26 | API client parsing, PATCH/PUT body construction, rendering |
| Browser (Playwright) | 29 | All six verbs cross-origin, against **either** backend |

The Go unit tests prove the HTTP layer; the integration tests prove the
`COALESCE` behaves. Those are different claims and both are worth pinning — a
fake store that drifts from the SQL will keep the unit tests green while
production is wrong.

Frontend tests query by **role, label or `data-testid`** — never by CSS class.
The whole console was restyled from a bespoke stylesheet to the `wireframe-ui`
system without a single test changing.

## API

Base URL `http://localhost:8080`.

| Method | Path | Returns | Notes |
|---|---|---|---|
| `GET` | `/healthz` | 200 | liveness; does not touch the DB |
| `GET` | `/readyz` | 200/503 | readiness; pings the DB |
| `GET` | `/api/v1/records` | 200 | always an array, `[]` when empty |
| `POST` | `/api/v1/records` | 201 | plus a `Location` header |
| `GET` | `/api/v1/records/{id}` | 200/404 | |
| `PUT` | `/api/v1/records/{id}` | 200/404 | **full replace** — omitted fields reset |
| `PATCH` | `/api/v1/records/{id}` | 200/404 | **partial** — omitted fields untouched |
| `DELETE` | `/api/v1/records/{id}` | 204/404 | |

```bash
# same body, same URL, different verb, different outcome:
curl -X PATCH localhost:8080/api/v1/records/1 -H 'Content-Type: application/json' -d '{"name":"x"}'
# -> description and active unchanged
curl -X PUT   localhost:8080/api/v1/records/1 -H 'Content-Type: application/json' -d '{"name":"x"}'
# -> description reset to "", active reset to false
```

## How work gets done here

Read `CLAUDE.md`. In short: every feature starts as a ticket on the project
board, is built test-first with the failing test committed separately as
evidence, lands on `dev` before its card says "In review", and reaches `main`
only after a human has ticked the manual checklist and moved the card to Done.

The process lives in `.claude/skills/` and is repo-agnostic. Everything
specific to this repository — board number, column names, branch names, test
commands — is in `.claude/workflow.config.json`. To use the same process on
another project, copy `.claude/` across and edit that one file.

Two global skills are expected to be installed: `superpowers:brainstorming` for
requirements, and `wireframe-ui` for the house UI style. They are declared in
`requiredSkills` and checked at the start of a session rather than assumed.

## Why two backends

Because the honest answer to "Go or TypeScript?" is "it depends", and having
both lets you show the reasoning rather than assert it.

They share a schema, a frontend and an error contract, so the differences that
remain are the interesting ones:

- **Absent vs zero.** Go needs `*string` to tell `{"active": false}` from `{}`.
  TypeScript gets it free from optional properties. Same contract, different
  mechanism.
- **Pool failure.** pgxpool absorbs a dropped connection internally. Node's
  `pg` emits an `error` event on the idle client, and an unhandled one
  terminates the process — so `db.ts` has a handler the Go code does not need.
  There is an integration test that kills the connections and asserts recovery.
- **Type coercion.** Fastify's ajv coerces by default, so `{"name": 42}` would
  be silently accepted as `"42"`. `coerceTypes: false` restores parity with
  Go's decoder. Caught by a test, not by inspection.
- **Handler registration order.** Fastify children inherit the error handler
  present when they are registered, so `setErrorHandler` after the routes
  silently leaks raw error messages. Also caught by a test.

## Things worth being able to explain

**PUT vs PATCH.** `PUT` is a replacement: the body is the complete new state,
so anything omitted resets, and sending it twice leaves the same row. `PATCH`
is a diff: only the keys present are touched. That shows up directly in the
types — `ReplaceParams` uses plain values so absent fields decode to zero
values and the SQL writes them unconditionally; `UpdateParams` uses `*string` /
`*bool` so absent decodes to `nil` and `COALESCE($n, column)` leaves it alone.
Without the pointers you could not send `{"active": false}` and have it mean
anything. On the console's UPDATE tab, tick and untick fields under PATCH and
watch the request body change.

**CORS and the preflight.** Different ports are different origins, so every
request is cross-origin. `chi/cors` is middleware registered *before* the
routes, because it answers the browser's `OPTIONS` preflight itself. Every verb
must appear in `AllowedMethods` — forgetting `PUT` is the classic bug, because
`GET` and `POST` keep working so the router looks fine. `localhost` and
`127.0.0.1` are different origins, which is why both are allowlisted. And
cross-origin JS can only read headers named in `ExposedHeaders`, which is why
`Location` is listed.

**Why handlers depend on a `Store` interface.** Not on `*pgxpool.Pool`. That is
what lets the handler tests run the full HTTP surface against an in-memory fake
in about 10ms, with no database and no Docker.

**Graceful shutdown.** `signal.NotifyContext` catches SIGINT/SIGTERM,
`srv.Shutdown` drains in-flight requests, then the pool closes. Without it you
drop live requests on every deploy.

**DELETE and idempotency.** Deleting twice is safe but the second call returns
404. The *effect* is idempotent; the status code is not.

## Deliberately missing

No auth, no pagination, no rate limiting, and migrations are a shell script
rather than [goose](https://github.com/pressly/goose) or
[golang-migrate](https://github.com/golang-migrate/migrate) — fine at this
size, first thing to replace at any real size. The console re-fetches rather
than caching; React Query would be the move if it grew.
