package names_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/internal/names"
)

func TestDuplicate(t *testing.T) {
	t.Parallel()

	ident := func(s *string) string { return *s }

	tests := []struct {
		name    string
		items   []string
		want    string
		wantDup bool
	}{
		{name: "nil"},
		{name: "single", items: []string{"a"}},
		{name: "unique", items: []string{"a", "b", "c"}},
		{name: "adjacent repeat", items: []string{"a", "a"}, want: "a", wantDup: true},
		{name: "first repeat wins", items: []string{"a", "b", "b", "a"}, want: "b", wantDup: true},
		{name: "empty strings repeat", items: []string{"", ""}, want: "", wantDup: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, dup := names.Duplicate(tt.items, ident)
			if dup != tt.wantDup || got != tt.want {
				t.Fatalf("Duplicate(%q) = (%q, %v), want (%q, %v)", tt.items, got, dup, tt.want, tt.wantDup)
			}
		})
	}
}

func TestDuplicateKeyedStruct(t *testing.T) {
	t.Parallel()

	type item struct {
		Name  string
		Other int
	}

	items := []item{{Name: "x", Other: 1}, {Name: "y", Other: 2}, {Name: "x", Other: 3}}

	got, dup := names.Duplicate(items, func(it *item) string { return it.Name })
	if !dup || got != "x" {
		t.Fatalf("Duplicate = (%q, %v), want (\"x\", true)", got, dup)
	}
}
