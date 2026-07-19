# Alexandria Agent Rules

## 1. Parent Reference Updates
When creating, renaming, or deleting a file, always update every parent
document that indexes or links to it. Common parents:
- `index.md` in the same directory (OKF directory listing)
- `README.md` in the same or parent directory
- Root `README.md` module/directory tables
- `CHANGELOG.md` for user-facing changes
- `maturity` frontmatter (e.g. `seed` → `standard` when content is added)

## 2. Verify Before Delivering
Include a **Verification Plan** section in every implementation plan. For
release-facing work, the verification plan MUST include running the
`release-review` skill (6-agent parallel review) before committing.

## 3. Read Specs Before Assuming
When the project references an external standard (OKF, MADR, Conventional
Commits, etc.), read the upstream spec before making changes that touch
that standard. Do not assume conventions — verify them.

## 4. Terminology Consistency
This project uses the following canonical terms:
- **Go module** (not "Go package" when referring to a `go.mod`-rooted unit)
- **monorepo** (not "mono-repo")
- **multi-module** (hyphenated)
- **knowledge bundle** (OKF term for a directory of markdown files)

## 5. Batch File Operations
When creating or modifying 3+ files, prefer delegating to subagents. Group
related changes into a single subagent (e.g., all new files in one, all
modifications in another).

## 6. OKF Conventions
- `docs/` subdirectories use `index.md` (OKF reserved filename)
- `docs/` root uses `README.md` (GitHub rendering convention)
- All other top-level directories use `README.md`
- Every `docs/` markdown file must have the full Alexandria frontmatter schema
  (see `docs/08-reference/okf-profile.md`)

## 7. CI Link Checker Awareness
The CI docs job scans all `.md` files for markdown links and verifies targets
exist. Avoid writing example markdown link syntax (text in brackets, URL in
parens) in documentation — the grep pattern will match it as a real link. Use
prose descriptions or backtick-escaped syntax instead.

## 8. Go Module Tagging
- Use path-prefixed annotated tags: `git tag -a go/<module>/v<semver> -m "<message>"`
- The tag path prefix must match the module's subdirectory relative to repo root.
- Libraries should pin `go.mod` to minor version only (e.g., `go 1.26`), not patch.
- Use `v0.0.x` for modules with foreseeable API changes; `v0.1.x` when API stabilizes.

## 9. CI Tool Compatibility
When updating Go versions, also verify `golangci-lint-action` version compatibility.
The action major version must support the Go version used by the module.
Committed `replace` directives are forbidden in `go/**/go.mod` (enforced by the
`mod-hygiene` CI job and the `gomoddirectives` linter). Inter-module requires
must pin published tags; cross-module local development uses an uncommitted
`go.work` file (see CONTRIBUTING.md). After tagging a module, bump any sibling
modules that pin it.

## 10. PR Merge Strategy
Merge PRs with `gh pr merge --squash --delete-branch` to keep main history
clean and auto-delete feature branches after merge.
Never commit directly to main — all changes must go through a PR,
including chore commits (rules, docs, config).

## 11. Go Context & HTTP Testing Linting
When writing tests that execute HTTP requests or verify contexts:
- Always use `httptest.NewRequestWithContext(context.Background(), ...)` instead of `httptest.NewRequest` to satisfy the `noctx` linter.
- Avoid capturing parent variables inside inline handler closures to verify context propagation, as it triggers `fatcontext`. Instead, define a small helper handler struct (e.g., `captureHandler`) that implements `http.Handler` and stores context values in struct fields.

## 12. Skill Location
Skills live at `skills/` (repo root), not `.agents/skills/`. Other repos
inherit skills via `skills.json` in their `.agents/` directory.

## 13. Public Repo Hygiene
Never reference private GitHub orgs (e.g., `duizendstra-com`) in public
repositories. Use generic placeholders (`OWNER/REPO`) in templates and
documentation.

## 14. Shell Portability in Documentation
Shell commands in public documentation must work on both macOS and Linux.
Known pitfalls: `sed -i ''` (macOS-only, use `sed -i.bak`),
`readlink -f` (GNU-only), `grep -P` (GNU-only).

## 15. Isolated Wildcard Compilation & Testing
In multi-module monorepos, nested subdirectories (such as `go/slog-gcp/otelgcp` nested inside `go/slog-gcp`) cause wildcard expansions (`./...`) to error in workspace mode due to directory prefix overlapping. To perform clean, isolated compilation, testing, or linting of any module, always prepend `GOWORK=off` to the execution command (e.g., `GOWORK=off go test ./...`). This disables workspace resolution and guarantees that the module builds self-sufficiently against its pinned dependencies.
