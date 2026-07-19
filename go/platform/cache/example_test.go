package cache_test

import (
	"fmt"
	"time"

	"github.com/duizendstra/alexandria/go/platform/cache"
)

func ExampleNew() {
	// A positive cleanup interval starts a background worker that evicts
	// expired items; Close stops it.
	c := cache.New[string](time.Minute)
	defer func() { _ = c.Close() }()

	c.Set("greeting", "hello", time.Minute)

	if v, ok := c.Get("greeting"); ok {
		fmt.Println(v)
	}
	// Output: hello
}

func ExampleCache_Get() {
	// Interval 0 disables the background cleanup worker.
	c := cache.New[int](0)
	defer func() { _ = c.Close() }()

	// Duration <= 0 means the item never expires.
	c.Set("port", 8080, 0)

	v, ok := c.Get("port")
	fmt.Println(v, ok)

	_, ok = c.Get("missing")
	fmt.Println(ok)
	// Output:
	// 8080 true
	// false
}

func ExampleCache_Set_expiration() {
	c := cache.New[string](0)
	defer func() { _ = c.Close() }()

	c.Set("token", "abc123", time.Nanosecond)
	time.Sleep(10 * time.Millisecond) // Let the item expire.

	// Expired items are evicted on access.
	_, ok := c.Get("token")
	fmt.Println(ok)
	// Output: false
}

func ExampleCache_Delete() {
	c := cache.New[string](0)
	defer func() { _ = c.Close() }()

	c.Set("session", "active", time.Minute)
	c.Delete("session")

	_, ok := c.Get("session")
	fmt.Println(ok)
	// Output: false
}
