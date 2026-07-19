package budgets

import (
	"fmt"
	"strconv"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/billing"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/monitoring"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Apply creates a billing budget with notification channels and alert thresholds.
func Apply(ctx *pulumi.Context, projectID pulumi.StringOutput, cfg *Config, deps []pulumi.Resource) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	currency := cfg.Currency
	if currency == "" {
		currency = "USD"
	}

	// Create notification channels.
	var channelIDs pulumi.StringArray
	for _, email := range cfg.AlertEmails {
		channel, err := monitoring.NewNotificationChannel(ctx, "alert-"+email, &monitoring.NotificationChannelArgs{
			Project:     projectID,
			DisplayName: pulumi.Sprintf("Budget alert: %s", email),
			Type:        pulumi.String("email"),
			Labels: pulumi.StringMap{
				"email_address": pulumi.String(email),
			},
		}, pulumi.DependsOn(deps))
		if err != nil {
			return fmt.Errorf("create notification channel for %s: %w", email, err)
		}
		channelIDs = append(channelIDs, channel.ID())
	}

	// Create threshold rules.
	var rules billing.BudgetThresholdRuleArray
	for _, t := range cfg.Thresholds {
		rules = append(rules, &billing.BudgetThresholdRuleArgs{
			ThresholdPercent: pulumi.Float64(t),
		})
	}

	// Create budget.
	_, err := billing.NewBudget(ctx, cfg.DisplayName, &billing.BudgetArgs{
		BillingAccount: pulumi.String(cfg.BillingAccount),
		DisplayName:    pulumi.String(cfg.DisplayName),
		Amount: &billing.BudgetAmountArgs{
			SpecifiedAmount: &billing.BudgetAmountSpecifiedAmountArgs{
				CurrencyCode: pulumi.String(currency),
				Units:        pulumi.String(strconv.Itoa(cfg.Amount)),
			},
		},
		ThresholdRules: rules,
		AllUpdatesRule: &billing.BudgetAllUpdatesRuleArgs{
			MonitoringNotificationChannels: channelIDs,
			DisableDefaultIamRecipients:    pulumi.Bool(true),
		},
		BudgetFilter: &billing.BudgetBudgetFilterArgs{
			ResourceAncestors: pulumi.StringArray{
				pulumi.String(cfg.Scope),
			},
		},
	}, pulumi.DependsOn(deps))
	if err != nil {
		return fmt.Errorf("create budget %s: %w", cfg.DisplayName, err)
	}

	return nil
}
