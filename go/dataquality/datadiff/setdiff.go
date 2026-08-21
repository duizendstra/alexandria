package datadiff

import "fmt"

// SetResult holds the outcome of an in-memory set comparison between two slices.
type SetResult[T comparable] struct {
	LeftOnly     []T
	RightOnly    []T
	Intersection []T
}

// Match returns true if both sets contain exactly the same unique elements.
func (r SetResult[T]) Match() bool {
	return len(r.LeftOnly) == 0 && len(r.RightOnly) == 0
}

// IsSubset returns true if every unique element in Left is also present in Right.
func (r SetResult[T]) IsSubset() bool {
	return len(r.LeftOnly) == 0
}

// IsSuperset returns true if every unique element in Right is also present in Left.
func (r SetResult[T]) IsSuperset() bool {
	return len(r.RightOnly) == 0
}

// IsDisjoint returns true if Left and Right share zero common elements.
func (r SetResult[T]) IsDisjoint() bool {
	return len(r.Intersection) == 0
}

// SetDiff computes the difference, intersection, and disjoint sets of two collections.
// Duplicates within input slices are deduplicated while preserving first-seen order.
func SetDiff[T comparable](left, right []T) SetResult[T] {
	leftSet := make(map[T]struct{}, len(left))
	rightSet := make(map[T]struct{}, len(right))

	for _, v := range right {
		rightSet[v] = struct{}{}
	}

	var leftOnly []T
	var intersection []T

	for _, v := range left {
		if _, seen := leftSet[v]; seen {
			continue
		}
		leftSet[v] = struct{}{}

		if _, inRight := rightSet[v]; inRight {
			intersection = append(intersection, v)
		} else {
			leftOnly = append(leftOnly, v)
		}
	}

	var rightOnly []T
	for _, v := range right {
		if _, inLeft := leftSet[v]; !inLeft {
			if _, inRightOnly := rightSet[v]; inRightOnly {
				rightOnly = append(rightOnly, v)
				delete(rightSet, v) // Avoid duplicate appends to rightOnly.
			}
		}
	}

	return SetResult[T]{
		LeftOnly:     leftOnly,
		RightOnly:    rightOnly,
		Intersection: intersection,
	}
}

// Mismatch captures a value discrepancy for a specific key present in both maps.
type Mismatch[V any] struct {
	Left  V `json:"left"`
	Right V `json:"right"`
}

// MapResult holds the outcome of a key-by-key map comparison.
type MapResult[K comparable, V any] struct {
	Identical  map[K]V
	LeftOnly   map[K]V
	RightOnly  map[K]V
	Mismatched map[K]Mismatch[V]
}

// Match returns true if both maps have identical key-value pairs.
func (r MapResult[K, V]) Match() bool {
	return len(r.LeftOnly) == 0 && len(r.RightOnly) == 0 && len(r.Mismatched) == 0
}

// Summary returns a concise human-readable description of the map comparison result.
func (r MapResult[K, V]) Summary() string {
	return fmt.Sprintf("identical=%d left_only=%d right_only=%d mismatched=%d",
		len(r.Identical), len(r.LeftOnly), len(r.RightOnly), len(r.Mismatched))
}

// MapDiff compares two maps with comparable keys and arbitrary values.
// If equals is nil, standard value assignment equality is used where applicable,
// or callers provide a custom equality predicate.
func MapDiff[K comparable, V any](left, right map[K]V, equals func(V, V) bool) MapResult[K, V] {
	res := MapResult[K, V]{
		Identical:  make(map[K]V),
		LeftOnly:   make(map[K]V),
		RightOnly:  make(map[K]V),
		Mismatched: make(map[K]Mismatch[V]),
	}

	for k, lv := range left {
		rv, inRight := right[k]
		if !inRight {
			res.LeftOnly[k] = lv

			continue
		}

		var matched bool
		if equals != nil {
			matched = equals(lv, rv)
		} else {
			// Fallback: use fmt.Sprint comparison for generic any values if no equals predicate provided.
			matched = fmt.Sprint(lv) == fmt.Sprint(rv)
		}

		if matched {
			res.Identical[k] = lv
		} else {
			res.Mismatched[k] = Mismatch[V]{
				Left:  lv,
				Right: rv,
			}
		}
	}

	for k, rv := range right {
		if _, inLeft := left[k]; !inLeft {
			res.RightOnly[k] = rv
		}
	}

	return res
}

// ThreeWayResult holds the outcome of a 3-angle migration verification:
// comparing pre-migration baseline, post-migration target, and leftover source items.
type ThreeWayResult[T comparable] struct {
	Moved              []T
	Leftovers          []T
	MissingFromTarget  []T
	UnaccountedTarget  []T
}

// Passed returns true if all baseline items were cleanly moved, zero items were left behind,
// zero items went missing, and zero unaccounted items appeared in target.
func (r ThreeWayResult[T]) Passed() bool {
	return len(r.Leftovers) == 0 && len(r.MissingFromTarget) == 0 && len(r.UnaccountedTarget) == 0
}

// ThreeWayDiff implements the mathematical migration equation: baseline ∖ leftovers == target.
//
// Categorizes all elements into:
// - Moved: baseline items verified present in target and absent from leftovers.
// - Leftovers: baseline items still present in source leftovers.
// - MissingFromTarget: baseline items absent from both target and leftovers (lost items).
// - UnaccountedTarget: items in target that were not part of the baseline (ghost items).
func ThreeWayDiff[T comparable](baseline, target, leftovers []T) ThreeWayResult[T] {
	targetSet := make(map[T]struct{}, len(target))
	for _, v := range target {
		targetSet[v] = struct{}{}
	}

	leftoverSet := make(map[T]struct{}, len(leftovers))
	for _, v := range leftovers {
		leftoverSet[v] = struct{}{}
	}

	baselineSet := make(map[T]struct{}, len(baseline))
	var moved []T
	var leftoverItems []T
	var missing []T

	for _, v := range baseline {
		if _, seen := baselineSet[v]; seen {
			continue
		}
		baselineSet[v] = struct{}{}

		_, inLeftovers := leftoverSet[v]
		_, inTarget := targetSet[v]

		switch {
		case inLeftovers:
			leftoverItems = append(leftoverItems, v)
		case inTarget:
			moved = append(moved, v)
		default:
			missing = append(missing, v)
		}
	}

	var unaccounted []T
	seenTarget := make(map[T]struct{}, len(target))
	for _, v := range target {
		if _, seen := seenTarget[v]; seen {
			continue
		}
		seenTarget[v] = struct{}{}

		if _, inBaseline := baselineSet[v]; !inBaseline {
			unaccounted = append(unaccounted, v)
		}
	}

	return ThreeWayResult[T]{
		Moved:             moved,
		Leftovers:         leftoverItems,
		MissingFromTarget: missing,
		UnaccountedTarget: unaccounted,
	}
}
