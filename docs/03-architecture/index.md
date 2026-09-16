# 03 — Architecture

This folder documents the technical design principles and structural patterns shared across all Alexandria modules.

How these modules reach a downstream repository — and how a package from one ascends into Alexandria — is the [Ascension & Consumption Protocol](../07-playbooks/ascension-and-consumption.md).

## What Belongs Here

- **Package Design Patterns** — Conventions for public API surface, internal packages, and option functions.
- **Module Boundaries** — Rules for dependency isolation between modules.
- **Decorator Patterns** — How middleware, wrappers, and composable behaviors are structured.
- **Handler Chains** — Patterns for request/response pipelines and processing chains.
- **Agentic Platform Engineering** — Architectural guardrails and verification workflows for AI agents and human engineers.

## Contents

* [Domain-Driven Design (DDD) & Clean Architecture Boundaries](domain-driven-design-boundaries.md) - Defines the core, standard-based directory conventions, layer boundaries, and dependency directions enforced across all ecosystem software modules.
* [Go as the Foundation for Agentic Platform Engineering](go-agentic-engineering.md) - Explains why Alexandria is built on Go for human-agent pair programming, detailing the verification-first paradigm, deterministic self-correction loops, standard library primacy, native fuzz testing, and automated ecosystem modernization.
* [Governance Bounded Context: Domain Model](governance-domain-model.md) - Describes the go/governance domain model — plan, scope, tiers, hierarchy, classification, and exports — and how the iac/governance blueprint and iac/pulumi/gcpinfra adapters realize it on Google Cloud.
* [Identity-Aware Proxy (IAP) & Split-Microservice Security Topology](iap-cloudrun-native-validation.md) - Outlines the architectural topology, split-microservice boundary layout, and zero-trust security model for the GCP Cloud Run Native IAP and Google Groups authorization validation stack.
* [Split-Microservice Security Architecture Document Template](gcp-microservice-security-template.md) - Standardized Open Knowledge Format (OKF) template for documenting multi-service systems, boundary isolation, edge access-control mechanisms, and directory integrations.
* [Writing Self-Documenting, Zero-Rot Go Packages](writing-enterprise-go-packages.md) - Details the symbiotic 'doc.go' and 'example_test.go' patterns required to write self-documenting, compiler-validated Go library packages in Alexandria.
