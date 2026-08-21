package datadiff_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/duizendstra/alexandria/go/dataquality/datadiff"
)

func TestSetDiff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		left          []string
		right         []string
		wantLeftOnly  []string
		wantRightOnly []string
		wantIntersect []string
		wantMatch     bool
		wantSubset    bool
		wantSuperset  bool
		wantDisjoint  bool
	}{
		{
			name:          "identical slices",
			left:          []string{"a", "b", "c"},
			right:         []string{"a", "b", "c"},
			wantLeftOnly:  nil,
			wantRightOnly: nil,
			wantIntersect: []string{"a", "b", "c"},
			wantMatch:     true,
			wantSubset:    true,
			wantSuperset:  true,
			wantDisjoint:  false,
		},
		{
			name:          "disjoint slices",
			left:          []string{"a", "b"},
			right:         []string{"c", "d"},
			wantLeftOnly:  []string{"a", "b"},
			wantRightOnly: []string{"c", "d"},
			wantIntersect: nil,
			wantMatch:     false,
			wantSubset:    false,
			wantSuperset:  false,
			wantDisjoint:  true,
		},
		{
			name:          "partial overlap with duplicates",
			left:          []string{"a", "b", "b", "c"},
			right:         []string{"b", "c", "c", "d"},
			wantLeftOnly:  []string{"a"},
			wantRightOnly: []string{"d"},
			wantIntersect: []string{"b", "c"},
			wantMatch:     false,
			wantSubset:    false,
			wantSuperset:  false,
			wantDisjoint:  false,
		},
		{
			name:          "left is subset of right",
			left:          []string{"a", "b"},
			right:         []string{"a", "b", "c"},
			wantLeftOnly:  nil,
			wantRightOnly: []string{"c"},
			wantIntersect: []string{"a", "b"},
			wantMatch:     false,
			wantSubset:    true,
			wantSuperset:  false,
			wantDisjoint:  false,
		},
		{
			name:          "both empty",
			left:          nil,
			right:         nil,
			wantLeftOnly:  nil,
			wantRightOnly: nil,
			wantIntersect: nil,
			wantMatch:     true,
			wantSubset:    true,
			wantSuperset:  true,
			wantDisjoint:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res := datadiff.SetDiff(tc.left, tc.right)

			if !reflect.DeepEqual(res.LeftOnly, tc.wantLeftOnly) {
				t.Errorf("LeftOnly = %v, want %v", res.LeftOnly, tc.wantLeftOnly)
			}
			if !reflect.DeepEqual(res.RightOnly, tc.wantRightOnly) {
				t.Errorf("RightOnly = %v, want %v", res.RightOnly, tc.wantRightOnly)
			}
			if !reflect.DeepEqual(res.Intersection, tc.wantIntersect) {
				t.Errorf("Intersection = %v, want %v", res.Intersection, tc.wantIntersect)
			}
			if got := res.Match(); got != tc.wantMatch {
				t.Errorf("Match() = %v, want %v", got, tc.wantMatch)
			}
			if got := res.IsSubset(); got != tc.wantSubset {
				t.Errorf("IsSubset() = %v, want %v", got, tc.wantSubset)
			}
			if got := res.IsSuperset(); got != tc.wantSuperset {
				t.Errorf("IsSuperset() = %v, want %v", got, tc.wantSuperset)
			}
			if got := res.IsDisjoint(); got != tc.wantDisjoint {
				t.Errorf("IsDisjoint() = %v, want %v", got, tc.wantDisjoint)
			}
		})
	}
}

