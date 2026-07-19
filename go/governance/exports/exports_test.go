// Domain:  Governance
// Concern: Are all export names valid and unique?
package exports_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/governance/exports"
)

func TestExportNamesAreNonEmpty(t *testing.T) {
	names := []string{
		exports.OrgID,
		exports.BillingAccount,
		exports.RootFolderID,
		exports.FolderIDs,
	}
	for _, name := range names {
		if name == "" {
			t.Error("export name must not be empty")
		}
	}
}

func TestTagKeyID(t *testing.T) {
	if got := exports.TagKeyID("environment"); got != "environmentTagKeyId" {
		t.Errorf("TagKeyID(environment) = %q, want environmentTagKeyId", got)
	}
}

func TestExportNamesAreUnique(t *testing.T) {
	names := []string{
		exports.OrgID,
		exports.BillingAccount,
		exports.RootFolderID,
		exports.FolderIDs,
	}
	seen := make(map[string]bool)
	for _, name := range names {
		if seen[name] {
			t.Errorf("duplicate export name: %s", name)
		}
		seen[name] = true
	}
	t.Logf("All %d export names are valid and unique", len(names))
}
