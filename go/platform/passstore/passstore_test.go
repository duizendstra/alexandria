package passstore_test

import (
	"strings"
	"testing"

	"github.com/duizendstra/alexandria/go/platform/passstore"
)

// missingPath is an entry no pass store contains; Show must error whether
// the pass binary is absent (exec error) or present (entry not found).
const missingPath = "alexandria-test/definitely/not-an-entry"

func TestShowMissingEntryErrors(t *testing.T) {
	if _, err := passstore.Show(missingPath); err == nil {
		t.Error("expected error for missing entry")
	} else if !strings.Contains(err.Error(), missingPath) {
		t.Errorf("error should name the path, got: %v", err)
	}
}

func TestMustShowPanicsOnError(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic for missing entry")
		}
	}()

	passstore.MustShow(missingPath)
}
