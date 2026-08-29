# go/retry (Deprecated)

> [!WARNING]
> **Deprecated**: This module is retained exclusively for backward compatibility.  
> The canonical retry and resilience primitives have moved to **[`go/platform/retry`](../platform/retry/)** (`github.com/duizendstra/alexandria/go/platform/retry`).

## Migration

Replace your imports with the new canonical path:

```go
// Old (Deprecated)
import "github.com/duizendstra/alexandria/go/retry"

// New (Canonical)
import "github.com/duizendstra/alexandria/go/platform/retry"
```
