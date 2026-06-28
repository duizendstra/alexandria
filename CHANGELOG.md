# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

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
