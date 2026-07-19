# Git hooks blueprint

Golden git hooks for Go repositories. Copy the three scripts into a
repo's `.githooks/` directory; they are self-contained POSIX sh with
no dependencies beyond git and the Go toolchain.

Battle-tested in a production Pulumi stack (hardened 2026-07-19 after
a review found the originals checked the working tree instead of the
index and rejected git-generated commit messages).

## Hooks

| Hook | Runs on | Enforces |
|---|---|---|
| `commit-msg` | every commit | Conventional Commits (`type(scope)!: description`); git-generated messages (`Merge`, `Revert "…"`, `fixup!`, `squash!`) pass through |
| `pre-commit` | every commit | gofmt on **staged** content (the index, not the working tree) + credential scan (GitHub tokens, GCP/AWS keys, private keys) |
| `pre-push` | every push | quality gate: `go vet` → `golangci-lint` (fails closed if missing) → `go test` → `go build`, with `GOWORK=off` for CI parity |

## Install

```bash
mkdir -p .githooks
cp <alexandria>/blueprints/githooks/* .githooks/
chmod +x .githooks/*
```

Hooks are opt-in per clone — activate them (and document this in the
repo's README, or it will be forgotten):

```bash
git config core.hooksPath .githooks
```

## Adaptation notes

- **Multi-module repos**: `pre-push` assumes a single Go module at the
  repo root. For a multi-module layout, wrap the gate in a loop over
  `go.mod` directories (or gate only the modules touched by the push).
- **Non-Go repos**: drop the gofmt block from `pre-commit` and replace
  the `pre-push` gate; the secret scan and `commit-msg` are
  language-agnostic.
- **Secret scan limits**: the scanner matches token *prefixes*
  (`ghp_`/`gho_`-family, `github_pat_`, `AIza`, `AKIA`, private key
  headers). It does not catch billing accounts, org IDs, or customer
  names — publication scrubs still need human review.
- The private-key pattern is written as `-{5}BEGIN` so the scanner
  never matches its own source. Keep that property when extending it.
