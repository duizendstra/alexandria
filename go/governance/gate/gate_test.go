package gate_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/duizendstra/alexandria/go/governance/gate"
	"github.com/duizendstra/alexandria/go/governance/invariant"
)

func TestGate_Evaluate_AllPass(t *testing.T) {
	ctx := context.Background()
	g := gate.New("preflight-wave-1",
		gate.WithPolicy(gate.PolicyStandard),
		gate.WithRules(
			gate.NewRule("dwd-scope-check", func(_ context.Context) gate.Result {
				return gate.Result{
					Status:   gate.StatusPass,
					Evidence: []string{"scope https://www.googleapis.com/auth/drive: OK"},
				}
			}),
			gate.NewRule("target-quota-check", func(_ context.Context) gate.Result {
				return gate.Result{
					Status:   gate.StatusPass,
					Evidence: []string{"quota: 400 GB available"},
				}
			}),
		),
	)

	rep := g.Evaluate(ctx)
	if rep.Verdict != gate.VerdictPass {
		t.Fatalf("expected VerdictPass, got %s", rep.Verdict)
	}
	if rep.TotalRules != 2 || rep.PassedRules != 2 {
		t.Fatalf("expected 2 passed rules, got total=%d passed=%d", rep.TotalRules, rep.PassedRules)
	}
	if rep.GateName != "preflight-wave-1" {
		t.Errorf("expected gate name %q, got %q", "preflight-wave-1", rep.GateName)
	}
}

func TestGate_Enforce_Failure(t *testing.T) {
	ctx := context.Background()
	g := gate.New("security-gate",
		gate.WithPolicy(gate.PolicyStandard),
		gate.WithRules(
			gate.NewRule("admin-mfa", func(_ context.Context) gate.Result {
				return gate.Result{
					Status:   gate.StatusFail,
					Reason:   "MFA not enabled on target admin account",
					Evidence: []string{"account admin@example.com has enrolled_mfa=false"},
				}
			}),
		),
	)

	rep, err := g.Enforce(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, gate.ErrGateBlocked) {
		t.Fatalf("expected ErrGateBlocked, got %v", err)
	}
	if rep.Verdict != gate.VerdictBlocked {
		t.Fatalf("expected VerdictBlocked, got %s", rep.Verdict)
	}
	if rep.FailedRules != 1 {
		t.Fatalf("expected 1 failed rule, got %d", rep.FailedRules)
	}

	summary := rep.Summary()
	if !strings.Contains(summary, "FAIL") || !strings.Contains(summary, "admin-mfa") {
		t.Errorf("summary missing expected contents: %s", summary)
	}
}

func TestGate_PolicyBehavior(t *testing.T) {
	ctx := context.Background()

	anomalyRule := gate.NewRule("stale-mapping-check", func(_ context.Context) gate.Result {
		return gate.Result{
			Status:   gate.StatusAnomaly,
			Reason:   "mapping was generated 3 days ago",
			Evidence: []string{"map_version: 2026-08-18T10:00:00Z"},
		}
	})

	t.Run("Standard policy permits anomalies", func(t *testing.T) {
		g := gate.New("standard-gate", gate.WithPolicy(gate.PolicyStandard))
		g.AddRule(anomalyRule)

		rep, err := g.Enforce(ctx)
		if err != nil {
			t.Fatalf("expected pass under standard policy, got error: %v", err)
		}
		if rep.Verdict != gate.VerdictPass {
			t.Fatalf("expected VerdictPass, got %s", rep.Verdict)
		}
		if rep.AnomalyRules != 1 {
			t.Fatalf("expected 1 anomaly rule, got %d", rep.AnomalyRules)
		}
	})

	t.Run("Strict policy blocks anomalies", func(t *testing.T) {
		g := gate.New("strict-gate", gate.WithPolicy(gate.PolicyStrict))
		g.AddRule(anomalyRule)

		rep, err := g.Enforce(ctx)
		if err == nil {
			t.Fatal("expected strict gate to block on anomaly, got nil error")
		}
		if !errors.Is(err, gate.ErrGateBlocked) {
			t.Fatalf("expected ErrGateBlocked, got %v", err)
		}
		if rep.Verdict != gate.VerdictBlocked {
			t.Fatalf("expected VerdictBlocked, got %s", rep.Verdict)
		}
	})

	t.Run("Permissive policy never blocks", func(t *testing.T) {
		g := gate.New("permissive-gate", gate.WithPolicy(gate.PolicyPermissive))
		g.AddRule(gate.NewRule("critical-failure", func(_ context.Context) gate.Result {
			return gate.Result{Status: gate.StatusFail, Reason: "broken"}
		}))

		rep, err := g.Enforce(ctx)
		if err != nil {
			t.Fatalf("expected permissive gate to not return error, got %v", err)
		}
		if rep.Verdict != gate.VerdictPass {
			t.Fatalf("expected VerdictPass, got %s", rep.Verdict)
		}
		if rep.FailedRules != 1 {
			t.Fatalf("expected 1 failed rule recorded, got %d", rep.FailedRules)
		}
	})
}

