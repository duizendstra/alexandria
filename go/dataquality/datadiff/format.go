package datadiff

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

var ErrInvalidTarget = errors.New("expected project.dataset.table")

// ParseTarget parses a "project.dataset.table" string into a Target.
func ParseTarget(s string) (Target, error) {
	const requiredParts = 3
	parts := strings.SplitN(s, ".", requiredParts)

	if len(parts) != requiredParts {
		return Target{}, fmt.Errorf("%w: got %q", ErrInvalidTarget, s)
	}

	return Target{
		Project: parts[0],
		Dataset: parts[1],
		Table:   parts[2],
	}, nil
}

// ParseTargetPair parses left and right table strings into Targets.
//nolint:gocritic // gocritic unnamedResult clashes with nonamedreturns linter
func ParseTargetPair(left, right string) (Target, Target, error) {
	l, err := ParseTarget(left)
	if err != nil {
		return Target{}, Target{}, fmt.Errorf("left: %w", err)
	}

	r, err := ParseTarget(right)
	if err != nil {
		return Target{}, Target{}, fmt.Errorf("right: %w", err)
	}

	return l, r, nil
}

// BillingProject returns the explicit project if set, or falls back to
// GOOGLE_CLOUD_PROJECT env var, or the target's project.
func BillingProject(explicit string, fallback Target) string {
	if explicit != "" {
		return explicit
	}

	if p := os.Getenv("GOOGLE_CLOUD_PROJECT"); p != "" {
		return p
	}

	return fallback.Project
}

// PrintResult writes a human-readable comparison report to stdout.
func PrintResult(r *Result) {
	if r.Schema.Match {
		fmt.Println("✅ Schema: MATCH")
	} else {
		fmt.Println("❌ Schema: MISMATCH")
		for _, c := range r.Schema.LeftOnly {
			fmt.Printf("   left-only:  %s (%s)\n", c.Name, c.DataType)
		}
		for _, c := range r.Schema.RightOnly {
			fmt.Printf("   right-only: %s (%s)\n", c.Name, c.DataType)
		}
		for _, d := range r.Schema.TypeDiffs {
			fmt.Printf("   type-diff:  %s (%s → %s)\n", d.Name, d.LeftType, d.RightType)
		}
	}

	if r.Volume.Match {
		fmt.Printf("✅ Volume: MATCH (%d rows)\n", r.Volume.LeftCount)
	} else {
		fmt.Printf("❌ Volume: MISMATCH (left=%d, right=%d, delta=%d)\n",
			r.Volume.LeftCount, r.Volume.RightCount, r.Volume.RightCount-r.Volume.LeftCount)
	}

	if r.Content.Match {
		fmt.Printf("✅ Content: MATCH (%d rows identical)\n", r.Content.Matched)
	} else {
		fmt.Printf("❌ Content: MISMATCH (matched=%d, left-only=%d, right-only=%d, differed=%d)\n",
			r.Content.Matched, r.Content.LeftOnly, r.Content.RightOnly, r.Content.Differed)
		for _, d := range r.Content.Diffs {
			fmt.Printf("   %s: key=%s\n", d.Side, d.Key)
		}
	}

	if r.Stats.Match {
		fmt.Println("✅ Stats: MATCH")
	} else {
		fmt.Println("❌ Stats: MISMATCH")
		for _, d := range r.Stats.Diffs {
			fmt.Printf("   %s.%s: left=%.4f right=%.4f delta=%.4f\n",
				d.Column, d.Field, d.Left, d.Right, d.Delta)
		}
	}

	fmt.Println()
	if r.Pass() {
		fmt.Println("🎉 ALL LAYERS PASS")
	} else {
		fmt.Println("⚠️  COMPARISON FAILED")
	}
}

// PrintCost writes bytes processed and estimated cost to stdout.
func PrintCost(bytes int64) {
	const (
		bytesPerMB = 1e6
		bytesPerTB = 1e12
		costPerTB  = 6.25
	)
	mb := float64(bytes) / bytesPerMB
	cost := float64(bytes) / bytesPerTB * costPerTB
	fmt.Printf("📊 Bytes processed: %.2f MB ($%.4f on-demand)\n", mb, cost)
}
