---
name: ticket-authoring
description: Turn gathered requirements into a GitHub issue on the project board, with user story, capabilities, endpoint design, the TDD test plan and acceptance criteria. Use after brainstorming, before any code is written. Triggers on "write the ticket", "create the issue", "raise a ticket for", "put this on the board", "spec this out as a ticket".
---

# Ticket authoring

A ticket is a contract: it says why the work exists, what "done" means, and
which tests will prove it. If it cannot answer those, it is not ready and the
work should not start.

## Refuse to write a thin ticket

Before writing anything, check you can answer all of these. If you cannot, stop
and go back to `superpowers:brainstorming` — say which answers are missing.

- **Why** is this being done, for whom, and what happens if it is not?
- What **capabilities** does it add — the observable behaviours, not the code?
- What is explicitly **out of scope**?
- How will a **human** know it worked?

A ticket written from a one-line request is worse than no ticket: it looks like
a decision was made when none was.

## Check the story is actually buildable

A user story almost always assumes a capability the codebase does not have.
"Only the owner can see it" assumes identity. "Notify them" assumes a channel.
"Restore it" assumes soft delete. "Their records" assumes a user table.

**Before writing anything, list the assumptions the story makes and check each
one against the code.** Grep for it. If a prerequisite is missing, stop and say
so plainly, then propose the split:

> This needs an identity system and there isn't one — no users table, no
> session, no auth middleware in either backend. That's at least two tickets:
> authentication first, then per-record visibility on top of it. Do you want
> both, or should the first one stub identity behind an interface so the
> visibility work can proceed independently?

Then let the human decide. Options worth putting to them, in rough order of
how often they turn out to be right:

- **Split it.** Two tickets, the dependency first. Usually correct.
- **Stub the dependency behind an interface.** A `CurrentUser` port with a
  hardcoded implementation lets the real work proceed and get tested, and
  makes the second ticket a swap rather than a rewrite.
- **Cut the scope** so the dependency is not needed — a public/private flag
  with no per-user rules is a smaller, still-useful ticket.
- **Build it all in one.** Occasionally right, usually a ticket nobody can
  review.

Never write acceptance criteria that the code could not satisfy even when the
ticket is finished. A ticket that silently assumes a missing system is the most
expensive kind: it reads like a plan, so nobody questions it until someone is
three days into building it.

## The shape

Use `references/ticket-template.md`. Every section earns its place:

**User story** — one paragraph of prose, not the `As a… I want… so that…`
formula. Say who is stuck, what they are trying to do, and why the current
situation fails them. If you cannot name someone who benefits, question whether
the work should happen.

**Capabilities** — a list of observable behaviours. "The console can send a
PUT" is a capability. "Add a `Replace` method to the store" is an
implementation detail; it belongs in the design section or nowhere.

**Endpoint / interface design** — for API work, the method, path, request body,
every status code it can return and what each means. Decide `PUT` versus
`PATCH` **here**, not while coding: they are different contracts, and if you
need both, say so and give each its own acceptance criterion.

**TDD test plan** — the tests you will write, before you write them. Each is
one line describing the behaviour, tagged with which suite it belongs to:

```
- [go]     rejects a blank name with 422
- [go]     PUT resets fields the body omits
- [vitest] PATCH body omits unticked fields
- [manual] the console shows the reset field as blank in the table
```

Anything tagged `[manual]` must be justified — see `manual-test-design`. The
default assumption is that a behaviour **can** be automated and the tagger has
not thought hard enough.

**Acceptance criteria** — checkboxes, each independently verifiable by someone
who did not write the code. "Works correctly" is not a criterion. "Sending
`{\"name\":\"x\"}` to PUT leaves description empty" is.

**Manual verification checklist** — leave the heading in place with a note that
`review-handoff` fills it in. Writing the steps now, before the UI exists,
produces steps for a UI you imagined.

## Creating it

```bash
gh issue create \
  --title "<verb-first summary>" \
  --body-file <path> \
  --label enhancement
```

Then put it on the board in **backlog**, not ready:

```bash
./ghboard add <issue-number>
./ghboard move <issue-number> backlog
```

## Then stop

**Backlog means written but not approved.** Show the human the ticket URL and
ask them to review it. Do not create a branch, do not write a test, do not move
the card to Ready — moving it is their signal that the design is right.

This gate exists because a design problem costs one edit to fix at this stage
and a rewrite once the code exists. It is the cheapest review in the process,
so do not skip it to seem responsive.

When they approve:

```bash
./ghboard move <issue-number> ready
```

If they ask for changes, edit the issue with `gh issue edit` rather than
opening a new one — the ticket number is the thread the whole process hangs on.
