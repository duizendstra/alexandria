package finops

import (
	"maps"
	"strings"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// typeBudget is the resource the thresholds config ends up on.
const typeBudget = "gcp:billing/budget:Budget"

// requiredConfig is the minimal config Apply needs with no Params and no
// governance stack: placement plus the budget's required keys.
var requiredConfig = map[string]any{
	"projectName":    "finops-test",
	"folderID":       "folders/123",
	"billingAccount": "000000-000000-000000",
	"orgID":          "123456",
	"monthlyBudget":  1000,
	"alertEmails":    []string{"alerts@example.com"},
}

// runApply runs Apply(ctx, nil) under mocks with requiredConfig plus
// extraConfig.
func runApply(t *testing.T, extraConfig map[string]any) (recordingMocks, error) {
	t.Helper()

	cfg := make(map[string]any, len(requiredConfig)+len(extraConfig))
	maps.Copy(cfg, requiredConfig)
	maps.Copy(cfg, extraConfig)
	setConfig(t, cfg)

	mocks := newRecordingMocks()
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return Apply(ctx, nil)
	}, pulumi.WithMocks("project", "stack", mocks))

	return mocks, err
}

// budgetThresholds returns the threshold percentages of the single
// budget the program registered.
func budgetThresholds(t *testing.T, mocks recordingMocks) []float64 {
	t.Helper()

	budgets := mocks.registered(typeBudget)
	if len(budgets) != 1 {
		t.Fatalf("budgets registered = %d, want 1", len(budgets))
	}

	rules := budgets[0]["thresholdRules"].ArrayValue()
	out := make([]float64, 0, len(rules))
	for _, rule := range rules {
		out = append(out, rule.ObjectValue()["thresholdPercent"].NumberValue())
	}

	return out
}

func assertThresholds(t *testing.T, got, want []float64) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("thresholds = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("thresholds = %v, want %v", got, want)
		}
	}
}

func TestApply_AbsentThresholdsUseDefaults(t *testing.T) {
	mocks, err := runApply(t, nil)
	if err != nil {
		t.Fatalf("Apply with thresholds absent: %v", err)
	}

	assertThresholds(t, budgetThresholds(t, mocks), []float64{0.50, 0.75, 0.90, 1.00})
}

func TestApply_EmptyThresholdsUseDefaults(t *testing.T) {
	mocks, err := runApply(t, map[string]any{keyThresholds: []float64{}})
	if err != nil {
		t.Fatalf("Apply with thresholds empty: %v", err)
	}

	assertThresholds(t, budgetThresholds(t, mocks), []float64{0.50, 0.75, 0.90, 1.00})
}

func TestApply_CustomThresholdsOverrideDefaults(t *testing.T) {
	mocks, err := runApply(t, map[string]any{keyThresholds: []float64{0.8, 1.0, 1.2}})
	if err != nil {
		t.Fatalf("Apply with custom thresholds: %v", err)
	}

	assertThresholds(t, budgetThresholds(t, mocks), []float64{0.8, 1.0, 1.2})
}

// A thresholds value that is present but does not parse must abort the
// update: read as absent it would provision the alerts on the default
// thresholds without telling the operator their override was ignored.
func TestApply_MalformedThresholdsFails(t *testing.T) {
	mocks, err := runApply(t, map[string]any{keyThresholds: malformedThresholds})
	if err == nil {
		t.Fatal("Apply with malformed thresholds: want error, got nil")
	}
	if want := `config key "` + keyThresholds + `"`; !strings.Contains(err.Error(), want) {
		t.Fatalf("error %v does not contain %q", err, want)
	}
	if n := len(mocks.registered(typeBudget)); n != 0 {
		t.Fatalf("budgets registered after malformed thresholds = %d, want 0", n)
	}
}
