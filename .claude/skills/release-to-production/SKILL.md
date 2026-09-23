---
name: release-to-production
description: Ship verified work to production by opening and merging a PR from the integration branch to the production branch. Use only after a human has moved the ticket to Done. Triggers on "release", "ship it", "deploy to production", "merge dev to main", "push to prod", "cut a release".
---

# Release to production

The card reaching Done is the authorisation. Nothing else is.

## 1. Verify the authorisation is real

```bash
.claude/skills/feature-workflow/scripts/board.sh status <issue>
```

It must report the `done` column. If it says In review, the human has not
finished verifying — **stop and say so**. Do not move it to Done yourself to
unblock the release; that defeats the only gate in the process that a machine
cannot fake.

If the user asks to release something still in review, tell them what is
outstanding and let them decide. They may well say "yes, go" — that is their
call, and it should be made knowingly rather than by a tool quietly assuming.

## 2. See exactly what would ship

```bash
git fetch origin
git log --oneline origin/main..origin/dev
gh pr list --base main --state merged --limit 5   # what went out last time
```

A release often carries more than the ticket you are thinking about. List
**every** issue in the diff, not just the one that prompted this:

```bash
git log origin/main..origin/dev --grep='#[0-9]' -oE '#[0-9]+' | sort -u
```

If something in that list is *not* in Done, say so. Shipping an unverified
change alongside a verified one is the most common way unverified code reaches
production.

## 3. Confirm before opening

Show the human:

- the issues that would ship, and each one's column
- the commit count and rough shape of the diff
- anything in the list that is not Done

Then ask. This is the last reversible moment.

## 4. Open and merge

```bash
gh pr create --base main --head dev \
  --title "Release: <summary>" \
  --body-file <path>

gh pr checks --watch
gh pr merge --merge          # a merge commit, not a squash
```

**Merge, do not squash.** `dev` and `main` must keep a shared history, or every
subsequent `main..dev` diff is wrong and the next release cannot be read.

## 5. Afterwards

Comment on each shipped issue that it is in production, with the PR link.
Leave the cards in Done — Done means verified and shipped; there is no further
column, and inventing one would mean two places to look.

## What this skill will not do

- Move a card to Done.
- Release anything whose card is not in Done, without the human explicitly
  saying so after being shown what is unverified.
- Squash `dev` into `main`.
- Force-push either branch.
