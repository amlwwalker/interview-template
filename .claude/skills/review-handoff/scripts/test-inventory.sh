#!/usr/bin/env bash
#
# test-inventory.sh — list the project's tests as markdown, with permalinks.
#
#   ./test-inventory.sh                 every test in the repo
#   ./test-inventory.sh --since <ref>   only tests in files changed since <ref>
#   ./test-inventory.sh --sha <sha>     build permalinks against this commit
#
# Output is the table that goes on the ticket when it moves to In review.
#
# Descriptions:
#   vitest  the it()/test() string — already a sentence
#   go      the `//` comment line directly above `func TestX`. Go test names
#           are identifiers, not descriptions, so the convention is that each
#           test carries one comment line saying what it proves. Tests missing
#           one are listed at the end so they can be fixed.
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLAUDE_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
REPO_ROOT="$(cd "$CLAUDE_DIR/.." && pwd)"
CONFIG="$CLAUDE_DIR/workflow.config.json"

die() { printf 'error: %s\n' "$1" >&2; exit 1; }

command -v jq >/dev/null 2>&1 || die "jq is not installed."
[[ -f "$CONFIG" ]] || die "no config at $CONFIG"

SINCE=""
SHA=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --since) SINCE="${2:-}"; shift 2 ;;
    --sha)   SHA="${2:-}";   shift 2 ;;
    *) die "unknown argument: $1" ;;
  esac
done

cd "$REPO_ROOT"

# --verify, or an empty repo prints "HEAD" to stdout AND errors, giving a
# two-line SHA that silently corrupts every permalink.
[[ -n "$SHA" ]] || SHA="$(git rev-parse --verify HEAD 2>/dev/null || true)"
[[ -n "$SHA" ]] || SHA="main"

# nameWithOwner from git, so this works on forks without configuration.
SLUG="$(git config --get remote.origin.url 2>/dev/null \
  | sed -E 's#(git@|https://)github\.com[:/]##; s#\.git$##' || true)"
[[ -n "$SLUG" ]] || SLUG="OWNER/REPO"

link() { printf 'https://github.com/%s/blob/%s/%s#L%s' "$SLUG" "$SHA" "$1" "$2"; }

# Files changed since a ref, as a filter set.
changed_filter() {
  if [[ -n "$SINCE" ]]; then
    git diff --name-only "$SINCE"...HEAD
  else
    echo "__ALL__"
  fi
}
CHANGED="$(changed_filter)"

included() {
  [[ "$CHANGED" == "__ALL__" ]] && return 0
  grep -Fxq "$1" <<<"$CHANGED"
}

# find(1) does not expand brace alternatives, so "*.test.{ts,tsx}" matches
# nothing. Expand them into a -name ... -o -name ... group by hand.
find_tests() {
  local glob="$1" dir base
  dir="${glob%%/**}"            # backend/**/... -> backend
  [[ -d "$dir" ]] || dir="."
  base="$(basename "$glob")"     # *.test.{ts,tsx}

  local -a names=()
  if [[ "$base" =~ ^(.*)\{([^}]*)\}(.*)$ ]]; then
    local pre="${BASH_REMATCH[1]}" alts="${BASH_REMATCH[2]}" post="${BASH_REMATCH[3]}"
    local IFS=','
    for alt in $alts; do
      names+=(-name "${pre}${alt}${post}" -o)
    done
    unset IFS
    unset 'names[${#names[@]}-1]'   # drop the trailing -o
  else
    names=(-name "$base")
  fi

  find "$dir" \( -name node_modules -o -name dist -o -name .git \) -prune -o \
       -type f \( "${names[@]}" \) -print0 2>/dev/null
}

missing_desc=()
rows=0

emit_row() { printf '| `%s` | %s | [%s:%s](%s) |\n' "$1" "$2" "$(basename "$3")" "$4" "$(link "$3" "$4")"; }

printf '| Test | What it proves | File |\n|---|---|---|\n'

while IFS=$'\t' read -r lang glob; do
  # shellcheck disable=SC2044
  while IFS= read -r -d '' file; do
    rel="${file#./}"
    included "$rel" || continue

    case "$lang" in
      go)
        # Walk the file keeping the previous line, so the comment above a test
        # function is available when the function is found.
        # Accumulate the contiguous // block preceding each func, and use its
        # FIRST line — a two-line comment read backwards yields a fragment
        # like "rather than leave it as it was", which reads as nonsense.
        block=()
        lineno=0
        while IFS= read -r line; do
          lineno=$((lineno + 1))
          if [[ "$line" =~ ^[[:space:]]*//[[:space:]]?(.*)$ ]]; then
            block+=("${BASH_REMATCH[1]}")
            continue
          fi
          if [[ "$line" =~ ^func[[:space:]]+(Test[A-Za-z0-9_]+) ]]; then
            name="${BASH_REMATCH[1]}"
            if (( ${#block[@]} > 0 )) && [[ -n "${block[0]}" ]]; then
              emit_row "$name" "${block[0]}" "$rel" "$lineno"
              rows=$((rows + 1))
            else
              missing_desc+=("$rel:$lineno $name")
            fi
          fi
          block=()
        done < "$file"
        ;;

      vitest)
        # The it() string IS the description, so the name column carries the
        # enclosing describe() instead — the unit under test. Printing the same
        # sentence in both columns tells a reviewer nothing.
        suite=""
        while IFS=: read -r lineno content; do
          # bash regex is POSIX ERE, which has no backreferences, so matching
          # the closing quote needs sed rather than [[ =~ ]].
          if [[ "$content" =~ ^[[:space:]]*describe\( ]]; then
            suite="$(sed -E 's/^[[:space:]]*describe\((["'"'"'])(.*)\1.*$/\2/' <<<"$content")"
            continue
          fi
          desc="$(sed -E 's/^[[:space:]]*(it|test)\((["'"'"'])(.*)\2.*$/\3/' <<<"$content")"
          [[ -n "$desc" && "$desc" != "$content" ]] || continue
          emit_row "${suite:-$(basename "$rel" .test.tsx)}" "$desc" "$rel" "$lineno"
          rows=$((rows + 1))
        done < <(grep -nE '^[[:space:]]*(describe|it|test)\(' "$file" || true)
        ;;
    esac
  done < <(find_tests "$glob")
done < <(jq -r '.tests | to_entries[] | [.value.language, .value.glob] | @tsv' "$CONFIG")

if (( rows == 0 )); then
  printf '| _no tests found_ | | |\n'
fi

if (( ${#missing_desc[@]} > 0 )); then
  printf '\n> **%d Go test(s) have no description comment.** The convention is a\n' "${#missing_desc[@]}"
  printf '> single `//` line directly above `func TestX` saying what it proves:\n>\n'
  for m in "${missing_desc[@]}"; do
    printf '> - `%s`\n' "$m"
  done
fi
