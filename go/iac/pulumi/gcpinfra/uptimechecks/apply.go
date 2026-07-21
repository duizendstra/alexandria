package uptimechecks

import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/monitoring"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Detection window for the failure alert. Failed probes are counted over
// alertAlignmentPeriod (GCP's canonical uptime window) and the alert fires once
// that count stays above threshold for alertDuration. Fixed for v1 — a
// configurable window can be added if a consumer needs one.
const (
	alertAlignmentPeriod = "1200s"
	alertDuration        = "60s"
)

// Apply creates an HTTPS UptimeCheckConfig for cfg.Host and an AlertPolicy that
// fires — routing to the caller-supplied channelIDs — when the check reports
// failures over the detection window. Notification channels are provided by the
// caller (e.g. reusing the ones budgets creates) rather than created here.
func Apply(
	ctx *pulumi.Context,
	projectID pulumi.StringOutput,
	cfg *Config,
	channelIDs pulumi.StringArrayInput,
	deps []pulumi.Resource,
) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	r := cfg.resolved()

	var statusCodes monitoring.UptimeCheckConfigHttpCheckAcceptedResponseStatusCodeArray
	for _, sc := range r.statusClasses {
		statusCodes = append(statusCodes, &monitoring.UptimeCheckConfigHttpCheckAcceptedResponseStatusCodeArgs{
			StatusClass: pulumi.String(sc),
		})
	}

	check, err := monitoring.NewUptimeCheckConfig(ctx, r.displayName, &monitoring.UptimeCheckConfigArgs{
		Project:     projectID,
		DisplayName: pulumi.String(r.displayName),
		Timeout:     pulumi.String(r.timeout),
		Period:      pulumi.String(r.period),
		HttpCheck: &monitoring.UptimeCheckConfigHttpCheckArgs{
			Path:                        pulumi.String(r.path),
			Port:                        pulumi.Int(r.port),
			UseSsl:                      pulumi.Bool(true),
			ValidateSsl:                 pulumi.Bool(true),
			RequestMethod:               pulumi.String("GET"),
			AcceptedResponseStatusCodes: statusCodes,
		},
		MonitoredResource: &monitoring.UptimeCheckConfigMonitoredResourceArgs{
			Type: pulumi.String("uptime_url"),
			Labels: pulumi.StringMap{
				"project_id": projectID,
				"host":       pulumi.String(r.host),
			},
		},
	}, pulumi.DependsOn(deps))
	if err != nil {
		return fmt.Errorf("create uptime check %s: %w", r.displayName, err)
	}

	// Alert when the check reports failures over the detection window.
	filter := pulumi.Sprintf(
		`metric.type="monitoring.googleapis.com/uptime_check/check_passed" `+
			`AND resource.type="uptime_url" AND metric.labels.check_id="%s"`,
		check.UptimeCheckId,
	)

	_, err = monitoring.NewAlertPolicy(ctx, r.displayName+"-failing", &monitoring.AlertPolicyArgs{
		Project:              projectID,
		DisplayName:          pulumi.Sprintf("%s failing", r.displayName),
		Combiner:             pulumi.String("OR"),
		NotificationChannels: channelIDs,
		Conditions: monitoring.AlertPolicyConditionArray{
			&monitoring.AlertPolicyConditionArgs{
				DisplayName: pulumi.Sprintf("%s check failing", r.displayName),
				ConditionThreshold: &monitoring.AlertPolicyConditionConditionThresholdArgs{
					Filter:         filter,
					Comparison:     pulumi.String("COMPARISON_GT"),
					ThresholdValue: pulumi.Float64(1),
					Duration:       pulumi.String(alertDuration),
					Aggregations: monitoring.AlertPolicyConditionConditionThresholdAggregationArray{
						&monitoring.AlertPolicyConditionConditionThresholdAggregationArgs{
							AlignmentPeriod:    pulumi.String(alertAlignmentPeriod),
							PerSeriesAligner:   pulumi.String("ALIGN_NEXT_OLDER"),
							CrossSeriesReducer: pulumi.String("REDUCE_COUNT_FALSE"),
							GroupByFields:      pulumi.StringArray{pulumi.String("resource.label.host")},
						},
					},
					Trigger: &monitoring.AlertPolicyConditionConditionThresholdTriggerArgs{
						Count: pulumi.Int(1),
					},
				},
			},
		},
	}, pulumi.DependsOn([]pulumi.Resource{check}))
	if err != nil {
		return fmt.Errorf("create alert policy for %s: %w", r.displayName, err)
	}

	return nil
}
