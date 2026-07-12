---
uuid: ee14bc6c-1349-411a-8bb4-f58c70a84e20
title: "Domain-Driven Design (DDD) & Clean Architecture Boundaries"
domain: "architecture"
type: "guide"
diataxis_quadrant: "explanation"
status: "active"
maturity: "standard"
owner: "@duizendstra"
created_at: "2026-03-04T09:00:00Z"
updated_at: "2026-07-12T14:30:00Z"
summary: >
  Defines the core, standard-based directory conventions, layer boundaries, and dependency directions
  enforced across all ecosystem software modules.
audience: [public]
tags: [ "architecture", "ddd", "boundaries" ]
relations: []
---
# DDD & Clean Architecture Boundaries

## Architectural Objective

To establish strict, clean separation of concern across our codebases, ensuring that our core business logic remains pristine, easily mockable, and entirely insulated from infrastructure, database, and third-party transport layers.

This architecture enforces **dependency inversion**: the core domain logic defines the interfaces (Ports), and outer infrastructure layers implement those interfaces (Adapters).

```
  +--------------------------------------------------------+
  |                   Outer Adapters Layer                 |
  |    (HTTP Handlers, SQL Drivers, Google Client SDKs)    |
  +---------------------------+----------------------------+
                              |
                              | Implements & Calls
                              v
  +--------------------------------------------------------+
  |                    Core Domain Layer                   |
  |     (Domain Models, Domain Services, Port Interfaces)  |
  +--------------------------------------------------------+
```

---

## Layer Definitions & Standards

### 1. The Core Domain Layer (Pure Domain)
*   **Purpose**: Represents the pure system ontology, containing the entity structures, validation logic, domain events, and interface definitions (Ports).
*   **Isolation Standards**:
    *   **Strict Dependency Purity** — Must have **zero dependencies** on external libraries (excluding standard Go types). 
    *   **Interface Enforcement (Ports)** — All storage, networks, and integrations must be declared as interfaces. For example, a domain service must interact with an interface `Repository`, not a concrete SQL or BigQuery connection pool.
    *   **No Configuration Leakage** — Configuration structures (e.g. reading from `.env` or YAML) are prohibited. Configuration is mapped into native domain settings in the application layer.

### 2. The Application Layer (Use Cases)
*   **Purpose**: Orchestrates the execution of use cases, loading data via repositories, coordinating domain actions, and pushing state mutations.
*   **Isolation Standards**:
    *   **No Infrastructure Leakage** — Does not parse JSON or handle HTTP protocols. It receives plain Go structures (Requests) and returns standard Go structures (Responses).

### 3. The Outer Adapters Layer (Infrastructure & Transport)
*   **Purpose**: Implements the interfaces defined in the domain layer. This layer contains database drivers (Postgres, SQLite), Google client libraries (Drive, BigQuery), file system rotators, and REST/gRPC routing frameworks.
*   **Isolation Standards**:
    *   **Decoupled Types** — Outer types (e.g. database row structs, third-party API request payloads) must never leak into the domain layer. All outer structures must be mapped to native domain entities before returning across the domain boundary.

---

## Repository Implementation Rules

*   **Idempotency by Design** — Infrastructure repositories must guarantee idempotent behavior. Save and write operations must be safe to retry infinitely under network failure conditions.
*   **Batch Operations** — For high-throughput persistence paths, repositories must expose batch write interfaces (`SaveBatch`) to buffer entries and flush efficiently in transactions, avoiding individual lock contentions.
*   **Read from Primary** — Under dual-write or federated persistence topologies, all read operations must execute against the primary, consistency-guaranteed store. Secondary analytics indexes are written asynchronously.
