package invariant

import (
	"testing"
)

func FuzzBuilderMonotonicity(f *testing.F) {
	// Seed corpus: sequences of operations (0 = Note, 1 = Anomaly, 2 = Fail).
	f.Add("name1", 1, []byte{0, 0, 1, 0, 2, 1, 0})
	f.Add("rule-x", 0, []byte{2, 1, 0})
	f.Add("empty", 10, []byte{})
	f.Add("anom-only", 5, []byte{1, 1, 1})

	f.Fuzz(func(t *testing.T, name string, inv int, ops []byte) {
		b := New(name, inv)
		if b.Status() != Pass {
			t.Fatalf("initial status not Pass: got %v", b.Status())
		}

		hasAnomaly := false
		hasFail := false

		for i, op := range ops {
			switch op % 3 {
			case 0:
				b.Notef("note %d", i)
			case 1:
				b.Anomalyf("anomaly %d", i)
				hasAnomaly = true
			case 2:
				b.Failf("fail %d", i)
				hasFail = true
			}

			// Invariant: Fail is irreversible.
			if hasFail && b.Status() != Fail {
				t.Errorf("status degraded from Fail to %v after op %d", b.Status(), op)
			}
			// Invariant: Anomaly never degrades back to Pass.
			if hasAnomaly && !hasFail && b.Status() != Anomaly {
				t.Errorf("status degraded from Anomaly to %v after op %d", b.Status(), op)
			}
		}

		chk := b.Done()
		if len(chk.Evidence) != len(ops) {
			t.Errorf("evidence count mismatch: got %d, want %d", len(chk.Evidence), len(ops))
		}
	})
}

func FuzzVerdict(f *testing.F) {
	f.Add([]byte{0, 0, 0})
	f.Add([]byte{0, 1, 0})
	f.Add([]byte{0, 1, 2})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, statuses []byte) {
		checks := make([]Check, len(statuses))
		hasFail := false
		hasAnomaly := false

		for i, s := range statuses {
			switch s % 3 {
			case 0:
				checks[i] = Check{Name: "c", Status: Pass}
			case 1:
				checks[i] = Check{Name: "c", Status: Anomaly}
				hasAnomaly = true
			case 2:
				checks[i] = Check{Name: "c", Status: Fail}
				hasFail = true
			}
		}

		v := Verdict(checks)
		switch {
		case hasFail && v != Fail:
			t.Errorf("Verdict failed to report Fail when present, got %v", v)
		case !hasFail && hasAnomaly && v != Anomaly:
			t.Errorf("Verdict failed to report Anomaly when present without Fail, got %v", v)
		case !hasFail && !hasAnomaly && v != Pass:
			t.Errorf("Verdict failed to report Pass when no failures/anomalies, got %v", v)
		}
	})
}
