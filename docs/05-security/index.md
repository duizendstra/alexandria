# 05 — Security

This folder documents the security policies and practices that protect the Alexandria supply chain and its consumers.

## What Belongs Here

- **Workload Security** — Data classification, secret management, and cloud authentication policies (documented below).
- **Vulnerability Handling** — The responsible-disclosure process lives in the repository-root [SECURITY.md](../../SECURITY.md).
- **Dependency Policy & Supply Chain Security** *(planned)* — Today the dependency policy is enforced mechanically (the `depguard` allowlist in `.golangci.yml` and the CI `mod-hygiene` job) but not yet written up here; supply-chain measures (checksums, reproducible builds, signing) are future work.

## Contents

* [Zero-Trust Workload Security & Credential Hygiene](zero-trust-workload-security.md) - Defines the standard data classification, secure secret management, and keyless cloud authentication policies enforced across all modules.
* [SECURITY.md](../../SECURITY.md) - Responsible disclosure and supported-versions policy; lives at the repository root, outside the vault.
