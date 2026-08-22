# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

- **go/dataquality/datadiff**: The stats tolerance filter now reports a `StatDiff` whose `Left`, `Right` or `Delta` is NaN or ±Inf instead of silently dropping it as "within tolerance" — including NaN-vs-NaN and Inf-vs-Inf, since a non-finite aggregate is never floating-point noise (#247).
- **go/platform/web**: `WriteError` no longer panics on an `apierr.StatusError` whose status is outside `net/http`'s 100–999 range, and no longer reflects the upstream response excerpt (`StatusError.Body`) into the client-facing `{"error": …}` payload. A `StatusError` sets the response status only for 4xx/5xx codes (the client sees `"<sentinel>: <status>"`); 0, 1xx, 2xx, 3xx and out-of-range codes fall back to the generic 500 response (#250).
- **go/platform/procrun** (v0.3.1): `Runner.LookPath` resolves commands on Windows via PATHEXT; the Unix execute-bit check moved to a build-tagged file (#251).
- **go/platform/runstate** (v0.4.0): `Locker` now implements coordination v0.2.0's `Excluder` — `TryAcquire(coordination.Subject)` validates the subject and refuses a second run with the coordination sentinel. `ErrLocked` and `ErrBadSubject` are now aliases of `coordination.ErrLocked` and `coordination.ErrBadSubject` (`errors.Is` matches under either name; the `ErrLocked` message text changed to the coordination wording). The interrupt re-raise moved behind build tags: Unix re-delivers the signal as before, Windows — which has no `syscall.Kill` — exits with code 1 after the lock file is removed, so the module vets on Windows again.
- **go/observability/audit**: Withdrew the unresolvable `v2.0.0` tag — Go rejects a v2+ version whose module path has no `/v2` suffix, and the major bump was a release-please rescan artifact, not an API break. The `Reader`/`Filter`/`AggregateScorecard` release is **v1.1.0**; the manifest and module index now say so.
- **go/platform/coordination** (**breaking**, v0.2.0): The module now publishes the whole coordination language — `Subject` (path-safe by `Validate`), `Holder`/`Self` (the readable record of who is inside), `Waiter.Acquire(ctx, subject)` returning a release and a per-subject fence for callers that queue, `Excluder.TryAcquire(subject)` for callers that refuse, and the sentinels `ErrBadSubject`, `ErrLocked`, `ErrStaleLock`. `Excluder.Acquire(string)` became `TryAcquire(Subject)` and `NopExcluder` was removed (nothing consumed it). `filelock.Locker` is replaced by `filelock.Store`: a readable JSON holder record published by hard link so it is never partially visible, a per-subject fsynced fence, opt-in reclaim by age (`Options.StaleAfter`, no default) with identity-checked release and set-aside so a release never removes a successor's record (identity is the dev/inode pair and the record's bytes, because ext4 reuses a freed inode number), jittered polling, and no signal handling or build tags — standard library only, the same code on Linux, macOS and Windows. `go/platform/runstate` keeps pinning coordination v0.1.0 until its own release adopts the new contract.
- **Documentation & Playbooks**: Added [`docs/07-playbooks/adding-a-module.md`](docs/07-playbooks/adding-a-module.md) — the "Adding a New Module" how-to the playbooks index advertised but never shipped (scaffold, zero-rot authoring, CI wiring triplet, local verification, release) — and refreshed the [OKF profile](docs/08-reference/okf-profile.md) to the upstream v0.2 specification at its new canonical home, `GoogleCloudPlatform/open-knowledge-format`.
- **go/platform/workflow**: Added comprehensive documentation, runnable example tests, SRE hardening guidance, and minor Go version alignment.
- **go/dataquality/datadiff**: Added in-memory set and map parity comparison engine (`SetDiff[T]`, `MapDiff[K, V]`) and 3-way migration reconciler (`ThreeWayDiff[T]`) proving the mathematical equation `baseline ∖ leftovers == target`.
- **go/observability/audit**: Added streaming JSONL `Reader`, predicate `Filter`, and summary `AggregateScorecard` calculation for audit trail event processing.
- **go/governance/gate**: Policy evaluation gate engine (`Gate`, `Policy`, `Rule`, `Verdict`, `Status`, `Enforce`) supporting Standard, Strict, and Permissive evaluation, invariant check adapter bridges, and JSON report serialization.
- **go/google/workspace/drive**: Added Shared Drive lifecycle management (`CreateSharedDrive`, `FindSharedDriveByName`, `GetSharedDrive`, `ListSharedDrives`), folder operations (`CreateFolder`, `FindFolder`, `FindFolderByProperty`), safe file relocation with automatic parent resolution (`MoveFile`, `TrashFile`), and idempotent drive membership reconciliation (`EnsureDriveMembership`, `RoleRank`, `ListDriveMembers`, `ListFilePermissions`, `DeleteDriveMember`, `DeleteFilePermission`).
- **go/google/auth**: Added `ValidateAccessAs(ctx, expectedEmail)` on `*DWDValidator` to verify effective impersonated identity via `About.Get` before root access validation, returning `ErrSubjectMismatch` on identity discrepancy.
- **go/slog-gcp**: Supported Cloud Logging deduplication using native `logging.googleapis.com/insertId` via `WithInsertID`, `WithInsertIDKey`, and `WithCustomInsertIDKey` options.
- **go/observability/audit**: Added bidirectional Protocol Buffer contract conversions (`(Entry).ToProto()`, `EntryFromProto()`, `(Scorecard).ToProto()`, `ScorecardFromProto()`) consuming `go/contracts/observability/audit/v1alpha1`.
- **go/governance/invariant**: Domain invariant rule evaluation engine with typed verdicts (pass, anomaly, fail), ordered execution, and structured notes.
- **go/platform/buildstamp**: Build provenance stamping and strict pre-run verification (commit revision, dirty worktree detection, dependency revisions), with native fuzzing (`FuzzParseStamp`, `FuzzMatches`).
- **go/platform/procrun**: Controlled external command runner isolating PATH, scrubbing parent environment variables, and capturing output safely to logs, with native environment scrubbing fuzzing (`FuzzEnvironScrubbing`).
- **go/platform/runstate**: Repeatable on-disk run state primitives with exclusive per-subject filesystem locking and content/revision leases, with native path traversal and lease fuzzing (`FuzzCheckSubject`, `FuzzLeaseValid`, `FuzzLeaseJSON`).
- **Native Fuzz Testing & Modernization**: Added native Go fuzz testing suites across core platform and governance primitives (`fuzz-all` task runner recipe), `just modernize` (`go fix`) automation, and CI module hygiene modernization enforcement.
- **Documentation & Architecture**: Added [`docs/03-architecture/go-agentic-engineering.md`](docs/03-architecture/go-agentic-engineering.md) on verification-first platform engineering and Go guardrails, Rule 16 on standard library primacy and AI self-correction loops in `.agents/AGENTS.md`, and [`docs/07-playbooks/module-adoption.md`](docs/07-playbooks/module-adoption.md) on downstream adoption.
- **Release Automation**: Adopted Google's `release-please` in manifest mode with `.release-please-manifest.json`, `.release-please-config.json`, `.github/workflows/release.yml`, and module-hygiene version parity enforcement across all 29 Go modules.
- **Documentation & Reference**: Added five engineering-lessons reference pages to `docs/08-reference/` — [`retry-wrapping-shared-adapters.md`](docs/08-reference/retry-wrapping-shared-adapters.md), [`cross-process-locks-shared-remote-resources.md`](docs/08-reference/cross-process-locks-shared-remote-resources.md), [`concurrency-safe-lazy-initialization.md`](docs/08-reference/concurrency-safe-lazy-initialization.md), [`macos-shell-pitfalls-for-agents.md`](docs/08-reference/macos-shell-pitfalls-for-agents.md), and [`go-proxy-tag-lag-and-workspace-drift.md`](docs/08-reference/go-proxy-tag-lag-and-workspace-drift.md) — each a generalized symptom/cause/rule/how-to-test writeup, linked from the reference index.

### Changed

- **Modernization**: Upgraded Go error assertion idioms to `errors.AsType` across `go/platform`, `go/retry`, and consumer modules.
- **Protobuf**: Regenerated all compiled Go contracts and pinned `buf.gen.yaml` plugins to `buf.build/protocolbuffers/go:v1.36.12` and `buf.build/connectrpc/go:v1.18.1`.
- **CI Tooling**: Pinned `golangci-lint` to `v2.13.1` in CI matrix, added `govulncheck` vulnerability scanning step across Go modules, added CI module hygiene enforcement requiring coverage baselines for every Go module, and calibrated coverage floors across all 28 library and blueprint modules.
- **Housekeeping**: Added Dependabot update grouping across all module ecosystems to consolidate weekly PR volume, added Review-Skill Selection Guide in `skills/README.md`, and aligned contracts overview documentation.
- **Dependencies**: Upgraded Pulumi SDK to `3.258.0`, Google Auth to `0.23.1`, Testify to `1.12.1`, and Pulumi GCP provider to `9.34.1` across all `go/iac` modules.

## 2026-07-21 — observability config-driven uptime checks

Released tags: `go/iac/observability/v0.2.0`.

### Added

- **go/iac/observability v0.2.0**: the blueprint now provisions monitoring. When
  `alertEmail` is set it creates an ops email notification channel, and for
  each entry in the optional `uptimeTargets` JSON config it resolves a stack
  reference, derives the host from that stack's URL output, and creates an
  HTTPS uptime check with a failure alert routed to the channel (via
  `gcpinfra/uptimechecks`). Bumps the `gcpinfra` dependency to v0.7.0.

## 2026-07-21 — gcpinfra uptimechecks host as runtime input

Released tags: `go/iac/pulumi/gcpinfra/v0.7.0`.

### Changed

- **go/iac/pulumi/gcpinfra v0.7.0** (breaking): `uptimechecks.Apply` now takes
  the probed `host` as a runtime `pulumi.StringInput` parameter instead of a
  static `Config.Host` string, so the check can target a dynamic stack-ref
  output such as a Cloud Run URI. `Config.Host` and `ErrHostRequired` are
  removed. New signature: `Apply(ctx, projectID, cfg, host, channelIDs, deps)`.

## 2026-07-21 — gcpinfra uptimechecks building block

Released tags: `go/iac/pulumi/gcpinfra/v0.6.0`.

### Added

- **go/iac/pulumi/gcpinfra v0.6.0**: `uptimechecks.Apply` provisions an HTTPS
  `UptimeCheckConfig` on a caller-supplied host/path plus a failure
  `AlertPolicy` tied to the check. IAP-aware acceptance —
  `AcceptedStatusClasses` defaults to `[Class2xx]`; callers add `Class3xx`
  for endpoints whose unauthenticated probes are redirected to the IAP
  sign-in page. Notification channels are caller-supplied IDs (a
  `pulumi.StringArrayInput` to `Apply`) rather than created here, reusing
  the channels `budgets` already provisions instead of duplicating the
  primitive.

## 2026-07-19 — cloudrun multi-container (sidecar) services

Released tags: `go/iac/pulumi/gcpinfra/v0.5.0`.

### Added

- **go/iac/pulumi/gcpinfra v0.5.0**: `cloudrun.ApplySidecarService`
  provisions a multi-container (sidecar) Cloud Run v2 service — one
  ingress container plus localhost sidecars — with optional
  Identity-Aware Proxy (`IAPEnabled`), a scaling cap (`MaxInstances`),
  per-container env/limits/start-order (`DependsOn`), and container
  image changes ignored for CI/CD deploys. A nil service account runs
  the service as the project's default compute SA.

## 2026-07-19 — cloudrun explicit CPU limit

Released tags: `go/iac/pulumi/gcpinfra/v0.4.2`.

> **Note:** `go/iac/pulumi/gcpinfra/v0.4.1` was mis-cut before this
> change merged (it is content-identical to v0.4.0 plus unrelated
> githooks work) and had already been cached by the Go module proxy, so
> it stays published but must not be pinned — use v0.4.2.

### Added

- **go/iac/pulumi/gcpinfra v0.4.1**: `cloudrun.ServiceConfig` and
  `JobConfig` gain an optional `CPU` limit (e.g. `"1000m"`). Cloud Run
  applies a server-side CPU default when unset, which surfaces in
  `pulumi preview` as a phantom limit removal on stacks that never
  declared it; declaring the limit keeps the desired state aligned with
  the live resource.

## 2026-07-19 — ingestion/transform IaC primitives

Released tags: `go/iac/pulumi/gcpinfra/v0.4.0`,
`go/iac/pulumi/stackref/v0.1.0`, `go/platform/passstore/v0.1.0`.

### Added

- **go/iac/pulumi/gcpinfra v0.4.0**: five new building-block packages —
  `cloudrun` (v2 services and jobs with image changes ignored for CI/CD
  deploys, invoker grants), `scheduler` (HTTP-target jobs with OAuth
  authentication), `firestore` (databases and seeded documents with field
  changes ignored after creation), `tables` (native BigQuery tables with
  optional DAY partitioning, and external tables such as Google Sheets),
  and `dataform` (repositories with Git remotes, release/workflow
  configs, P4SA enablement). All follow the established adapter shape:
  sentinel validation errors, `Apply*` entry points, config-validation
  tests.
- **go/iac/pulumi/stackref v0.1.0**: typed readers for Pulumi stack
  reference outputs (`RequireString`), for composition roots that chain
  stacks together.
- **go/platform/passstore v0.1.0**: deploy-time secret retrieval from
  the local pass store (`Show` / `MustShow`) for operator-workstation
  tools such as Pulumi programs.

## 2026-07-19 — delivery secret-accessor grant

Released tags: `go/iac/delivery/v0.1.1`.

### Added

- **go/iac/delivery v0.1.1**: the Compute default SA is granted `secretmanager.secretAccessor` on the GitHub OAuth token secret once the connection is configured — Cloud Build v2 triggers run as that SA and read the authorizer credential.

## 2026-07-19 — maturity & graduation wave

Released tags: `go/platform/apierr/v0.1.0`, `go/retry/v0.1.0`,
`go/retry/gcp/v0.1.0`, `go/platform/gcpenv/v0.1.0`,
`go/discovery/search/v0.1.0`, `go/governance/v0.2.0`,
`go/iac/pulumi/gcpinfra/v0.3.1`, `go/iac/governance/v0.1.1`.
Graduations per ADR-0001: each v0.1.0 module has its API validated by at
least one real consumer.

### Changed

- **go/governance v0.2.0** (breaking): `plan.NewStarter` and
  `plan.NewStandard` now take an `orgID` parameter, mirroring
  `NewEnterprise`. Previously they could never satisfy `validateScope` at
  Organization scope (no way to supply the required OrgID), so starter and
  standard tiers were undeployable at org level. `go/iac/governance` derives
  the org ID from the GCP parent for org-scope plans.

### Added

- **go/iac/delivery**: Configuration-driven Pulumi Delivery blueprint — dedicated CI/CD project with an Artifact Registry (build-SA writer grant), a Cloud Build v2 Git connection with per-repo tag-push triggers, and cross-project registry reader grants for consumer workload stacks; placement resolved from a governance stack reference.
- **scripts/okf-lint.py**: the OKF vault integrity lint ADR-0002 promised —
  validates the full frontmatter schema (required fields, enums, ISO 8601
  timestamps, domain↔folder agreement), rejects malformed/duplicate UUIDs,
  and blocks relations with dangling `target_uuid`s. Wired into the CI
  `docs` job alongside the link checker. Self-contained python3, no
  third-party packages.
- **docs**: canonical frontmatter schema unified — `okf-profile.md` now
  documents the full ADR-0002 schema (`uuid`, `created_at`, `updated_at`,
  `tags`, `relations` were previously undocumented there); the 11 documents
  missing those fields (all indexes, `adr-0001`, `okf-profile.md`,
  `docs/README.md`) backfilled; the divergent relations syntax in
  `writing-enterprise-go-packages.md` converted to the canonical
  `target_uuid`/`rel_type` form. `05-security/index.md` now links the root
  `SECURITY.md` and marks unwritten policies as planned instead of
  advertising them.
- **go/platform/apierr**: `RetryableStatus(int)` and
  `RetryableGRPCCode(uint32)` — the ecosystem's single source of truth for
  transient-failure classification (HTTP 408/429/5xx; gRPC
  DEADLINE_EXCEEDED/RESOURCE_EXHAUSTED/ABORTED/INTERNAL/UNAVAILABLE).
  `FromGRPCCode` now maps ABORTED → `ErrConflict` (the gRPC analogue of
  HTTP 409) instead of `ErrUnexpectedStatus`.
- **godoc examples**: verified `Example*` functions for
  `dataquality/datadiff` (9 examples — previously 0 for 84 exported
  symbols), `platform/async`, `platform/cache`, `platform/gcpenv`,
  `platform/web`, and `slog-gcp/otelgcp` (full trace-bridge wiring).
- **testing**: closed the zero-test gaps — `observability/audit/file`
  (rotation, rename-failure self-healing, concurrent logging under `-race`,
  scorecard parsing), the full `iac/*` tree (`folders.ParseScope`/`OrgID`
  tables + fuzz, adapter validation, tier deployments via Pulumi mocks), and
  fuzz targets for `privacyfilter` redaction and `datadiff` target parsing.
- **justfile**: `test-all` / `vet-all` / `lint-all` / `cover-all` / `check`
  recipes iterating every `go/**/go.mod` exactly like the CI matrix; `just`
  added to the Nix dev shell.
- **CI**: per-module coverage ratchet backed by
  `.github/coverage-baselines.json` — coverage below the recorded baseline
  fails the build; per-module percentages land in the job summary
  (`go/contracts` exempt as generated code).
- **go/retry/gcp**: retryable-error classification now
  delegates to `apierr.RetryableStatus`/`RetryableGRPCCode` instead of
  maintaining a second copy of the transient tables (which had already
  drifted: apierr lacked ABORTED). Behavior delta: HTTP 408 responses from
  Google APIs and OAuth token endpoints are now retried. GCP-specific
  extensions (403 quota reasons, RFC 6749 OAuth codes) remain local.
  Requires `apierr v0.1.0` — tag apierr before retry/gcp.
- **go/slog-gcp**: duplicated managed-platform detection extracted into a
  single `runningOnGCP()` helper (no behavior change).
- **dependencies**: aligned across modules — `grpc v1.82.1`,
  `otel/trace v1.44.0`, `genproto/rpc 20260706` in `go/google` and
  `go/iac/pulumi/gcpinfra` (`go/iac/governance` picked the aligned set
  up via `go mod tidy` once the tags landed).

### Fixed

- **go/observability/audit/file**: `ReadScorecard` no longer hangs on a
  malformed log line. The `json.Decoder` stream loop could not resync after
  a syntax error, so a single torn write (crash mid-append) spun the reader
  forever; it now reads per-line and skips malformed lines as documented.
- **go/iac/pulumi/gcpinfra**: `folders.Apply` validated
  tier policy it does not own — `hierarchy.Config.Validate()` requires ≥1
  child, while `plan.validateStarter` forbids children, so every starter
  deployment failed. The adapter now checks well-formedness only (parent,
  root name, child uniqueness).
- **go/iac/governance**: starter and standard tiers now
  deploy at Organization scope (and starter at folder scope); the two
  known-limitation pinning tests flipped to assert success. Pins
  resolve against the published `governance v0.2.0` and `gcpinfra v0.3.1`.

- **.githooks**: replaced the live hooks with the golden `blueprints/githooks`
  set — `commit-msg` now accepts the `!` breaking-change marker (the previous
  regex rejected the repo's own `feat(async)!:`-style commits) and passes
  git-generated merge/revert/fixup messages; `pre-commit` gains the
  index-based gofmt gate and fine-grained-PAT secret patterns; new
  `pre-push` runs the fail-closed vet/lint/test/build gate across every
  module (multi-module adaptation of the blueprint).
- **.golangci.yml**: pruned 17 dead `depguard` allowlist entries for
  externals no module imports (whatsmeow, libsignal, sqlite, cobra/viper,
  qrterminal, uuid, go-github, …); the allowlist now matches the blueprint
  starter set plus `connectrpc.com/connect` (required by generated
  `go/contracts` code). No lint behavior change for existing code.
- **CONTRIBUTING.md**: the "test all modules" loop now iterates `go.mod`
  files (`find go -name go.mod`) — the previous `go/*/` glob missed all
  nested modules and reached only 5 of 19.
- **README.md**: module index synced with reality — added the missing
  `platform/gcpenv` row, corrected 8 stale version cells (incl.
  `platform/async` → v0.1.0), and fixed the blueprint description to match
  what exists on disk (ko service builds, githooks, golangci profiles).
- **CHANGELOG.md**: restructured — tagged work moved out of `[Unreleased]`
  into dated release-wave sections matching the actual path-prefixed tags.
- **contracts/README.md**: package list synced (34 packages; added the five
  missing: `alx/email`, `alx/postmark`, `common/privacy`,
  `deployment/status`, `timeline/event`) and the `domain/* = v1` vs
  `v1alpha1` versioning convention documented.
- **CI**: `mod-hygiene` now enforces README module-index parity for every
  module and rejects drift between the live `.githooks` and
  `blueprints/githooks` (pre-push exempt as a documented multi-module
  adaptation).

## 2026-07-19 — reliability & governance wave

Released tags: `go/retry/v0.0.4`, `go/retry/gcp/v0.0.4`, `go/slog-gcp/v0.0.3`,
`go/google/v0.0.3`, `go/dataquality/datadiff/v0.0.3`–`v0.0.4`,
`go/observability/audit/v0.0.3`, `go/contracts/v0.0.3`–`v0.0.4`,
`go/platform/apierr/v0.0.3`, `go/platform/async/v0.1.0`,
`go/platform/web/v0.0.2`, `go/platform/gcpenv/v0.0.1`, `go/governance/v0.1.0`,
`go/iac/pulumi/gcpinfra/v0.1.0`–`v0.3.0`, `go/iac/governance/v0.1.0`,
`go/iac/identity/v0.1.0`, `go/iac/workloads/v0.1.0`, `go/iac/finops/v0.1.0`,
`go/iac/observability/v0.1.0`.

### Added

- **go/iac/observability v0.1.0**: configuration-driven Pulumi observability
  blueprint — dedicated project with a BigQuery log-analytics dataset and an
  org-level audit-log sink routed into it (writer identity exported for
  downstream grants), placement resolved from a governance stack reference.
- **go/iac/finops v0.1.0**: configuration-driven Pulumi FinOps blueprint —
  dedicated project with a BigQuery billing-export dataset and an org-scoped
  budget (threshold alerts, email notification channels), placement resolved
  from a governance stack reference.
- **go/iac/identity v0.1.0**: configuration-driven Pulumi identity blueprint —
  dedicated project with Secret Manager secrets (pluggable `SecretResolver`,
  default `pass`), service accounts, consumer/impersonator IAM, placement
  resolved from a governance stack reference.
- **go/iac/workloads v0.1.0**: configuration-driven Pulumi workloads
  blueprint — one or more projects per environment, each serving one or more
  concerns with per-concern exports, placement from a governance stack
  reference, optional deploy-access grant for a delivery trigger SA.
- **go/iac/pulumi/gcpinfra v0.2.0–v0.3.0**: ten new building blocks — GCP
  projects with API enablement (`projects`), Secret Manager secrets
  (`secrets`), service accounts (`serviceaccounts`), IAM member bindings
  (`iambindings`), billing budgets (`budgets`), BigQuery datasets
  (`datasets`), org-level log sinks (`logsinks`), Cloud Build v2 Git
  connections (`connections`), Artifact Registry repositories (`registries`),
  and tag-push Cloud Build triggers (`triggers`).

- **go/platform/gcpenv v0.0.1**: canonical GCP project ID resolver (env vars,
  then metadata service with `GCP_METADATA_DISABLED` bypass); `slog-gcp` and
  `datadiff` adopt it (#50).
- **go/governance v0.1.0**: cloud-agnostic governance domain model — tiered
  plans (Starter/Standard/Enterprise), organizational hierarchy, classification
  dimensions, scope capabilities, and stack export contract. Pure Go, zero
  dependencies (#33).
- **go/iac/pulumi/gcpinfra v0.1.0**: Pulumi building blocks for GCP — folder
  hierarchies (`folders`) and org-level tag keys (`tagkeys`), both
  deletion-protected, consuming validated `go/governance` domain input (#34).
- **go/iac/governance v0.1.0**: configuration-driven Pulumi governance
  blueprint — reads stack config, builds a validated tiered plan, deploys via
  `gcpinfra`, and exports the downstream contract (#35).
- **blueprints/githooks**: golden git hooks for Go repos — Conventional Commits
  validation (git-generated messages pass through), index-based gofmt +
  credential scan on commit, and a fail-closed vet/lint/test/build gate on push
  (#41).
- **blueprints/golangci**: golden golangci-lint profiles — one quality bar in
  two dependency postures: `library` (curated external allowlist, relaxed
  complexity) and `consumer` (stdlib + library modules only, tight complexity)
  (#47).
- **flake.nix**: Nix dev shell pinning `go`, `gotools`, `golangci-lint`,
  `buf`, and `jq` so the documented onboarding works (#39).

### Changed

- **go/platform/async v0.1.0** (breaking): context-aware `Runner` with bounded
  goroutines and TTL janitor (#45); `BatchBuffer` surfaces failed batches
  instead of dropping them (#40).
- **go/retry v0.0.4** (breaking): exhausted `Transport` retries now return an
  error; `Retry-After` honored (#42).
- **go/retry/gcp v0.0.4**: OAuth errors classified structurally before string
  matching (#46).
- **go/observability/audit v0.0.3** (breaking): `Entry.Time` is `time.Time`
  with stable RFC3339 wire format (#49).
- **go/contracts v0.0.4** (breaking): unified conventions; unproven packages
  demoted to `v1alpha1` (#43). Prose "reserved" range comments converted to
  real protobuf `reserved` statements (18 ranges) so tag reuse is rejected at
  the wire level; `contracts` CI job runs `buf lint`, `buf breaking` against
  main, and a generated-code drift check (#38).
- **go/slog-gcp v0.0.3**: `WithProjectID`, `WithEventIDEnabled`,
  `WithTraceResolver`, `WithLabels` setup options; `WithTraceContext` helper
  for async workers; pre-computed trace prefix parsing to reduce hot-path
  allocations.
- **go/google v0.0.3**: uniform retry via transport, single Drive constructor,
  honest `ValidateAccess` (#48); Workspace Drive scanner `WithDriveID` option
  (#31, #32).
- **module hygiene**: standardized `go 1.26` across modules; removed all
  `replace` directives — inter-module requires now pin published tags
  (`platform/web` previously required a non-existent `apierr v0.0.0` and was
  unresolvable for external consumers) (#36, #37); `mod-hygiene` CI job rejects
  committed `replace` directives, `v0.0.0` pins, and modules missing Dependabot
  coverage; Dependabot expanded to all modules + actions; Google, ConnectRPC,
  and Protobuf dependencies upgraded.
- **.golangci.yml**: removed dead rules; config restated as an instance of
  `blueprints/golangci/library.golangci.yml` (#47).
- **documentation**: docs claims aligned with reality — unbuilt machinery
  marked as planned (#44); `contracts/README.md` rewritten around the actual
  Protocol Buffer schemas and Buf workflow; unit tests added for
  `go/slog-gcp/otelgcp` span context extraction.

## 2026-07-12 — initial module harvest

Released tags: `go/retry/v0.0.1`–`v0.0.3`, `go/retry/gcp/v0.0.1`–`v0.0.3`,
`go/google/v0.0.2`, `go/contracts/v0.0.1`–`v0.0.2`,
`go/dataquality/datadiff/v0.0.1`–`v0.0.2`,
`go/observability/audit/v0.0.1`–`v0.0.2`,
`go/discovery/privacyfilter/v0.0.1`–`v0.0.2`,
`go/discovery/search/v0.0.1`–`v0.0.2`, `go/discovery/search/searchtest/v0.0.2`,
`go/platform/apierr/v0.0.1`–`v0.0.2`, `go/platform/async/v0.0.1`–`v0.0.3`,
`go/platform/cache/v0.0.1`, `go/platform/web/v0.0.1`,
`go/slog-gcp/otelgcp/v0.0.2`.

### Added

- **go/retry**: zero-dependency general-purpose exponential backoff retry
  engine with fail-fast `Permanent` error classification and HTTP client
  `Transport` retrier.
- **go/retry/gcp**: extends `go/retry` with comprehensive Google API/GCP error
  classification (rate limits, transient network failures, quota exceeded, and
  OAuth/DWD permanent fail-fast cases).
- **go/dataquality/datadiff**: deep comparison and data validation tool for
  schemas, volume, and metric stats with configurable tolerance limits.
- **go/observability/audit**: production-proven audit logger with structured
  file outputs, automatic file-size rotation, and scorecard readers.
- **go/discovery/privacyfilter**: context-aware scan and redaction filter that
  skips sensitive directory patterns and redacts exposed credentials/tokens.
- **go/discovery/search**: core interfaces and data structures for resilient
  document search, indexing, scoring, and text extraction logic.
- **go/discovery/search/searchtest**: reusable contract tests for
  `search.Index` adapters.
- **go/platform/apierr**: REST API/gRPC error mapping layer with retryable
  classification, unified status responses, and error-unwrapping middleware.
- **go/platform/async**: thread-safe task coordinator for submitting,
  fetching, and pruning background asynchronous tasks.
- **go/platform/cache**: generic, concurrent-safe in-memory TTL cache.
- **go/platform/web**: project-agnostic HTTP server, client, and response
  utilities.
- Root `.golangci.yml` using the standard library lint profile.
- **skills/diffract-review**: agentic Diffract review skill with 9 parallel
  tool-equipped lens agents and CHECK mediator, based on contextvibes/diffract.
- **blueprints/service/.ko.yaml**: golden ko build template for Go Cloud Run
  services (pinned Chainguard static base, reproducible builds).
- **skills/ko-build**: skill for setting up ko container builds with CI/CD
  patterns and troubleshooting.
- **documentation**: SRE-hardened `README.md` files for the core Go modules
  following the standard template pattern.


## [go/slog-gcp/otelgcp/v0.0.1] - 2026-06-28

### Added

- Initial release of the `slog-gcp/otelgcp` module
- OpenTelemetry span context integration as a resolver for `slog-gcp`

## [go/slog-gcp/v0.0.1] - 2026-06-28

### Added

- Initial release of the `slog-gcp` module
- `slog.Handler` decorator with GCP Cloud Logging JSON output
- Cloud Trace header extraction via HTTP middleware
- Cloud Error Reporting integration via `ErrorAttrs()`
- One-call `Setup()` for Cloud Run services
- Test helpers (`SyncBuffer`, `LogEntries`, assertion functions)
- Godoc examples (`ExampleSetup`, `ExampleNewHandler`, `ExampleErrorAttrs`)

## [repository] - 2026-06-28

### Added

- Initial repository structure with 5-concern layout (`go/`, `contracts/`, `skills/`, `blueprints/`, `docs/`)
- Documentation vault following the 8-folder Open Knowledge Format
- GitHub issue templates (bug report, feature request) and PR template
- Git hooks for conventional commit validation and secret scanning
- CI pipeline with dynamic Go module discovery
- Contributor Covenant Code of Conduct
- Security policy with responsible disclosure process
- Dual licensing: Apache-2.0 (code) and CC-BY-4.0 (documentation)
