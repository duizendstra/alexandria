---
uuid: 4a2d8e41-6fb7-4b42-a89e-26f6323c9de2
title: "Writing Self-Documenting, Zero-Rot Go Packages"
domain: "architecture"
type: "guide"
diataxis_quadrant: "how-to"
status: "active"
maturity: "standard"
owner: "@duizendstra"
created_at: "2026-07-13T10:52:00Z"
updated_at: "2026-07-13T10:52:00Z"
summary: >
  Details the symbiotic 'doc.go' and 'example_test.go' patterns required to write
  self-documenting, compiler-validated Go library packages in Alexandria.
audience: [public]
tags: [ "go", "documentation", "dx", "agent-resilience" ]
relations:
  - { type: "extends", target: "ee14bc6c-1349-411a-8bb4-f58c70a84e20" } # DDD Boundaries
---

# Writing Self-Documenting, Zero-Rot Go Packages

## 🏛️ Context & Objective

In modern cloud engineering architectures, high-quality documentation is not a luxury—it is a core requirement for system resilience, developer velocity (DX), and **agentic ease of ingestion (AX)**.

Historically, documentation suffers from **decay (rot)**: as code refactorings occur, text-based guides (like `README.md` or external wikis) are rarely updated, leading to stale and misleading information.

To prevent documentation rot, Alexandria enforces the **Symbiotic Documentation Pattern**: a dual-layered documentation architecture combining conceptual high-level designs inside `doc.go` with compiler-validated code blocks inside `example_test.go`.

---

## 🧭 The Symbiotic Documentation Pattern

```mermaid
graph TD
    %% Styling
    classDef docStyle fill:#1e1e2e,stroke:#3b4252,stroke-width:2px,color:#cdd6f4;
    classDef testStyle fill:#181825,stroke:#585b70,stroke-width:2px,color:#a6adc8;

    subgraph "doc.go (SRE Operator's Guide)"
        A[doc.go]:::docStyle
        A1[• Architectural Intent & Rationale]:::docStyle
        A2[• Identity Translation & Bounded Contexts]:::docStyle
        A3[• SRE Operational Guides & Admin Links]:::docStyle
        A --> A1
        A --> A2
        A --> A3
    end

    subgraph "example_test.go (Runnable Contracts)"
        B[example_test.go]:::testStyle
        B1[• Compilable Code Examples]:::testStyle
        B2[• Direct & Subject Impersonation Samples]:::testStyle
        B3[• Compile-Only Guards (No Output Comments)]:::testStyle
        B --> B1
        B --> B2
        B --> B3
    end

    A -. "Integrates & References" .-> B
```

### 1. `doc.go` (The SRE Operator's Guide)
Every package inside the Alexandria ecosystem **must** contain a `doc.go` file. It serves as the single source of truth for the package's architecture and operational rules.

#### Key Content Requirements:
*   **Bounded Context Mapping**: Define the package’s role within Domain-Driven Design (e.g., *Infrastructure Identity Translator*).
*   **SRE Guardrails**: Detail critical security boundaries, least-privilege scoping rules, and local caching paths (such as `~/.kratos/`).
*   **Administrative Actions**: Document necessary external actions, such as deep-links to the Google Admin Console for Domain-Wide Delegation authorizations.
*   **Resilience & Backoff Policies**: Explain how network operations are wrapped in retry backoffs to handle transient 5xx or 429 rate limit failures.

### 2. `example_test.go` (The Runnable Contract)
To guarantee that documentation never rots, usage examples must be written inside a compilable test file named `example_test.go` (under `package <name>_test`).

#### Key Content Requirements:
*   **Compiler Validation**: Code blocks are validated directly by the compiler during `go test`. Any API change that breaks consumption will break the build, forcing engineers (and agents) to keep documentation in sync.
*   **Compile-Only SRE Guards**: For examples with external side effects (like launching HTTP local listener ports or connecting to GCP APIs), **do not** write any `// Output:` comments. This ensures they are checked for compile-time correctness in headless CI environments without executing network calls or hanging indefinitely.
*   **Happy & Edge Paths**: Provide clear, copy-pasteable examples for direct service account authentication and Domain-Wide Delegation user impersonation.

---

## 🛠️ Step-by-Step Package Creation Blueprint

When creating a new package in Alexandria (e.g. `google/drive`):

1.  **Define domain interfaces (Ports)**: Place raw interfaces in the domain directory.
2.  **Initialize `doc.go`**: Create `doc.go` inside the package directory, outlining its boundaries and SRE considerations.
3.  **Implement the package logic**: Write code conforming to the interfaces.
4.  **Create `example_test.go`**: Write clear, compile-only examples in `example_test.go` under `package drive_test`.
5.  **Run TDD validation loops via the standard toolchain**:
    ```bash
    go test -v ./...
    ```
