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

Test all Go modules:

```bash
for dir in go/*/; do
    (cd "$dir" && go test -race -count=1 ./...)
done
```

## Commit Conventions

All commits follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

feat(slog-gcp): add WithProject option
fix(cloudrun): handle missing trace header
docs: update module index in README
```

Valid types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`,
`build`, `ci`, `chore`, `revert`.

## Git Hooks

Activate the versioned hooks after cloning:

```bash
git config core.hooksPath .githooks
```

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

## Issues & Pull Requests

- **Issues**: Welcome. Use the templates provided.
- **Pull requests**: Please open an issue first to discuss the change.

## License

By contributing, you agree that your contributions will be licensed under the
Apache License 2.0 (code) or Creative Commons Attribution 4.0 International
(documentation, skills, and blueprints), matching the license of the directory
you are contributing to. See the [README](README.md#license) for details.
