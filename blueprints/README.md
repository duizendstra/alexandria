# Blueprints

Project scaffolding templates for bootstrapping new repositories.

## Blueprint Index

| Blueprint | Description |
|---|---|
| [service/.ko.yaml](service/.ko.yaml) | Golden ko build config for Go Cloud Run services |
| [githooks/](githooks/) | Golden git hooks for Go repos — conventional commits, staged-content gofmt + secret scan, pre-push quality gate |

## Categories

- **service/** — Go Cloud Run service scaffolding
- **githooks/** — repository git hooks (opt-in via `core.hooksPath`)
