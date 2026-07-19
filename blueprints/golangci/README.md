# golangci-lint blueprint

Golden lint configurations for Go repositories (golangci-lint v2
schema). One quality bar, two dependency postures:

| Profile | For | Dependencies | Complexity |
|---|---|---|---|
| [`library.golangci.yml`](library.golangci.yml) | Reusable library repos | stdlib + curated external allowlist | relaxed (funlen 120, cyclop 15, lll 160) |
| [`consumer.golangci.yml`](consumer.golangci.yml) | Applications, services, Pulumi stacks, CLIs | stdlib + vetted library modules **only** | tight (funlen 80, cyclop 10, lll 120) |

The posture difference is the architecture: externals are integrated
once, inside a library module, behind the library profile's curated
allowlist — composition roots then consume libraries and never grow
their own dependency surface. If a consumer repo "needs" a new
external, that's a signal the code belongs in a library.

## Install

```bash
cp <alexandria>/blueprints/golangci/consumer.golangci.yml .golangci.yml   # or library.golangci.yml
```

Then edit the sections marked `ADJUST:` — the depguard allowlist (and,
for libraries, the ireturn framework types). Everything else is the
standard; deviations should be deliberate and explainable.

## Principles

- **Strict by default, fail closed**: `max-issues-per-linter: 0`,
  `nolintlint` requires specific, explained suppressions.
- **The depguard allowlist is the dependency policy**: `list-mode:
  strict` means unlisted imports fail. Every addition is a recorded
  decision, not drift.
- **Tests are relaxed, not exempt**: test files skip style/dependency
  strictness but still pass vet, staticcheck, and the core bug linters.
- **No stale rules**: path exclusions must reference paths that exist.
  When a package is deleted, delete its exclusion — a config full of
  dead rules stops being trusted.

## Pairing

Pairs with [`blueprints/githooks/`](../githooks/): the pre-push hook
runs `golangci-lint run ./...` against this config and fails closed if
the tool is missing.
