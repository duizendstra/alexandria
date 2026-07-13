package async

import (
	"context"
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
