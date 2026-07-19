package async

import (
	"context"
	"fmt"
	"sync"
)

// FlushError reports a failed flush and carries the exact batch that could
// not be delivered. It is returned (wrapped) by Add and Flush whenever the
// onFlush callback fails, so callers can retry or salvage the batch:
//
//	var flushErr *async.FlushError[Event]
//	if errors.As(err, &flushErr) {
//		retryLater(flushErr.Items)
//	}
type FlushError[T any] struct {
	// Items is the batch that was passed to the failing flush callback.
	Items []T
	// Err is the error returned by the flush callback.
	Err error
}

// Error implements the error interface.
func (e *FlushError[T]) Error() string {
	return fmt.Sprintf("failed to flush batch of %d item(s): %v", len(e.Items), e.Err)
}

// Unwrap returns the underlying flush callback error so errors.Is/As work
// against sentinel errors returned by the callback.
func (e *FlushError[T]) Unwrap() error {
	return e.Err
}

// BatchBuffer provides a thread-safe generic slice buffer that automatically flushes
// its elements once a specified capacity limit is hit.
//
// Failure contract: when the onFlush callback returns an error, the failed
// batch is NOT retained by the buffer. Ownership of the batch transfers to
// the caller through the returned *FlushError[T] (see its Items field), which
// keeps retry policy in the caller's hands and avoids unbounded growth or
// reordering against concurrent Adds. Callers that cannot tolerate loss must
// handle the FlushError (retry, spill to disk, etc.).
type BatchBuffer[T any] struct {
	mu      sync.Mutex
	limit   int
	onFlush func(ctx context.Context, batch []T) error
	items   []T
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
//
// If the triggered flush fails, Add returns a *FlushError[T] carrying the
// batch that could not be delivered; the buffer no longer holds those items,
// so the caller is responsible for retrying or salvaging them.
func (b *BatchBuffer[T]) Add(ctx context.Context, item T) error {
	b.mu.Lock()
	b.items = append(b.items, item)

	if len(b.items) >= b.limit {
		// Release lock to avoid blocking during network I/O in the flush function.
		batch := b.items
		b.items = make([]T, 0, b.limit)
		b.mu.Unlock()

		if err := b.onFlush(ctx, batch); err != nil {
			return &FlushError[T]{Items: batch, Err: err}
		}

		return nil
	}

	b.mu.Unlock()

	return nil
}

// Flush forces a synchronization of any remaining items in the buffer.
//
// If the flush callback fails, Flush returns a *FlushError[T] carrying the
// batch that could not be delivered; the buffer no longer holds those items,
// so the caller is responsible for retrying or salvaging them.
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
		return &FlushError[T]{Items: batch, Err: err}
	}

	return nil
}

// Len returns the number of items currently in the buffer.
func (b *BatchBuffer[T]) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return len(b.items)
}