func TestGate_FromInvariant(t *testing.T) {
	ctx := context.Background()

	b := invariant.New("source-residue", 1)
	b.Notef("inspected source folder %s", "folder-123")
	b.Anomalyf("found 1 unmigrated file %s", "orphan.pdf")

	rule := gate.FromBuilder(b)
	if rule.Name() != "source-residue" {
		t.Errorf("expected name 'source-residue', got %s", rule.Name())
	}

	res := rule.Evaluate(ctx)
	if res.Status != gate.StatusAnomaly {
		t.Fatalf("expected StatusAnomaly, got %s", res.Status)
	}
	if len(res.Evidence) != 2 {
		t.Fatalf("expected 2 evidence entries, got %d", len(res.Evidence))
	}

	// Test nil builder.
	nilRule := gate.FromBuilder(nil)
	nilRes := nilRule.Evaluate(ctx)
	if nilRes.Status != gate.StatusPass {
		t.Fatalf("expected StatusPass for nil builder, got %s", nilRes.Status)
	}
}

func TestGate_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	g := gate.New("cancelled-gate")
	g.AddRules(
		gate.NewRule("rule1", func(_ context.Context) gate.Result {
			return gate.Result{Status: gate.StatusPass}
		}),
	)

	rep := g.Evaluate(ctx)
	if rep.Verdict != gate.VerdictBlocked {
		t.Fatalf("expected VerdictBlocked on cancelled context, got %s", rep.Verdict)
	}
	if rep.FailedRules != 1 {
		t.Fatalf("expected 1 failed rule on cancellation, got %d", rep.FailedRules)
	}
}

func TestGate_JSONSerialization(t *testing.T) {
	g := gate.New("json-gate", gate.WithPolicy(gate.PolicyStandard))
	g.AddRule(gate.NewRule("pass-rule", func(_ context.Context) gate.Result {
		return gate.Result{Status: gate.StatusPass, Evidence: []string{"ok"}}
	}))
	g.AddRule(gate.NewRule("skip-rule", func(_ context.Context) gate.Result {
		return gate.Result{Status: gate.StatusSkipped, Reason: "not applicable"}
	}))

	rep := g.Evaluate(context.Background())
	data, err := json.Marshal(rep)
	if err != nil {
		t.Fatalf("failed to marshal report: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON report: %v", err)
	}

	if parsed["gate_name"] != "json-gate" {
		t.Errorf("expected gate_name 'json-gate', got %v", parsed["gate_name"])
	}
	if parsed["verdict"] != "PASS" {
		t.Errorf("expected verdict 'PASS', got %v", parsed["verdict"])
	}
	skipped, ok := parsed["skipped_rules"].(float64)
	if !ok || int(skipped) != 1 {
		t.Errorf("expected 1 skipped rule, got %v", parsed["skipped_rules"])
	}
}

func FuzzGateEvaluation(f *testing.F) {
	f.Add("gate-a", "rule-1", "PASS", "evidence note", false)
	f.Add("gate-b", "rule-2", "FAIL", "defect observed", true)
	f.Add("gate-c", "rule-3", "ANOMALY", "weird input", false)
	f.Add("gate-d", "rule-4", "SKIPPED", "", true)

	f.Fuzz(func(t *testing.T, gateName, ruleName, statusStr, evidence string, strict bool) {
		if gateName == "" || ruleName == "" {
			return
		}

		policy := gate.PolicyStandard
		if strict {
			policy = gate.PolicyStrict
		}

		g := gate.New(gateName, gate.WithPolicy(policy))
		g.AddRule(gate.NewRule(ruleName, func(_ context.Context) gate.Result {
			var st gate.Status
			switch statusStr {
			case "PASS":
				st = gate.StatusPass
			case "FAIL":
				st = gate.StatusFail
			case "ANOMALY":
				st = gate.StatusAnomaly
			case "SKIPPED":
				st = gate.StatusSkipped
			default:
				st = gate.StatusPass
			}

			var ev []string
			if evidence != "" {
				ev = []string{evidence}
			}

			return gate.Result{
				RuleName: ruleName,
				Status:   st,
				Evidence: ev,
			}
		}))

		rep := g.Evaluate(context.Background())
		if rep.GateName != gateName {
			t.Errorf("expected gate name %q, got %q", gateName, rep.GateName)
		}

		_ = rep.Summary()
		_, _ = rep.MarshalJSON()
	})
}
