# go/contracts

`go/contracts` contains the compiled Protocol Buffers and ConnectRPC definitions that form the Ubiquitous Language of the Alexandria platform, ensuring type safety and API compatibility across all distributed services.

## Features

- **Ubiquitous Language**: Standardized protobuf messages mapping across domains such as Workspace, Collaboration, registry, and intelligence.
- **Type-Safe ConnectRPC Integration**: Pre-generated client/server stubs for low-latency RPC streaming.
- **Zero Platform Dependency**: Clean library structure only importing standard protobuf and ConnectRPC runtimes.
- **Privacy Domain Support**: Embedded schemas for data anonymization, redaction, and access classification.

## Installation

```bash
go get github.com/duizendstra/alexandria/go/contracts
```

## Quick Start

### Utilizing the Generated Privacy Message Definitions

```go
package main

import (
	"fmt"

	privacyv1 "github.com/duizendstra/alexandria/go/contracts/common/privacy/v1"
)

func main() {
	// Instantiate a generated protobuf message for field anonymization
	rule := &privacyv1.FieldRule{
		FieldPath: "user.billing_address",
		Action:    privacyv1.AnonymizeAction_ANONYMIZE_ACTION_REDACT,
	}

	fmt.Printf("Configured field rule: path=%s action=%s\n", rule.GetFieldPath(), rule.GetAction())
}
```

## SRE & Performance Hardening details

1. **Minimized Allocations**: Leverages generated getters (e.g., `GetFieldPath()`) which return default zero values for nil pointers, preventing nil pointer panics on hotpaths.
2. **Backward Compatibility**: Fully conforms to the Proto3 specification where fields are optional and backward-compatible by default, preventing deserialization errors during rolling upgrades.
3. **Optimized Serialization**: Generated structs utilize `google.golang.org/protobuf/runtime/protoimpl` for fast, CPU-optimized binary and JSON serialization.

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- **Services on both ends of an RPC**: producers and consumers that must agree
  on message shape without sharing a repository.
- **Adapters mapping domain types to the wire**: code translating an internal
  model into the published language and back.

### Load-Bearing Promises
1. **Generated, Never Hand-Edited**: the Go code here is compiled from
   `contracts/proto/`. That directory is the source of truth; an edit made
   directly to a generated file is lost on the next generation.
2. **The Import Path States The Stability**: a package under `v1alpha1` may
   change shape between releases; a package under `v1` does not break within
   its major version. Consumers choosing an alpha package are choosing to
   track it.
3. **Runtime Dependencies Only**: this module imports the protobuf and
   ConnectRPC runtimes and nothing else from the platform. A dependency added
   here reaches every service on both ends of every call.
4. **Lint Is Enforced, Breaking-Change Detection Is Not**: `buf lint` runs
   against the `STANDARD` rule set. There is no `breaking` configuration, so
   wire compatibility rests on review rather than on a gate — treat any change
   to an existing message as needing that review.
