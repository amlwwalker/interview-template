---
name: feature-workflow
description: Entry point for any piece of work on a repo that tracks features on a GitHub project board. Use when starting a new feature, resuming one, asking "what's next", or when unsure which phase a piece of work is in. ALSO use whenever someone describes a feature they want, in any form — a user story ("as a user I want..."), a capability, a bug, or a loose idea — because work here starts with a ticket, never with code. Routes to ticket-authoring, the TDD loop, review-handoff or release-to-production. Triggers on "start work on", "new feature", "what should I do next", "pick up", "resume", "where is this up to", "as a user I want", "we need to be able to", "can you add", "I want it to".
---

# Feature workflow

The spine. This skill does not do the work — it works out which phase you are
in and hands off to the skill that does.

## Before anything else

1. **Read `.claude/workflow.config.json`.** If it is missing, this repo has not
   been set up for this process. Say so and offer to create one from
   `workflow.config.schema.json`; do not guess board numbers or column names.

2. **Resolve the repository from git, not from config:**

   ```bash
   gh repo view --json nameWithOwner -q .nameWithOwner
   ```

   The config deliberately does not store this, so a fork or rename cannot
   leave a stale value.

3. **Validate the board** with `scripts/board.sh validate`. It lists the live
   Status options and compares them to `project.columns`. If they disagree it
   stops. Do not "helpfully" create the missing column — a typo in the config
   and a genuinely absent column look identical from here, and one of those is
   fixed by editing a file while the other needs a human decision.

4. **Check `requiredSkills` are available.** If one is missing, name it and say
   what it is for, then ask whether to continue without it. A missing
   `wireframe-ui` means any UI you write will not match the house style; a
   missing brainstorming skill means requirements will be thinner than usual.

If `gh` is absent or unauthenticated, say so plainly and continue in local
mode: branches, TDD and commits all work offline. Board and PR steps are
deferred, not skipped — note which ones are outstanding.

## The phases

| Phase | State on disk / board | Skill |
|---|---|---|
| 1. Design | No ticket exists | `superpowers:brainstorming`, then `ticket-authoring` |
| 2. Approve | Ticket in Backlog | **The human approves it.** Then move to Ready |
| 3. Implement | Ticket in Ready | TDD loop (see below) |
| 4. Review | Branch green, no PR | `review-handoff` |
| 5. Release | Card in Done | `release-to-production` |

## Routing

Work out the phase from evidence, in this order, and stop at the first match:

1. **On a feature branch?** (`git branch --show-current` matches
   `branches.featurePrefix`.) Extract the issue number from the branch name.
   - Uncommitted changes or no PR yet → phase 3 or 4. Run the test commands
     from `tests`. All green and something to ship → `review-handoff`.
     Anything red → stay in the TDD loop and fix it.
2. **Card in Done with unreleased commits on the integration branch?**
   → `release-to-production`.
3. **The user named a ticket or issue number?** Read it, check its column,
   route accordingly.
4. **Nothing else matches** → phase 1. Requirements first.

## The TDD loop

This project's global TDD skill owns the discipline. What this project adds:

- **Backend tests are Go's standard `testing` package. Frontend tests are
  Vitest.** Never substitute another framework.
- **Run tests via the make targets** in `tests.*.command`, never by calling
  `go test` or `vitest` directly. The targets are what CI runs; bypassing them
  is how local and CI drift apart.
- **Two commits per behaviour, minimum:**

  ```
  test(red):   <test name>       the failing test, alone, nothing else
  feat(green): <what it does>    the implementation that turns it green
  ```

  The SHA pair is the red-green evidence. A reviewer can check out the red
  commit and watch it fail. Do not squash them.

- **Watch the test fail before writing the implementation.** Not "assume it
  would fail". A test that passes on first run is testing nothing, or testing
  something that already worked — either way it is not driving the design.

## Branch naming

```
<featurePrefix><issue-number>-<short-slug>
```

e.g. `feature/42-put-endpoint`. The issue number is how every later step finds
its way back to the ticket, so a branch without one breaks the chain.

## What this skill will not do

- Move a card to Done. Only the human does that; it is the signal that
  authorises a release.
- Start implementation on a ticket still in Backlog. Backlog means written but
  not approved.
- Invent board columns, issue numbers or test results.
