package async

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestBatchBuffer(t *testing.T) {
	var mu sync.Mutex
	var flushed [][]int

	onFlush := func(ctx context.Context, batch []int) error {
		mu.Lock()
		defer mu.Unlock()
		// Make a copy of the batch slice to verify values.
		copied := make([]int, len(batch))
		copy(copied, batch)
		flushed = append(flushed, copied)

		return nil
	}

	buf := NewBatchBuffer[int](3, onFlush)

	// Add items to trigger automatic flushes.
	ctx := context.Background()
	if err := buf.Add(ctx, 1); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if err := buf.Add(ctx, 2); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if buf.Len() != 2 {
		t.Errorf("expected buffer len 2, got %d", buf.Len())
	}

	// Triggering 3rd item should flush automatically.
	if err := buf.Add(ctx, 3); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if buf.Len() != 0 {
		t.Errorf("expected buffer len 0 after flush, got %d", buf.Len())
	}

	// Manual Flush on remaining items.
	if err := buf.Add(ctx, 4); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if err := buf.Flush(ctx); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(flushed) != 2 {
		t.Fatalf("expected 2 flushed batches, got %d", len(flushed))
	}

	if len(flushed[0]) != 3 || flushed[0][0] != 1 || flushed[0][1] != 2 || flushed[0][2] != 3 {
		t.Errorf("first batch incorrect: %v", flushed[0])
	}

	if len(flushed[1]) != 1 || flushed[1][0] != 4 {
		t.Errorf("second batch incorrect: %v", flushed[1])
	}
}

// TestBatchBufferAddFlushErrorRecoverable is the regression test for silent
// data loss: when the size-triggered flush inside Add fails, the failed batch
// must be recoverable through the returned error so callers can retry it.
func TestBatchBufferAddFlushErrorRecoverable(t *testing.T) {
	sentinel := errors.New("sink unavailable")
	fail := true

	var mu sync.Mutex
	var flushed [][]int

	onFlush := func(_ context.Context, batch []int) error {
		mu.Lock()
		defer mu.Unlock()

		if fail {
			return sentinel
		}

		copied := make([]int, len(batch))
		copy(copied, batch)
		flushed = append(flushed, copied)

		return nil
	}

	buf := NewBatchBuffer[int](2, onFlush)
	ctx := context.Background()

	if err := buf.Add(ctx, 1); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Second Add hits the limit and triggers a flush, which fails.
	err := buf.Add(ctx, 2)
	if err == nil {
		t.Fatal("expected error from Add when flush fails, got nil")
	}

	var flushErr *FlushError[int]
	if !errors.As(err, &flushErr) {
		t.Fatalf("expected *FlushError[int], got %T: %v", err, err)
	}

	if len(flushErr.Items) != 2 || flushErr.Items[0] != 1 || flushErr.Items[1] != 2 {
		t.Fatalf("expected failed batch [1 2], got %v", flushErr.Items)
	}

	if !errors.Is(err, sentinel) {
		t.Errorf("expected error to wrap the flush callback error, got %v", err)
	}

	// The failed batch is handed to the caller; the buffer must not retain it.
	if buf.Len() != 0 {
		t.Errorf("expected buffer len 0 after failed flush, got %d", buf.Len())
	}

	// Caller can retry the exact failed batch once the sink recovers.
	mu.Lock()
	fail = false
	mu.Unlock()

	for _, item := range flushErr.Items {
		if err := buf.Add(ctx, item); err != nil {
			t.Fatalf("retry Add failed: %v", err)
		}
	}

	mu.Lock()
	defer mu.Unlock()

	if len(flushed) != 1 {
		t.Fatalf("expected 1 flushed batch after retry, got %d", len(flushed))
	}

	if len(flushed[0]) != 2 || flushed[0][0] != 1 || flushed[0][1] != 2 {
		t.Errorf("expected retried batch [1 2], got %v", flushed[0])
	}
}

// TestBatchBufferFlushErrorRecoverable covers the explicit Flush path: a
// failing flush callback must surface the failed batch instead of dropping it.
func TestBatchBufferFlushErrorRecoverable(t *testing.T) {
	sentinel := errors.New("write timeout")

	onFlush := func(_ context.Context, _ []string) error {
		return sentinel
	}

	buf := NewBatchBuffer[string](10, onFlush)
	ctx := context.Background()

	if err := buf.Add(ctx, "a"); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if err := buf.Add(ctx, "b"); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	err := buf.Flush(ctx)
	if err == nil {
		t.Fatal("expected error from Flush when callback fails, got nil")
	}

	var flushErr *FlushError[string]
	if !errors.As(err, &flushErr) {
		t.Fatalf("expected *FlushError[string], got %T: %v", err, err)
	}

	if len(flushErr.Items) != 2 || flushErr.Items[0] != "a" || flushErr.Items[1] != "b" {
		t.Fatalf("expected failed batch [a b], got %v", flushErr.Items)
	}

	if !errors.Is(err, sentinel) {
		t.Errorf("expected error to wrap the flush callback error, got %v", err)
	}

	if buf.Len() != 0 {
		t.Errorf("expected buffer len 0 after failed flush, got %d", buf.Len())
	}
}
