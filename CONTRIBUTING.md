# Contributing to Alexandria

Thank you for your interest in Alexandria. This guide covers the development
workflow for the multi-module repository.

## Prerequisites

- Go 1.26+
- [buf](https://buf.build/) (for proto contracts)
- [golangci-lint](https://golangci-lint.run/)

## Repository Structure

Alexandria is a **multi-module Go repository**. Each directory under `go/`
is an independent Go module with its own `go.mod` and version lifecycle.

## Local Development

Use a `go.work` file (not committed) for cross-module development:

```bash
# Create a local workspace (one-time setup)
cat > go.work << 'EOF'
go 1.26
use (
    ./go/slog-gcp
)
EOF
```

## Running Tests

Test a single module:

```bash
cd go/slog-gcp
go test -race -count=1 ./...
```

Test all Go modules with [`just`](https://just.systems) (provided by the Nix
dev shell):

```bash
just test-all    # go test -race across every module
just lint-all    # golangci-lint across every module
just cover-all   # per-module coverage summary
just check       # vet + lint + test — the full pre-push gate
```

The recipes iterate `find go -name go.mod` — the same discovery the CI
matrix uses — so local runs and CI cannot diverge on module coverage.
Without `just`, run the equivalent loop directly:

```bash
for modfile in $(find go -name go.mod); do
    (cd "$(dirname "$modfile")" && GOWORK=off go test -race -count=1 ./...)
done
```

CI enforces a per-module coverage ratchet
(`.github/coverage-baselines.json`): coverage may not drop below the
recorded baseline. Raise baselines as coverage improves; the long-term
target is the 80% publication bar.

## Commit Conventions

All commits follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

feat(slog-gcp): add WithProject option
fix(cloudrun): handle missing trace header
docs: update module index in README
```

Valid types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`,
`build`, `ci`, `chore`, `revert`. Mark breaking changes with `!` before the
colon (e.g. `feat(async)!: redesign Runner lifecycle`).

## Git Hooks

Activate the versioned hooks after cloning:

```bash
git config core.hooksPath .githooks
```

The hooks are instances of the golden [`blueprints/githooks`](blueprints/githooks/)
set: `commit-msg` validates Conventional Commits (including the `!`
breaking-change marker; git-generated merge/revert messages pass through),
`pre-commit` checks staged content for gofmt cleanliness and leaked
credentials, and `pre-push` runs the fail-closed vet/lint/test/build gate.

`pre-push` scopes that gate to the commits being pushed rather than sweeping
all ~30 modules every time: a documentation-only push runs `okf-lint` and
`scripts/check-links.sh` alone, a change inside one module runs that module,
and a change to shared configuration falls back to the full sweep (every
module *and* the documentation checks — the doc-checking scripts are
themselves shared configuration). It prints the scope it chose before
running anything, and `PRE_PUSH_DRY_RUN=1` prints that plan without running
the gate. `just check` remains the full local gate,
and CI remains authoritative — narrowing the hook cannot let anything through.
(`pre-push` is the one hook that deliberately diverges from the golden
blueprint, since Alexandria has no root `go.mod`; CI exempts it from the
hook-drift check.)

## Adding a New Go Module

1. Create a directory under `go/`:
   ```bash
   mkdir go/my-package
   cd go/my-package
   go mod init github.com/duizendstra/alexandria/go/my-package
   ```

2. Add `doc.go`, `README.md`, and `example_test.go`.

3. Add the module to the root `README.md` module index table.

4. Add a Dependabot entry in `.github/dependabot.yml`.

## Consuming Alexandria Modules in Downstream Projects

External repositories consuming published Alexandria Go modules should follow these
guidelines (see the [Module Adoption Playbook](docs/07-playbooks/module-adoption.md) for full details):

- **No Committed Replace Directives**: Always remove filesystem `replace` directives pointing to
  local Alexandria checkouts (e.g. `replace github.com/duizendstra/alexandria/go/... => ../...`).
  Committed `replace` directives silently mask tagged versions and break reproducible builds.
- **Local Development via Workspace**: Use an uncommitted `go.work` file in your local workspace
  to test unreleased changes across repositories without editing `go.mod`.
- **Dynamic Caller Discovery**: Discover callers using `grep -rl` across your repository rather
  than assuming static file paths.
- **Build Stamping**: If your build scripts stamp a consumer-local build-info package, continue
  stamping that package in `-ldflags`. Alexandria's `buildstamp` module provides parsing and
  verification logic.

## Versioning

Each module is versioned independently with path-prefixed tags:

```bash
git tag go/slog-gcp/v0.1.0
git push origin go/slog-gcp/v0.1.0
```

## Publication Checklist

Before publishing a new Go module:

- [ ] Zero references to any client, company, or internal project
- [ ] No hardcoded credentials, project IDs, or internal URLs
- [ ] Used in production ≥ 1 month
- [ ] ≥ 80% test coverage
- [ ] `doc.go` with package documentation
- [ ] `example_test.go` with at least one Example function
- [ ] Apache-2.0 header in every `.go` file
- [ ] `golangci-lint` clean
- [ ] Per-module `README.md` with install + usage
- [ ] README code examples must compile against the actual API signatures
- [ ] Output examples must match actual handler output
- [ ] No internal company, project, or team names in source or tests

## Content Scan (CI Gate)

This repository is public. Every pull request and every push to `main` runs
an automated content scan (`.github/workflows/content-scan.yml`) in addition
to author discipline and the checklist above:

- **`secret-scan`** runs a standard secret scanner over the diff. This
  applies to every pull request, including ones opened from a fork.
- **`denylist-scan`** matches the diff against a list of patterns that must
  never appear in this repository. The list itself is not published here —
  it is supplied to the workflow at run time as a repository secret, so
  reading this repository does not tell you what it contains.

If `denylist-scan` fails with a "DEGRADED MODE" message, the denylist half
did not run for that change (this is expected for fork pull requests, since
GitHub never exposes repository secrets to fork `pull_request` runs) — only
the secret scan covered it. A maintainer with write access needs to re-run
the denylist half from a context that has the secret (for example, by
pushing the reviewed branch to this repository) before the change can be
treated as fully scanned.

If either check fails on your PR:

1. Read the job log/summary for which check failed and, for the secret
   scanner, which file and line.
2. Remove the offending content and force-push the fix; do not attempt to
   "fix" it with a follow-up commit that leaves the original commit in
   history — squash or amend it out.
3. If you believe a finding is a false positive, say so in the PR and ask a
   maintainer to review; do not modify the workflow to silence it.

## Issues & Pull Requests

- **Issues**: Welcome. Use the templates provided.
- **Pull requests**: Please open an issue first to discuss the change.

## License

By contributing, you agree that your contributions will be licensed under the
Apache License 2.0 (code) or Creative Commons Attribution 4.0 International
(documentation, skills, and blueprints), matching the license of the directory
you are contributing to. See the [README](README.md#license) for details.
