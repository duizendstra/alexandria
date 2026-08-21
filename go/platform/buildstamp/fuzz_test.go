package buildstamp

import (
	"testing"
)

func FuzzParseStamp(f *testing.F) {
	// Seed corpus.
	f.Add("tool 1.4.0 commit=1111111111111111111111111111111111111111 dirty=false built=2026-01-02T15:04:05Z lib=abc1234")
	f.Add("commit=2222222222222222222222222222222222222222 dirty=true")
	f.Add("app commit=unknown dirty=false")
	f.Add("malformed line without equals")
	f.Add("")
	f.Add("tool commit=1111111111111111111111111111111111111111 dirty=maybe")

	f.Fuzz(func(t *testing.T, line string) {
		stamp, err := ParseStamp(line)
		if err != nil {
			return // Expected rejection of malformed input.
		}

		// If parse succeeded, verify String() doesn't panic and produces a parseable string.
		rendered := stamp.String()
		if rendered == "" {
			t.Errorf("rendered string is empty for parsed stamp %+v", stamp)
		}

		// Round-trip check: parsing rendered string should not panic.
		roundTrip, err := ParseStamp(rendered)
		if err != nil {
			t.Fatalf("failed to parse back rendered stamp %q: %v", rendered, err)
		}

		if roundTrip.Commit != stamp.Commit || roundTrip.Dirty != stamp.Dirty {
			t.Errorf("round trip mismatch: got commit=%s dirty=%t, want commit=%s dirty=%t",
				roundTrip.Commit, roundTrip.Dirty, stamp.Commit, stamp.Dirty)
		}
	})
}

func FuzzMatches(f *testing.F) {
	f.Add("1111111111111111111111111111111111111111", "1111111111111111111111111111111111111111", false)
	f.Add("1111111111111111111111111111111111111111", "2222222222222222222222222222222222222222", false)
	f.Add("unknown", "1111111111111111111111111111111111111111", false)
	f.Add("short", "1111111111111111111111111111111111111111", true)

	f.Fuzz(func(t *testing.T, actualCommit, expectedCommit string, dirty bool) {
		s := Stamp{
			Commit: actualCommit,
			Dirty:  dirty,
		}
		// Matches should never panic regardless of input.
		_ = s.Matches(expectedCommit)
	})
}
