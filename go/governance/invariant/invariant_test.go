package invariant

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestBuilderStartsAtPass(t *testing.T) {
	c := New("empty-rule", 1).Done()
	if c.Status != Pass {
		t.Errorf("a rule that observed nothing is Pass, got %s", c.Status)
	}
	if c.Name != "empty-rule" || c.Invariant != 1 {
		t.Errorf("identity not carried: %+v", c)
	}
	if len(c.Evidence) != 0 {
		t.Errorf("no evidence expected, got %v", c.Evidence)
	}
}

func TestNotefKeepsStatus(t *testing.T) {
	b := New("counting", 2)
	b.Notef("compared %d items", 7)
	c := b.Done()
	if c.Status != Pass {
		t.Errorf("a note must not change the status, got %s", c.Status)
	}
	if len(c.Evidence) != 1 || c.Evidence[0] != "compared 7 items" {
		t.Errorf("note not recorded verbatim: %v", c.Evidence)
	}
}

func TestFailIsPermanent(t *testing.T) {
	b := New("degrading", 3)
	b.Failf("missing %s", "id-1")
	b.Anomalyf("could not read %s", "id-2")
	b.Notef("checked the rest")
	c := b.Done()
	if c.Status != Fail {
		t.Errorf("an anomaly must not downgrade a failure, got %s", c.Status)
	}
	want := []string{"FAIL: missing id-1", "ANOMALY: could not read id-2", "checked the rest"}
	if !reflect.DeepEqual(c.Evidence, want) {
		t.Errorf("evidence order/prefixes wrong:\n got %v\nwant %v", c.Evidence, want)
	}
}

func TestAnomalyRaisesPassOnly(t *testing.T) {
	b := New("unsure", 4)
	b.Anomalyf("unreadable input")
	if got := b.Status(); got != Anomaly {
		t.Errorf("Pass must rise to Anomaly, got %s", got)
	}
	b.Failf("and then a real defect")
	if got := b.Status(); got != Fail {
		t.Errorf("a failure must win, got %s", got)
	}
	b.Anomalyf("another anomaly")
	if got := b.Done().Status; got != Fail {
		t.Errorf("a later anomaly must not reset the failure, got %s", got)
	}
}

func TestWithLabelsKeepsExistingWording(t *testing.T) {
	// A suite that already publishes evidence lines cannot silently change
	// their wording: downstream reports are compared byte for byte.
	b := New("localised", 5, WithLabels(Labels{Fail: "FOUT: ", Anomaly: "AFWIJKING: "}))
	b.Failf("value %d out of range", 42)
	b.Anomalyf("skipped")
	c := b.Done()
	want := []string{"FOUT: value 42 out of range", "AFWIJKING: skipped"}
	if !reflect.DeepEqual(c.Evidence, want) {
		t.Errorf("custom labels not applied:\n got %v\nwant %v", c.Evidence, want)
	}
}

func TestVerdictTakesTheWorst(t *testing.T) {
	cases := []struct {
		name   string
		in     []Status
		expect Status
	}{
		{"empty", nil, Pass},
		{"all pass", []Status{Pass, Pass}, Pass},
		{"one anomaly", []Status{Pass, Anomaly, Pass}, Anomaly},
		{"one failure", []Status{Pass, Fail, Pass}, Fail},
		{"failure outranks anomaly", []Status{Anomaly, Fail}, Fail},
		{"anomaly after failure", []Status{Fail, Anomaly}, Fail},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			checks := make([]Check, 0, len(tc.in))
			for i, s := range tc.in {
				checks = append(checks, Check{Name: string(rune('a' + i)), Status: s})
			}
			if got := Verdict(checks); got != tc.expect {
				t.Errorf("Verdict = %s, want %s", got, tc.expect)
			}
		})
	}
}

func TestFailedAndAnomalousAreSorted(t *testing.T) {
	checks := []Check{
		{Name: "zulu", Status: Fail},
		{Name: "alpha", Status: Fail},
		{Name: "mike", Status: Anomaly},
		{Name: "bravo", Status: Pass},
	}
	if got, want := Failed(checks), []string{"alpha", "zulu"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Failed = %v, want %v", got, want)
	}
	if got, want := Anomalous(checks), []string{"mike"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Anomalous = %v, want %v", got, want)
	}
}

func TestEvaluateRunsRulesInOrder(t *testing.T) {
	type subject struct{ n int }
	rules := []Rule[subject]{
		{"first", 1, func(s subject) Check {
			b := New("first", 1)
			b.Notef("n=%d", s.n)
			return b.Done()
		}},
		{"second", 2, func(s subject) Check {
			b := New("second", 2)
			if s.n < 10 {
				b.Failf("n too small: %d", s.n)
			}
			return b.Done()
		}},
	}
	checks := Evaluate(subject{n: 3}, rules)
	if len(checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(checks))
	}
	if checks[0].Name != "first" || checks[1].Name != "second" {
		t.Errorf("rule order not preserved: %v", checks)
	}
	if Verdict(checks) != Fail {
		t.Errorf("verdict should be Fail, got %s", Verdict(checks))
	}
}

func TestCheckMarshalsAsPlainStrings(t *testing.T) {
	// Status is a named type; it must still serialise as a bare string so
	// existing report consumers keep working.
	b := New("serialised", 6)
	b.Failf("nope")
	raw, err := json.Marshal(b.Done())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"name":"serialised","invariant":6,"status":"FAIL","evidence":["FAIL: nope"]}`
	if string(raw) != want {
		t.Errorf("JSON shape changed:\n got %s\nwant %s", raw, want)
	}
}
