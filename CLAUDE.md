# CLAUDE.md

How this project is built. The rules are here; the process lives in skills.

## Non-negotiables

1. **TDD, always.** Write the failing test, watch it fail, then make it pass.
   A test that has never been red proves nothing. No production code is written
   before a test that demands it.
2. **Go tests are Go's standard `testing` package.** Not testify, not ginkgo.
   Table-driven where it helps, plain where it doesn't.
3. **Everything TypeScript is tested with Vitest** — the TypeScript API and the
   frontend both. Never Jest.
4. **Every piece of work has a ticket on the project board before code exists.**
5. **If a behaviour can only be confirmed by a human in a browser, it still gets
   a written test** — a plain-language manual step, as a GitHub checkbox on the
   ticket. "Can't be automated" is never the same as "untested".

## Commands

`make` on its own lists everything. You should not need to ask Claude to run a
test for you — these are the whole surface:

    make test                  every unit suite
    make test-backend          Go only, no database needed
    make test-backend-ts       TypeScript API only
    make test-frontend         frontend only
    make test-integration      Go against crud_test
    make test-integration-ts   TypeScript against crud_test
    make test-all              everything
    make watch-frontend        watch mode (the TDD loop)
    make lint                  vet, gofmt, tsc --noEmit
    make ci / make ci-ts       exactly what GitHub runs

CI calls these same targets. If it passes locally it passes there, and if the
two ever disagree that is a bug in the Makefile, not in the test.

**Integration tests run against `crud_test`, never `crud_dev`.** They TRUNCATE
before every test. If you add a suite that touches the database, point it at
`TEST_DB_URL` — pointing it at the dev database destroys the seed data and the
next person to open the app thinks the API is broken.

## Structure

There are **two interchangeable backends** behind one schema and one frontend.
Both serve the same API on :8080; run one or the other. They are kept
behaviourally identical, and the tests are what enforce that — the frontend's
browser suite passes unmodified against either.

    migrations/           the schema. ONE copy, shared. Never fork this.
    scripts/initdb.sh     creates the database, applies migrations, writes both .env files

    backend/
      cmd/api/              entrypoint, router, middleware, CORS
      internal/config/      environment parsing
      internal/database/    pgx pool
      internal/httpx/       JSON and error helpers — one error shape, everywhere
      internal/record/      the resource: model, store, handlers, tests

    backend-ts/           TypeScript + Fastify — the same API, same layering
      src/config.ts         environment parsing
      src/db.ts             pg pool
      src/httpx.ts          the error shape, identical on the wire to Go's
      src/app.ts            buildApp() — routes, CORS, error handler
      src/record/           the resource: model, store, routes, tests

    frontend/
      src/api/              typed client; returns results, never throws
      src/components/       one panel per operation, plus ui.tsx primitives
      src/styles/tokens.ts  design tokens — never retype a hex value
      src/test/setup.ts     Testing Library wiring

### Adding a new resource

**Add it to both backends, or to neither.** They are only interchangeable
while they agree, and a resource that exists in one is a trap for whoever
switches. Write the Go tests and the Vitest tests with the same names so a
reviewer can diff them.

Copy the shape of `internal/record` (or `src/record/`), do not edit it. A resource is a package
holding its model, its `Store` interface, a Postgres implementation of that
interface, and its handlers. Then:

1. A migration in `migrations/` at the repo root, numbered next in sequence.
   There is one schema; never fork it per backend.
2. Handlers depend on the `Store` **interface**, never on a pool directly.
   This is what lets handler tests run with no database — `httptest` in Go,
   `app.inject()` in Fastify.
3. Go gets a compile-time `var _ Store = (*PostgresStore)(nil)` assertion;
   TypeScript gets `class PostgresStore implements Store`.
4. Mount it under `/api/v1` in `cmd/api/main.go` and in `src/app.ts`.
5. **Add every verb it serves to the CORS allowed-methods list, in both
   backends.** Forgetting `PUT` or `PATCH` there is the classic bug: `GET` and
   `POST` keep working, so the router looks fine while the browser rejects the
   preflight.

Two TypeScript-specific traps, both already handled in `backend-ts` and both
worth knowing before you add code:

- **ajv coerces by default.** Fastify would quietly turn `{"name": 42}` into
  `{"name": "42"}` and accept it, where Go rejects it. `coerceTypes: false` in
  `app.ts` is what keeps the two agreeing.
- **`setErrorHandler` must be registered before the routes.** A child context
  inherits whichever handler exists when it is registered, so setting one
  afterwards silently leaves routes on Fastify's default — which echoes the raw
  error message to the client.

### Adding a screen

Follow the **`wireframe-ui`** skill — it is installed globally and defines the
house style. Square corners, hairline borders, no shadows, monochrome except
for genuine status. Read values from `src/styles/tokens.ts`; never retype a hex.
`src/components/ui.tsx` already holds the patterns this project uses.

Query tests by **role, label or `data-testid`** — never by CSS class or inline
style. The whole console was restyled from a bespoke stylesheet to wireframe-ui
without a single test changing, because the tests describe behaviour rather
than appearance. Keep it that way.

### API conventions

- Partial update is `PATCH`, full replacement is `PUT`. Absent must be
  distinguishable from "set to the zero value": Go uses pointer fields, and
  TypeScript uses optional properties, where `'active' in body` is the
  equivalent of a non-nil check. If you add one verb you almost certainly want
  both, and a test for each.
- The collection endpoint returns an empty array, never null — `make([]T, 0)`
  in Go, `rows.map(...)` in TypeScript — so the client needs no null check.
- Errors are always `{"error": "...", "details": {"field": "..."}}`. Validation
  failures are `422`; malformed input is `400`.
- Internal errors are logged in full and returned as a bare
  "internal server error". There is a test asserting nothing leaks.

## Branch model

    main   production. Only ever receives PRs from dev.
    dev    integration. Anything in "In review" is already here.
    feature/<issue-number>-<slug>   one per ticket.

## Commit convention — this is the red-green evidence

    test(red):   <test name>       the failing test, alone, nothing else
    feat(green): <what it does>    the implementation that turns it green

Two commits minimum per behaviour. The SHA pair is the audit trail: a reviewer
can check out the red commit and watch the test fail. Do not squash them.

## Where the process lives

This file states the rules. The process is in `.claude/skills/`:

| Phase | Skill |
|---|---|
| Gather requirements | `superpowers:brainstorming` (global) |
| Turn them into a ticket | `ticket-authoring` |
| Decide what needs a human | `manual-test-design` |
| Implement | your installed TDD skill, plus the rules above |
| Hand off for review | `review-handoff` |
| Ship to production | `release-to-production` |

`feature-workflow` is the entry point — it works out which phase you are in and
routes to the right one. Board, branch and command bindings for this repo live
in `.claude/workflow.config.json`; the skills themselves are repo-agnostic and
can be copied to any project that has a board.

## Working offline

Board and PR steps need network and `gh` auth. Skipping them locally is fine.
Skipping the ticket is not — write it when you reconnect, before the work counts
as started.
