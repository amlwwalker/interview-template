---
name: review-handoff
description: Move an implemented feature to In review — run both suites, publish the TDD test inventory, open and merge the PR to the integration branch, and append the layman's manual checklist to the ticket. Use when a feature is finished and green. Triggers on "ready for review", "hand this off", "move to in review", "open the PR", "this is done", "finished the feature".
---

# Review handoff

"In review" means a human can go and try it. That is a promise about the state
of the integration branch, not about your intentions — so the card moves last,
after the code is genuinely there.

## 1. Refuse unless it is actually green

Run the real commands from `.claude/workflow.config.json`:

```bash
make test              # both unit suites
make test-integration  # if the change touched storage
make lint
```

If anything is red, stop. Do not open a PR "for early feedback" on a red
branch — a reviewer cannot tell a known failure from a new one, so the review
is worthless and their time is spent anyway.

## 2. Check the red-green evidence exists

```bash
git log --oneline "$(git merge-base HEAD origin/dev)"..HEAD
```

You are looking for `test(red):` commits paired with `feat(green):` ones. If
every test arrived in the same commit as its implementation, TDD did not
happen. Say so plainly rather than writing a handoff that implies it did — the
point of the evidence is that it can be checked, which means it can also be
absent.

## 3. Build the test inventory

```bash
.claude/skills/review-handoff/scripts/test-inventory.sh \
  --since "$(git merge-base HEAD origin/dev)"
```

Emits the markdown table: test name, one-line description, permalink. If it
reports Go tests missing their description comment, **add the comments** — the
convention is one `//` line directly above `func TestX`. Do not hand a reviewer
a table with blanks in it.

## 4. Open the PR to the integration branch

```bash
gh pr create --base dev --head "$(git branch --show-current)" \
  --title "<same summary as the ticket>" \
  --body-file <path>
```

The body carries: `Closes #<issue>`, one paragraph on what changed and why, the
test inventory table, and the red→green SHA pair.

## 5. Wait for CI, then merge

```bash
gh pr checks --watch
gh pr merge --squash --delete-branch
```

Squashing the PR is fine — the red/green pair survives in the PR's own commit
list, which is where a reviewer looks. What must not happen is squashing them
together *before* the PR exists.

If CI fails, fix it on the branch. Do not move the card.

## 6. Append the manual checklist to the ticket

Use `manual-test-design` to write the steps. Then:

```bash
gh issue edit <issue> --body-file <path-with-checklist-appended>
```

It goes on the **issue**, not only the PR, because the issue is the board card
— the person verifying is looking at the card, and checkboxes there survive the
PR being merged and forgotten.

Every step names where to start, one observable outcome, and what "right" looks
like. If `environments.dev.url` is null, say the URL is unknown rather than
writing "open dev" and leaving the reader to guess.

## 7. Now move the card

```bash
.claude/skills/feature-workflow/scripts/board.sh move <issue> inReview
```

Last, deliberately. The card now says something true: it is on `dev` and can be
tried.

## 8. Tell the human what to do

One short message: the PR link, the issue link, and that the checklist is
waiting on the ticket. Do not paste the whole checklist into chat — it lives on
the card so it can be ticked.

## What this skill will not do

- Move the card to Done. That is the human's verification, and it is what
  authorises a release.
- Merge with CI red or failing tests.
- Write manual steps for a UI it has not seen running.
