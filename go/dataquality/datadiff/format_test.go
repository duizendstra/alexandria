package datadiff_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/dataquality/datadiff"
)

func TestParseTarget(t *testing.T) {
	target, err := datadiff.ParseTarget("my-project.my_dataset.my_table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if target.Project != "my-project" {
		t.Errorf("Project = %q, want %q", target.Project, "my-project")
	}
	if target.Dataset != "my_dataset" {
		t.Errorf("Dataset = %q, want %q", target.Dataset, "my_dataset")
	}
	if target.Table != "my_table" {
		t.Errorf("Table = %q, want %q", target.Table, "my_table")
	}
}

func TestParseTarget_Invalid(t *testing.T) {
	cases := []string{"", "foo", "foo.bar"}
	for _, s := range cases {
		_, err := datadiff.ParseTarget(s)
		if err == nil {
			t.Errorf("expected error for %q", s)
		}
	}
}

func TestParseTargetPair(t *testing.T) {
	l, r, err := datadiff.ParseTargetPair("p1.d1.t1", "p2.d2.t2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if l.Project != "p1" || r.Project != "p2" {
		t.Errorf("projects: left=%q right=%q", l.Project, r.Project)
	}
}

func TestBillingProject_Explicit(t *testing.T) {
	got := datadiff.BillingProject("explicit", datadiff.Target{Project: "fallback"})
	if got != "explicit" {
		t.Errorf("got %q, want %q", got, "explicit")
	}
}

func TestBillingProject_Fallback(t *testing.T) {
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")
	got := datadiff.BillingProject("", datadiff.Target{Project: "from-target"})
	if got != "from-target" {
		t.Errorf("got %q, want %q", got, "from-target")
	}
}