func TestMapDiff(t *testing.T) {
	t.Parallel()

	left := map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}
	right := map[string]int{
		"b": 2,
		"c": 99,
		"d": 4,
	}

	res := datadiff.MapDiff(left, right, nil)

	if res.Match() {
		t.Errorf("expected match to be false")
	}

	if len(res.Identical) != 1 || res.Identical["b"] != 2 {
		t.Errorf("Identical unexpected: %+v", res.Identical)
	}
	if len(res.LeftOnly) != 1 || res.LeftOnly["a"] != 1 {
		t.Errorf("LeftOnly unexpected: %+v", res.LeftOnly)
	}
	if len(res.RightOnly) != 1 || res.RightOnly["d"] != 4 {
		t.Errorf("RightOnly unexpected: %+v", res.RightOnly)
	}
	if len(res.Mismatched) != 1 || res.Mismatched["c"].Left != 3 || res.Mismatched["c"].Right != 99 {
		t.Errorf("Mismatched unexpected: %+v", res.Mismatched)
	}

	summary := res.Summary()
	if summary == "" {
		t.Errorf("expected non-empty summary")
	}

	// Test identical maps with custom equals.
	leftIdentical := map[string]string{"k": "V"}
	rightIdentical := map[string]string{"k": "v"}
	caseInsensitive := func(a, b string) bool {
		return len(a) == len(b) // Dummy custom equality.
	}
	res2 := datadiff.MapDiff(leftIdentical, rightIdentical, caseInsensitive)
	if !res2.Match() {
		t.Errorf("expected custom equality match")
	}
}

func TestThreeWayDiff(t *testing.T) {
	t.Parallel()

	t.Run("clean migration pass", func(t *testing.T) {
		t.Parallel()

		baseline := []string{"f1", "f2", "f3"}
		target := []string{"f1", "f2", "f3"}
		leftovers := []string{}

		res := datadiff.ThreeWayDiff(baseline, target, leftovers)
		if !res.Passed() {
			t.Fatalf("expected passed=true, got %+v", res)
		}
		if len(res.Moved) != 3 {
			t.Errorf("expected 3 moved, got %d", len(res.Moved))
		}
	})

	t.Run("leftovers and missing items", func(t *testing.T) {
		t.Parallel()

		baseline := []string{"f1", "f2", "f3", "f4"}
		target := []string{"f1", "f5"}
		leftovers := []string{"f2"}

		res := datadiff.ThreeWayDiff(baseline, target, leftovers)
		if res.Passed() {
			t.Fatalf("expected passed=false")
		}

		// f1 moved.
		if !reflect.DeepEqual(res.Moved, []string{"f1"}) {
			t.Errorf("Moved = %v, want [f1]", res.Moved)
		}
		// f2 leftover.
		if !reflect.DeepEqual(res.Leftovers, []string{"f2"}) {
			t.Errorf("Leftovers = %v, want [f2]", res.Leftovers)
		}
		// f3, f4 missing.
		sort.Strings(res.MissingFromTarget)
		if !reflect.DeepEqual(res.MissingFromTarget, []string{"f3", "f4"}) {
			t.Errorf("MissingFromTarget = %v, want [f3, f4]", res.MissingFromTarget)
		}
		// f5 unaccounted.
		if !reflect.DeepEqual(res.UnaccountedTarget, []string{"f5"}) {
			t.Errorf("UnaccountedTarget = %v, want [f5]", res.UnaccountedTarget)
		}
	})
}

func FuzzSetDiff(f *testing.F) {
	f.Add("a,b,c", "b,c,d")
	f.Add("", "")
	f.Add("x", "y")

	f.Fuzz(func(t *testing.T, s1, s2 string) {
		left := []byte(s1)
		right := []byte(s2)

		res := datadiff.SetDiff(left, right)

		// Invariant: len(Intersection) + len(LeftOnly) == number of unique elements in left.
		leftUnique := make(map[byte]struct{})
		for _, b := range left {
			leftUnique[b] = struct{}{}
		}

		if len(res.Intersection)+len(res.LeftOnly) != len(leftUnique) {
			t.Errorf("SetDiff invariant violated: intersect(%d) + leftOnly(%d) != leftUnique(%d)",
				len(res.Intersection), len(res.LeftOnly), len(leftUnique))
		}
	})
}
