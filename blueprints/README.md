# Blueprints

Project scaffolding templates for bootstrapping new repositories.

## Blueprint Index

| Blueprint | Description |
|---|---|
| [service/.ko.yaml](service/.ko.yaml) | Golden ko build config for Go Cloud Run services |
| [google-addon/](google-addon/) | Scaffolding for Google Calendar Add-ons: Level 1 (Apps Script) and Level 2 (Go Alternate Runtime on Cloud Run) |
| [githooks/](githooks/) | Golden git hooks for Go repos — conventional commits, staged-content gofmt + secret scan, pre-push quality gate |
| [golangci/](golangci/) | Golden golangci-lint profiles — one quality bar, library and consumer dependency postures |
| [workstation/](workstation/) | Workstation bootstrap for the pass + GPG secrets workflow — non-interactive agent unlock, .secrets.yaml → env exports |

## Verification status

Blueprints are teaching and scaffolding material, not published modules. Every Go
job in [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) enumerates
modules with `find go/`, so nothing under `blueprints/` is built, vetted, tested,
linted or gofmt-checked by CI.

That exclusion is deliberate: a blueprint is copied and adapted rather than
imported, and holding example code to the library lint profile costs more in
readability than it returns. It is also load-bearing, so it is written down here
rather than left to be inferred — `blueprints/google-addon/go` was committed
unformatted and with an API key travelling in a request URL, where every
transport error published it to logs and to the caller, and no gate reported
either fault.

Verify a Go blueprint by hand before trusting it:

```bash
cd blueprints/google-addon/go && GOWORK=off go build ./... && GOWORK=off go vet ./... && GOWORK=off go test -race ./...
```

## Categories

- **google-addon/** — Google Workspace Add-on scaffolding (Apps Script and Go Alternate Runtime)
- **service/** — Go Cloud Run service scaffolding
- **githooks/** — repository git hooks (opt-in via `core.hooksPath`)
- **golangci/** — lint configuration profiles
- **workstation/** — developer workstation secrets/GPG bootstrap
