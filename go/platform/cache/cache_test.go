package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheBasicOperations(t *testing.T) {
	c := New[string](0)
	defer func() {
		_ = c.Close()
	}()

	c.Set("foo", "bar", 0)

	val, found := c.Get("foo")
	assert.True(t, found)
	assert.Equal(t, "bar", val)

	c.Delete("foo")
	_, found = c.Get("foo")
	assert.False(t, found)
}

func TestCacheExpiration(t *testing.T) {
	c := New[int](0)
	defer func() {
		_ = c.Close()
	}()

	// Set with 2ms TTL.
	c.Set("short", 42, 2*time.Millisecond)

	val, found := c.Get("short")
	assert.True(t, found)
	assert.Equal(t, 42, val)

	// Wait 5ms for item to expire.
	time.Sleep(5 * time.Millisecond)

	_, found = c.Get("short")
	assert.False(t, found)
}

func TestCacheClear(t *testing.T) {
	c := New[string](0)
	defer func() {
		_ = c.Close()
	}()

	c.Set("k1", "v1", 0)
	c.Set("k2", "v2", 0)

	c.Clear()

	_, found := c.Get("k1")
	assert.False(t, found)
	_, found = c.Get("k2")
	assert.False(t, found)
}

func TestBackgroundCleanupWorker(t *testing.T) {
	// Startup cache with active 1ms background cleaner.
	c := New[string](1 * time.Millisecond)

	c.Set("key", "val", 2*time.Millisecond)

	// Wait 5ms to guarantee both item expiration and background worker tick execution.
	time.Sleep(5 * time.Millisecond)

	c.mu.RLock()
	_, found := c.items["key"]
	c.mu.RUnlock()

	assert.False(t, found, "item should be physically removed from internal map by background worker")

	err := c.Close()
	require.NoError(t, err)
}

func TestConcurrentAccess(t *testing.T) {
	c := New[int](0)
	defer func() {
		_ = c.Close()
	}()

	var wg sync.WaitGroup
	workers := 10
	iterations := 100

	wg.Add(workers * 2)

	// Launch concurrent writers.
	for i := range workers {
		go func(workerID int) {
			defer wg.Done()
			for range iterations {
				c.Set("concurrent_key", workerID, 0)
			}
		}(i)
	}

	// Launch concurrent readers.
	for range workers {
		go func() {
			defer wg.Done()
			for range iterations {
				_, _ = c.Get("concurrent_key")
			}
		}()
	}

	wg.Wait()
}
