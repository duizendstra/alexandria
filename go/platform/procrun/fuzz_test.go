package procrun

import (
	"strings"
	"testing"
)

func FuzzEnvironScrubbing(f *testing.F) {
	// Seed corpus: runner env key, runner env val, call env key, call env val, scrub prefix.
	f.Add("GCP_PROJECT", "proj-1", "LOCAL_VAR", "val-1", "GCP_")
	f.Add("FOO", "bar", "FOO", "override", "FOO")
	f.Add("EMPTY", "", "", "", "SECRET_")
	f.Add("A", "1", "B", "2", "XYZ_")

	f.Fuzz(func(t *testing.T, rKey, rVal, cKey, cVal, scrubPrefix string) {
		// Clean test keys to valid env var names.
		if strings.Contains(rKey, "=") || strings.Contains(rKey, "\x00") || rKey == "" {
			return
		}
		if strings.Contains(cKey, "=") || strings.Contains(cKey, "\x00") || cKey == "" {
			return
		}

		r := Runner{
			Env: map[string]string{
				rKey: rVal,
			},
			Scrub: []string{scrubPrefix},
		}

		callEnv := map[string]string{
			cKey: cVal,
		}

		env := r.Environ(callEnv)
		if env == nil {
			t.Fatalf("Environ returned nil")
		}

		// Verify every entry has valid format (key=value).
		seen := make(map[string]bool)
		for _, entry := range env {
			k, _, found := strings.Cut(entry, "=")
			if !found {
				t.Errorf("malformed env entry: %q", entry)
			}
			seen[k] = true
		}

		// Fixed runner and call env keys must always be present.
		if !seen[rKey] {
			t.Errorf("runner env key %q missing from environ", rKey)
		}
		if !seen[cKey] {
			t.Errorf("call env key %q missing from environ", cKey)
		}
	})
}
