---
title: Security
domain: security
type: index
diataxis_quadrant: explanation
status: active
maturity: standard
audience: [public]
owner: "@duizendstra"
summary: Security policies for Alexandria — workload security today, with dependency policy and supply-chain hardening planned.
uuid: 7ed79a00-7d71-4b95-9e54-28ac553b32aa
created_at: "2026-06-28T11:41:03Z"
updated_at: "2026-07-19T12:00:00Z"
tags: [ "index", "security" ]
relations: []
---

# 05 — Security

This folder documents the security policies and practices that protect the Alexandria supply chain and its consumers.

## What Belongs Here

- **Workload Security** — Data classification, secret management, and cloud authentication policies (documented below).
- **Vulnerability Handling** — The responsible-disclosure process lives in the repository-root [SECURITY.md](../../SECURITY.md).
- **Dependency Policy & Supply Chain Security** *(planned)* — Today the dependency policy is enforced mechanically (the `depguard` allowlist in `.golangci.yml` and the CI `mod-hygiene` job) but not yet written up here; supply-chain measures (checksums, reproducible builds, signing) are future work.

## Contents

- [zero-trust-workload-security.md](zero-trust-workload-security.md) — Standardized data classification, secure secret management (pass), and keyless cloud authentication policies.
- [SECURITY.md](../../SECURITY.md) *(repository root)* — Responsible disclosure and supported-versions policy.
