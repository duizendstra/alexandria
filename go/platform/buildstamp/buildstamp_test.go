package buildstamp

import (
	"reflect"
	"strings"
	"testing"
)

const (
	shaA    = "1111111111111111111111111111111111111111"
	shaB    = "2222222222222222222222222222222222222222"
	toolNm  = "tool"
	libRev  = "abc1234"
	libName = "lib"
)

func TestStringRoundTripsThroughParse(t *testing.T) {
	in := Stamp{
		Name:    toolNm,
		Version: "1.4.0",
		Commit:  shaA,
		Dirty:   false,
		BuiltAt: "2026-01-02T15:04:05Z",
		Deps:    map[string]string{libName: libRev, "sdk": "def5678"},
	}
	out, err := ParseStamp(in.String())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Errorf("round trip changed the stamp:\n got %+v\nwant %+v", out, in)
	}
}

func TestStringIsStableAcrossDepOrder(t *testing.T) {
	// Map iteration order is random; a stamp line that reorders between runs
	// cannot be compared by a supervising script.
	s := Stamp{Name: toolNm, Commit: shaA, Deps: map[string]string{"z": "1", "a": "2", "m": "3"}}
	first := s.String()
	for range 20 {
		if got := s.String(); got != first {
			t.Fatalf("stamp line is not stable:\n%s\n%s", first, got)
		}
	}
	if !strings.Contains(first, "a=2 m=3 z=1") {
		t.Errorf("dependency stamps not sorted: %s", first)
	}
}

func TestParseStampRejectsMalformedLines(t *testing.T) {
	cases := []struct {
		name string
		line string
	}{
		{"empty", "   "},
		{"no commit field", "tool 1.0 dirty=false"},
		{"dirty not boolean", "tool commit=" + shaA + " dirty=maybe"},
		{"bare word after fields", "tool commit=" + shaA + " dirty=false stray"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseStamp(tc.line); err == nil {
				t.Errorf("expected an error for %q", tc.line)
			}
		})
	}
}

func TestParseStampKeepsUnknownKeysAsDeps(t *testing.T) {
	// Silently dropping provenance would make a supervising check verify less
	// than it claims to.
	s, err := ParseStamp("tool commit=" + shaA + " dirty=false somelib=cafe123")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := s.Deps["somelib"]; got != "cafe123" {
		t.Errorf("unknown key not kept as a dependency stamp: %+v", s.Deps)
	}
}

func TestMatchesAcceptsACleanExactBuild(t *testing.T) {
	s := Stamp{Commit: shaA, Dirty: false, Deps: map[string]string{libName: libRev}}
	if err := s.Matches(shaA); err != nil {
		t.Errorf("clean exact build must be accepted, got: %v", err)
	}
}

func TestMatchesRefusals(t *testing.T) {
	cases := []struct {
		name     string
		stamp    Stamp
		expected string
		contains string
	}{
		{
			name:     "different commit",
			stamp:    Stamp{Commit: shaB},
			expected: shaA,
			contains: "rebuild",
		},
		{
			name:     "unknown commit",
			stamp:    Stamp{Commit: Unknown},
			expected: shaA,
			contains: "no known commit",
		},
		{
			name:     "empty commit",
			stamp:    Stamp{},
			expected: shaA,
			contains: "no known commit",
		},
		{
			name:     "abbreviated commit",
			stamp:    Stamp{Commit: "1111111"},
			expected: shaA,
			contains: "full 40-character SHA",
		},
		{
			name:     "dirty tree",
			stamp:    Stamp{Commit: shaA, Dirty: true},
			expected: shaA,
			contains: "dirty working tree",
		},
		{
			name:     "dirty dependency",
			stamp:    Stamp{Commit: shaA, Deps: map[string]string{"lib": "abc1234-dirty"}},
			expected: shaA,
			contains: "not reproducible",
		},
		{
			name:     "unknown dependency",
			stamp:    Stamp{Commit: shaA, Deps: map[string]string{"lib": Unknown}},
			expected: shaA,
			contains: "unknown revision",
		},
		{
			name:     "expected commit abbreviated",
			stamp:    Stamp{Commit: shaA},
			expected: "1111111",
			contains: "not a full 40-character SHA",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.stamp.Matches(tc.expected)
			if err == nil {
				t.Fatalf("expected a refusal for %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.contains) {
				t.Errorf("error should explain the cause (%q), got: %v", tc.contains, err)
			}
		})
	}
}

func TestRequireDepsCatchesAMissingStamp(t *testing.T) {
	// Matches can only judge what is recorded; a build that forgot to record
	// its dependency would otherwise pass.
	s := Stamp{Commit: shaA}
	if err := s.Matches(shaA); err != nil {
		t.Fatalf("Matches alone should accept it: %v", err)
	}
	if err := s.RequireDeps("lib"); err == nil {
		t.Error("RequireDeps must refuse a stamp that does not record the dependency")
	}

	s.Deps = map[string]string{libName: libRev}
	if err := s.RequireDeps("lib"); err != nil {
		t.Errorf("a recorded clean dependency must be accepted: %v", err)
	}
}

func TestShort(t *testing.T) {
	cases := []struct {
		name  string
		stamp Stamp
		want  string
	}{
		{"clean", Stamp{Commit: shaA}, "1111111"},
		{"dirty", Stamp{Commit: shaA, Dirty: true}, "1111111-dirty"},
		{"with version", Stamp{Commit: shaA, Version: "1.4.0"}, "1.4.0 (1111111)"},
		{"short commit kept as is", Stamp{Commit: Unknown}, Unknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.stamp.Short(); got != tc.want {
				t.Errorf("Short() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetFallsBackToUnknown(t *testing.T) {
	// With no -ldflags values and no VCS settings, the commit must be Unknown
	// rather than empty: Matches refuses Unknown, but an empty string could be
	// mistaken for "not applicable".
	s := Get(toolNm, nil)
	if s.Commit == "" {
		t.Error("commit must never be empty")
	}
	if s.Name != toolNm {
		t.Errorf("name not carried, got %q", s.Name)
	}
}

func TestGetCopiesDeps(t *testing.T) {
	deps := map[string]string{libName: libRev}
	s := Get(toolNm, deps)
	deps["lib"] = "mutated"
	if s.Deps[libName] != libRev {
		t.Error("Get must copy the dependency map, not alias the caller's")
	}
}
