---
name: manual-test-design
description: Decide which behaviours genuinely cannot be covered by an automated test, and write the layman's manual steps for those. Use when planning a ticket's test coverage, or when preparing the human verification checklist for review. Triggers on "manual test", "how do we test this", "can this be automated", "verification checklist", "QA steps", "acceptance steps".
---

# Manual test design

Two jobs: decide what genuinely needs a human, and write steps a human can
actually follow.

## What needs a human

Short list. Treat it as closed — if a behaviour is not on it, the automated
test exists and has not been thought of yet.

- **Visual judgement.** Does the layout hold at this width, does the hierarchy
  read, does it look right. (Note: *rule* conformance is automatable — "no
  border radius anywhere", "page title is 28px" are assertions, not opinions.)
- **Real-browser behaviour that a test double replaces.** Cross-origin
  preflights, cookie flags, redirects to third parties, clipboard, file
  pickers, notification permission.
- **Something outside the system.** A payment provider's sandbox, an email
  actually arriving, an OAuth consent screen.
- **Feel.** Is the loading state long enough to notice, does the error message
  make sense to someone who does not know the code.

## What does not

Be sceptical of these, they are the common excuses:

| "Needs a human because…" | Actually |
|---|---|
| it involves the database | integration test with a real Postgres |
| it's a UI interaction | Testing Library — click it and assert |
| it's about CORS | assert the preflight response headers directly |
| there are lots of combinations | table-driven test |
| it's hard to set up | that difficulty is the design telling you something |

A behaviour that ends up manual because it is *hard* to automate is a cost you
pay on every release, forever. Say so out loud before accepting it.

## Writing the steps

For someone who has never seen the code. No selectors, no jargon, no internal
names.

**One observable outcome per step.** If a step has two assertions, it is two
steps — otherwise a half-pass has nowhere to go.

**Say where to start.** "Open <dev url>, go to the UPDATE tab." Not "navigate
to the update panel". If `environments.dev.url` is null in the config, say so
rather than writing "open dev" and hoping.

**Say what to look at, and what it should be.** The reader should not need
judgement about whether it passed.

Good:

```
- [ ] On the UPDATE tab choose PUT, put 3 in the id box, type "replaced" in
      name, leave description empty, send. In the table at the bottom, row 3's
      description is now blank.
```

Bad:

```
- [ ] Verify PUT semantics work correctly
```

The second one cannot fail — anyone can convince themselves it passed.

**Include the negative where it is the point.** If the interesting behaviour is
that something *doesn't* happen, say what would be wrong:

```
- [ ] Repeat on the PATCH tab with only "name" ticked. Row 1's description is
      unchanged. (If it goes blank, PATCH is behaving like PUT — that's the bug
      this ticket fixes.)
```

## Where they go

- **At ticket time**: named in the TDD test plan as `[manual]` lines, with the
  justification. No steps yet — the UI does not exist, so any steps would be
  for an imagined one.
- **At review time**: written out in full as GitHub checkboxes, appended to the
  issue body by `review-handoff`, so the person verifying ticks them on the
  card itself.
