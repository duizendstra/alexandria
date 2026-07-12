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
