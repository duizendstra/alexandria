// Package cache provides a generic, concurrent-safe, low-overhead
// in-memory TTL caching component.
package cache

import (
	"context"
	"sync"
	"time"
)

// Item represents a single cached record with explicit expiration.
type Item[T any] struct {
	Value      T
	Expiration int64
}

// Expired returns true if the item has passed its time-to-live limit.
func (i Item[T]) Expired() bool {
	if i.Expiration <= 0 {
		return false
	}

	return time.Now().UnixNano() > i.Expiration
}

// Cache holds generic cached records with active TTL management.
type Cache[T any] struct {
	mu            sync.RWMutex
	items         map[string]Item[T]
	cleanupCancel context.CancelFunc
	cleanupWg     sync.WaitGroup
}

// New creates a new Cache instance with an optional background cleanup interval.
// If cleanupInterval is positive, a background goroutine is started to periodically
// remove expired items and prevent memory leaks. Call Close() to clean up resources.
func New[T any](cleanupInterval time.Duration) *Cache[T] {
	c := &Cache[T]{
		items: make(map[string]Item[T]),
	}

	if cleanupInterval > 0 {
		//nolint:gosec // Cancel function is explicitly called and managed inside c.Close()
		ctx, cancel := context.WithCancel(context.Background())
		c.cleanupCancel = cancel
		c.cleanupWg.Add(1)

		go c.startCleanupWorker(ctx, cleanupInterval)
	}

	return c
}

// Set adds or overwrites a key with a value and a given duration (TTL).
// If duration is <= 0, the item does not expire.
func (c *Cache[T]) Set(key string, val T, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiration int64
	if duration > 0 {
		expiration = time.Now().Add(duration).UnixNano()
	}

	c.items[key] = Item[T]{
		Value:      val,
		Expiration: expiration,
	}
}

// Get retrieves an item by key. Returns the zero value of T and false
// if the item is missing or has expired.
func (c *Cache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	item, found := c.items[key]
	c.mu.RUnlock()

	if !found {
		var zero T

		return zero, false
	}

	if item.Expired() {
		// Evict expired item on access.
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()

		var zero T

		return zero, false
	}

	return item.Value, true
}

// Delete explicitly removes a key from the cache.
func (c *Cache[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear flushes all items in the cache.
func (c *Cache[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]Item[T])
}

// Close stops the background cleanup worker and waits for it to exit.
func (c *Cache[T]) Close() error {
	if c.cleanupCancel != nil {
		c.cleanupCancel()
		c.cleanupWg.Wait()
	}

	return nil
}

func (c *Cache[T]) startCleanupWorker(ctx context.Context, interval time.Duration) {
	defer c.cleanupWg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.DeleteExpired()
		}
	}
}

// DeleteExpired iterates over the entire cache and removes all expired records.
func (c *Cache[T]) DeleteExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UnixNano()
	for k, item := range c.items {
		if item.Expiration > 0 && now > item.Expiration {
			delete(c.items, k)
		}
	}
}
