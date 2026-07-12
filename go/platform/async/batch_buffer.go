package async

import (
	"context"
	"fmt"
	"sync"
)

// BatchBuffer handles the in-memory accumulating and batch flushing of items of any type T.
type BatchBuffer[T any] struct {
	mu      sync.Mutex
	items   []T
	limit   int
	onFlush func(ctx context.Context, batch []T) error
}

// NewBatchBuffer creates a new type-safe BatchBuffer.
func NewBatchBuffer[T any](limit int, onFlush func(ctx context.Context, batch []T) error) *BatchBuffer[T] {
	if limit <= 0 {
		limit = 500
	}
	return &BatchBuffer[T]{
		limit:   limit,
		onFlush: onFlush,
		items:   make([]T, 0, limit),
	}
}

// Add appends an item to the buffer. If the buffer limit is reached, it automatically triggers a flush.
func (b *BatchBuffer[T]) Add(ctx context.Context, item T) error {
	b.mu.Lock()
	b.items = append(b.items, item)

	if len(b.items) >= b.limit {
		// Release lock to avoid blocking during network I/O in the flush function
		batch := b.items
		b.items = make([]T, 0, b.limit)
		b.mu.Unlock()

		if err := b.onFlush(ctx, batch); err != nil {
			return fmt.Errorf("failed to flush buffer batch: %w", err)
		}
		return nil
	}

	b.mu.Unlock()
	return nil
}

// Flush forces a synchronization of any remaining items in the buffer.
func (b *BatchBuffer[T]) Flush(ctx context.Context) error {
	b.mu.Lock()
	if len(b.items) == 0 {
		b.mu.Unlock()
		return nil
	}

	batch := b.items
	b.items = make([]T, 0, b.limit)
	b.mu.Unlock()

	if err := b.onFlush(ctx, batch); err != nil {
		return fmt.Errorf("failed to flush remaining buffer items: %w", err)
	}
	return nil
}

// Len returns the number of items currently in the buffer.
func (b *BatchBuffer[T]) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.items)
}
