package coordination_test

import (
	"errors"
	"testing"

	"github.com/duizendstra/alexandria/go/platform/coordination"
)

func TestNopExcluder(t *testing.T) {
	t.Parallel()

	ex := coordination.NopExcluder()
	release, err := ex.Acquire("any-subject")
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	if release == nil {
		t.Fatal("expected non-nil release func")
	}

	// Calling release multiple times should be safe.
	release()
	release()
}

func TestErrLocked(t *testing.T) {
	t.Parallel()

	if coordination.ErrLocked == nil {
		t.Fatal("expected ErrLocked to be non-nil")
	}
	if !errors.Is(coordination.ErrLocked, coordination.ErrLocked) {
		t.Fatal("expected errors.Is match on ErrLocked")
	}
}
