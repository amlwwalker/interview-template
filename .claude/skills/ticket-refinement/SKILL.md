---
name: ticket-refinement
description: Assess whether a Backlog ticket is ready to be worked on — read its body and comments, decide if the open questions have been answered, fold the answers back into the body, and either promote it to Ready or say exactly what is still outstanding. Use when asked whether a ticket is ready, to review or refine a ticket, to check what came back on one, or before promoting anything to Ready. Triggers on "is #7 ready", "check the ticket", "I commented on", "review the ticket", "can we start this", "refine", "groom the backlog", "what's blocking this".
---

# Ticket refinement

The question this answers is narrow: **could someone who did not write this
ticket pick it up and start, without asking anyone anything?** If not, say what
is missing. Nothing else.

## 1. Read the whole thing

```bash
./ghboard read <n>
```

Body **and** comments. The body is what you or someone else wrote when the
ticket was created; the comments are where it was argued with. Assessing the
body alone reaches a confident answer about a superseded plan.

## 2. Assess against these, in order

Work down. The first failure is the answer — do not list every fault, lead with
the one that blocks.

**Are the required sections present and filled?**
Why · Capabilities · Out of scope · Design · TDD test plan · Acceptance
criteria. A heading with a placeholder under it (`<verb-first summary>`,
`<behaviour>`) counts as missing, not present.

**Has every question raised in the ticket or its comments been answered?**
This is the one that actually blocks refinement, and the one most often
skipped. If the ticket asks "should this be per-user or global?" and no comment
answers it, the ticket is not ready however complete the sections look. Quote
the unanswered question back rather than paraphrasing it.

**Do the prerequisites exist in the code?**
Grep, do not assume. "Only the owner can see it" needs identity; "restore it"
needs soft delete. If a prerequisite is missing and the ticket does not say how
that is handled — split, stub, or cut — it is not ready.

**Is each acceptance criterion verifiable by someone who did not write the
code?** "Works correctly" is not. "PUT with `{"name":"x"}` leaves description
empty" is. A criterion that can only be checked by reading the implementation
is a criterion that will be marked done without being checked.

**Does the test plan name real suites?** `[go]`, `[vitest]`, `[manual]`. Every
`[manual]` needs a justification — see `manual-test-design`. A plan that is all
manual usually means nobody worked out how to test it.

**Is it still worth doing?** For anything that has sat a while: the reason it
was written may have gone away. Say so rather than refining a ticket that
should be closed.

## 3. Give a verdict

Two shapes, nothing in between. Do not hedge — "mostly ready" helps nobody
decide anything.

**Not ready:**

> #7 is not ready. Two things:
>
> 1. The ticket asks whether visibility is per-user or global and nothing
>    answers it. Everything downstream depends on that — per-user needs a
>    `records.owner_id` column and the global version does not.
> 2. Acceptance criterion 3 says "private records are protected", which cannot
>    be verified without reading the code. What is the observable behaviour —
>    a 404, a 403, or absence from the list?
>
> Answer those two and it is ready.

**Ready:**

> #7 is ready. The comments settled visibility as per-user, the criteria are
> each independently checkable, and `records.owner_id` is the only schema
> change. I will fold the comment decisions into the body and move it to Ready
> — say go.

## 4. Fold the answers into the body

Once it is ready, **before promoting it**, edit the body so the decisions live
there:

```bash
gh issue edit <n> --body-file <path>
```

A decision that exists only in comment #4 is a decision the next person will
miss. The body is the ticket; comments are how it got there.

Add a short line at the end recording what changed and why, so the edit is not
silent:

```markdown
---
_Refined after discussion: visibility is per-user, not global (see comments).
Acceptance criteria 3 rewritten as an observable 404._
```

## 5. Promote it

Only on the human's explicit go-ahead:

```bash
./ghboard move <n> ready
```

**Never promote a ticket you have just assessed as ready without asking.** The
assessment is advice; the promotion is a decision, and it is the gate that
stops work starting on a design nobody agreed to. You can be confident and
still ask.

## Refining several at once

```bash
./ghboard stale
```

Lists Backlog tickets that are old or structurally incomplete. Work through
them one at a time and give a verdict on each — but keep it short per ticket,
and lead with the single thing that blocks each one. A wall of analysis across
six tickets does not get read.

If a ticket has been sitting long enough that the reason is stale, say so and
offer to close it. A backlog nobody prunes is a backlog nobody trusts.
