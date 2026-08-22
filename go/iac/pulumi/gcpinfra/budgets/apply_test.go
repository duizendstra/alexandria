package budgets_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/budgets"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type mocks int

func (mocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) { //nolint:gocritic // hugeParam: interface-fixed signature
	return args.Name + "_id", args.Inputs, nil
}

func (mocks) Call(_ pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return resource.PropertyMap{}, nil
}

const alertEmail = "ops@example.com"

func validConfig() *budgets.Config {
	return &budgets.Config{
		DisplayName:    "Org Monthly",
		Amount:         100,
		BillingAccount: "XXX",
		Scope:          "organizations/123",
		Thresholds:     []float64{0.50, 1.00},
		AlertEmails:    []string{alertEmail, "finance@example.com"},
	}
}

func TestApplyCreates(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return budgets.Apply(ctx, pulumi.String("proj").ToStringOutput(), validConfig(), nil)
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}

func TestApplyInvalidConfig(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		err := budgets.Apply(ctx, pulumi.String("proj").ToStringOutput(), &budgets.Config{}, nil)
		if !errors.Is(err, budgets.ErrDisplayNameRequired) {
			t.Errorf("expected ErrDisplayNameRequired, got %v", err)
		}

		return nil
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}

// TestApplyDuplicateAlertEmail pins #248: each alert email becomes the Pulumi
// logical name of its notification channel, so a repeated email is one URN.
// The SDK mocks let the repeat through, so Apply must reject it itself.
func TestApplyDuplicateAlertEmail(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		cfg := validConfig()
		cfg.AlertEmails = append(cfg.AlertEmails, alertEmail)

		err := budgets.Apply(ctx, pulumi.String("proj").ToStringOutput(), cfg, nil)
		if !errors.Is(err, budgets.ErrDuplicateAlertEmail) {
			t.Errorf("expected ErrDuplicateAlertEmail, got %v", err)
		}
		if err == nil || !strings.Contains(err.Error(), `"`+alertEmail+`"`) {
			t.Errorf("error should name the duplicate %q, got %v", alertEmail, err)
		}

		return nil
	}, pulumi.WithMocks("example", "stack", mocks(0)))
	if err != nil {
		t.Fatalf("pulumi run: %v", err)
	}
}
