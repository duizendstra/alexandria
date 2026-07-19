# Git hooks blueprint

Golden git hooks for Go repositories. Copy the three scripts into a
repo's `.githooks/` directory; they are self-contained POSIX sh with
no dependencies beyond git and the Go toolchain.

Battle-tested in a production Pulumi stack (hardened 2026-07-19 after
a review found the originals checked the working tree instead of the
index and rejected git-generated commit messages; hardened again the
same day with lessons from the bloem workspace hooks — submodule
guard, govulncheck, deletion-aware pre-push).

## Hooks

| Hook | Runs on | Enforces |
|---|---|---|
| `commit-msg` | every commit | Conventional Commits (`type(scope)!: description`); git-generated messages (`Merge`, `Revert "…"`, `fixup!`, `squash!`) pass through |
| `pre-commit` | every commit | gofmt on **staged** content (the index, not the working tree) + credential scan (GitHub tokens, GCP/AWS keys, private keys) |
| `pre-push` | every push | quality gate: `go vet` → `golangci-lint` (fails closed if missing) → `go test` → `go build` → `govulncheck` (skipped if not installed), with `GOWORK=off` for CI parity; blocks on dirty submodules; skipped entirely when the push only deletes refs |

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

## Design principle

The pre-push gate should mirror the repo's CI: same checks, same module
resolution (hence `GOWORK=off`). The gate is only trustworthy while it
predicts CI — when CI gains a check that can run locally, add it here
too.

## Adaptation notes

- **Multi-module repos**: `pre-push` assumes a single Go module at the
  repo root. Alexandria's own `.githooks/pre-push` is the worked
  example of the multi-module adaptation — it loops over every `go.mod`
  under `go/`, mirroring the CI detect-modules matrix.
- **Submodule workspaces**: the dirty-submodule check activates only
  when `.gitmodules` exists, so plain repos need no changes. Meta-repos
  that delegate hooks into submodules should instead centralise the
  gate in the superproject (resolve it via
  `git rev-parse --show-superproject-toplevel`).
- **Non-Go repos**: drop the gofmt block from `pre-commit` and replace
  the `pre-push` gate; the secret scan and `commit-msg` are
  language-agnostic.
- **Secret scan limits**: the scanner matches token *prefixes*
  (`ghp_`/`gho_`-family, `github_pat_`, `AIza`, `AKIA`, private key
  headers). It does not catch billing accounts, org IDs, or customer
  names — publication scrubs still need human review.
- The private-key pattern is written as `-{5}BEGIN` so the scanner
  never matches its own source. Keep that property when extending it.
- Filenames containing spaces are handled; filenames containing
  newlines are not.
