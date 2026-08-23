# go/platform/cache

`go/platform/cache` provides a generic, concurrent-safe, in-memory TTL cache with optional background eviction.

## Features

- **Generic Over The Value Type**: `Cache[T]` stores any type directly — no `any` boxing and no type assertion at the call site. A miss returns the zero value of `T`.
- **Per-Entry TTL**: every `Set` carries its own duration. A duration of zero or less stores an entry that never expires.
- **Lazy Eviction On Read**: an entry past its expiry is deleted and reported as a miss the moment it is read, whether or not a background worker is running.
- **Optional Background Sweeper**: a positive `cleanupInterval` starts one worker goroutine that removes expired entries, so a key nobody reads again does not hold its value forever. `Close` cancels it and waits for it to exit.
- **Explicit Invalidation**: `Delete` drops one key, `Clear` empties the cache, `DeleteExpired` runs the sweep on demand.
- **Zero Runtime Dependencies**: standard library only; testify is a test-only dependency.

## Installation

```bash
go get github.com/duizendstra/alexandria/go/platform/cache
```

## Quick Start

### Caching Slow-Changing Lookups With A Time-To-Live

```go
package main

import (
	"fmt"
	"time"

	"github.com/duizendstra/alexandria/go/platform/cache"
)

type profile struct {
	DisplayName string
}

func main() {
	// A positive cleanup interval starts a background worker that evicts
	// expired entries. Close stops it and waits for it to exit.
	c := cache.New[profile](5 * time.Minute)
	defer func() { _ = c.Close() }()

	// Entries expire independently; this one is stale after 15 minutes.
	c.Set("u-1001", profile{DisplayName: "Reporting Service"}, 15*time.Minute)

	// A hit returns the stored value. A miss — absent or expired — returns
	// the zero value of the type and false, so there is nothing to unwrap.
	if p, ok := c.Get("u-1001"); ok {
		fmt.Println("cached:", p.DisplayName)
	}

	if _, ok := c.Get("u-9999"); !ok {
		fmt.Println("not cached; fetch it and Set it")
	}

	// A non-positive duration stores an entry with no expiry at all.
	c.Set("schema-version", profile{DisplayName: "v3"}, 0)

	// Invalidate explicitly when the underlying record changes.
	c.Delete("u-1001")
}
```

## SRE & Performance Hardening details

1. **At Most One Goroutine Per Cache**: the background sweeper is started only when `cleanupInterval` is positive, and `Close` cancels its context and blocks on a `sync.WaitGroup` until it has returned — a closed cache leaves no goroutine behind. A cache built with a non-positive interval starts nothing, and `Close` on it is still safe.
2. **Reads Take The Read Lock**: `Get` holds `RLock` for the map lookup, so concurrent readers do not serialise on each other. Only the eviction of an entry found expired takes the write lock.
3. **TTL Is The Only Eviction Policy**: the cache is unbounded in entry count. It suits a bounded key space — a per-request lookup table, a token cache — not an arbitrary-key memoiser, which needs a size bound this package does not provide.

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- **Request-path lookups of slow-changing reference data**: handlers that would
  otherwise re-fetch the same record on every request.
- **Long-lived daemons memoising credentials or metadata**: processes holding a
  value that is valid for a known window and must not be used past it.

### Load-Bearing Promises
1. **A Miss Is The Zero Value And False**: a key that is absent — or present but
   expired — returns `T`'s zero value and `false`. There is no third state and
   nothing to nil-check.
2. **Expiry Is Observed On Read, Not Only On Sweep**: an entry past its TTL is a
   miss the moment it is read, even when no background worker is running. A
   caller cannot be handed a stale value by configuring the sweep away.
3. **A Non-Positive TTL Means No Expiry**: an entry stored with a duration of
   zero or less stays readable indefinitely.
4. **The Sweeper Frees Memory, Not Just Visibility**: when a cleanup interval is
   configured, an expired entry is physically removed from the backing map
   without anyone reading the key — the reason the option exists.
5. **Concurrent Readers And Writers Are Safe**: parallel `Get` and `Set` on the
   same key are race-free under `-race`.
6. **`Close` Stops The Worker And Waits**: it returns only after the background
   goroutine has exited, and reports no error — so a test or a shutdown path can
   treat a closed cache as quiescent.
