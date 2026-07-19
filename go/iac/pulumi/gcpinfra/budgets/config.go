package budgets

import "errors"

var (
	// ErrDisplayNameRequired means the budget has no display name.
	ErrDisplayNameRequired = errors.New("budgets: DisplayName is required")
	// ErrAmountNotPositive means the budget amount is zero or negative.
	ErrAmountNotPositive = errors.New("budgets: Amount must be positive")
	// ErrBillingAccountRequired means the budget has no billing account.
	ErrBillingAccountRequired = errors.New("budgets: BillingAccount is required")
	// ErrScopeRequired means the budget has no resource ancestor scope.
	ErrScopeRequired = errors.New("budgets: Scope is required")
	// ErrThresholdsRequired means the budget has no alert thresholds.
	ErrThresholdsRequired = errors.New("budgets: at least one threshold is required")
	// ErrAlertEmailsRequired means the budget has no notification recipients.
	ErrAlertEmailsRequired = errors.New("budgets: at least one alert email is required")
)

// Config defines a budget with alert thresholds.
type Config struct {
	// DisplayName is the budget display name.
	DisplayName string `json:"displayName"`
	// Amount is the monthly budget amount in units.
	Amount int `json:"amount"`
	// Currency is the ISO 4217 currency code (default: "USD").
	Currency string `json:"currency"`
	// BillingAccount is the billing account ID.
	BillingAccount string `json:"billingAccount"`
	// Scope is the resource ancestor for the budget filter.
	// Supports both org ("organizations/123") and folder ("folders/456").
	Scope string `json:"scope"`
	// Thresholds are the alert percentages (e.g. 0.50, 0.75, 0.90, 1.00).
	Thresholds []float64 `json:"thresholds"`
	// AlertEmails are the notification recipients.
	AlertEmails []string `json:"alertEmails"`
}

// Validate checks that the budget configuration is complete.
func (c *Config) Validate() error {
	if c.DisplayName == "" {
		return ErrDisplayNameRequired
	}
	if c.Amount <= 0 {
		return ErrAmountNotPositive
	}
	if c.BillingAccount == "" {
		return ErrBillingAccountRequired
	}
	if c.Scope == "" {
		return ErrScopeRequired
	}
	if len(c.Thresholds) == 0 {
		return ErrThresholdsRequired
	}
	if len(c.AlertEmails) == 0 {
		return ErrAlertEmailsRequired
	}

	return nil
}
