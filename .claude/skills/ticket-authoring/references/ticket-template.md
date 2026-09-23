# <Verb-first summary, e.g. "Add PUT for full replacement of a record">

## Why

<One paragraph of prose. Who is stuck, what they are trying to do, why the
current situation fails them, and what happens if this is not built. Name a
person or role. If you cannot, question whether the work should happen.>

## Capabilities

Observable behaviours this adds. Not implementation steps.

- <e.g. A client can replace a record wholesale, resetting fields it omits>
- <e.g. The console can send PUT and shows the resulting row>

## Out of scope

Say this explicitly. It is the section that prevents scope creep later.

- <e.g. Bulk replacement of several records in one call>

## Design

### Endpoints

| Method | Path | Body | Returns | Notes |
|---|---|---|---|---|
| `PUT` | `/api/v1/records/{id}` | `{name, description, active}` — all required | `200` / `404` / `422` | Full replacement. Omitted fields reset. Idempotent. |

### Decisions

- <e.g. PUT takes plain values, not pointers: absent must mean "reset", which
  is exactly what JSON decoding into a value type gives you.>
- <Anything a reviewer might otherwise query, answered here once.>

## TDD test plan

Written before the code. Each line is one behaviour and names its suite.
`[manual]` entries need a justification — see the `manual-test-design` skill.

- [ ] `[go]` <behaviour>
- [ ] `[go]` <behaviour>
- [ ] `[vitest]` <behaviour>
- [ ] `[manual]` <behaviour — and why it cannot be automated>

## Acceptance criteria

Each independently verifiable by someone who did not write the code.

- [ ] <e.g. `PUT /records/1` with `{"name":"x"}` leaves description empty and active false>
- [ ] <e.g. `PUT` with a blank name returns 422 and does not modify the row>
- [ ] <e.g. Both suites pass via `make test`>

## Manual verification

_Filled in by `review-handoff` once the work is on the integration branch._
