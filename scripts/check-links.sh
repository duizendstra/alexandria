#!/bin/sh
# Broken internal link check for the markdown corpus.
#
# Single source of truth for CI (.github/workflows/ci.yml "Validate docs")
# and the pre-push hook. Portable POSIX sh: no `grep -oP` and no process
# substitution, so it runs on macOS/BSD as well as on GNU/Linux runners.
#
# Usage: scripts/check-links.sh [--github]
#   --github  emit GitHub Actions annotations instead of plain lines
#
# Exit code 0 = clean, 1 = at least one broken link.
set -eu

github=0
case "${1:-}" in
  --github) github=1 ;;
  "") ;;
  *)
    echo "check-links.sh: unknown argument '$1' (expected --github or none)" >&2
    exit 2
    ;;
esac

cd "$(git rev-parse --show-toplevel)"

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT INT TERM

status=0
findings=0

# Markdown inline links: [text](target). The alternation allows one level
# of nested brackets in the link text, so `[see [note] here](t.md)` and
# badge links `[![img](i.png)](t.md)` resolve their real target rather
# than being skipped or reporting the inner image URL.
#
# Fenced code blocks are stripped first: a link inside a fence is sample
# markdown being shown to the reader, not a link the document makes, and
# its target is a placeholder that resolves to nothing by design.
find . -name '*.md' -not -path './.git/*' | sort > "$TMP/files"
while IFS= read -r file; do
  awk '/^[[:space:]]*(```|~~~)/ { fence = !fence; next } !fence' "$file" \
    | grep -oE '\[([^][]|\[[^]]*\])*\]\([^)]*\)' 2>/dev/null \
    | sed -E 's/^\[([^][]|\[[^]]*\])*\]\(//; s/\)$//' > "$TMP/targets" || true
  while IFS= read -r link; do
    # Skip URLs, mail, pure anchors, and \<template placeholders\>.
    if printf '%s\n' "$link" | grep -qE '^(https?://|mailto:|#|\\<)'; then
      continue
    fi
    # Strip any #anchor — only the file part is resolvable.
    target=$(printf '%s\n' "$link" | sed 's/#.*//')
    if [ -z "$target" ]; then
      continue
    fi
    dir=$(dirname "$file")
    resolved="$dir/$target"
    if [ ! -e "$resolved" ]; then
      if [ "$github" -eq 1 ]; then
        echo "::warning file=$file::Broken link: $link (resolved: $resolved)"
      else
        echo "  ✗ $file: broken link $link (resolved: $resolved)"
      fi
      findings=$((findings + 1))
      status=1
    fi
  done < "$TMP/targets"
done < "$TMP/files"

if [ "$status" -eq 0 ]; then
  echo "✅ No broken internal links."
else
  echo "❌ $findings broken internal link(s)."
fi

exit $status
