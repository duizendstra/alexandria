package observability_test

import (
	"encoding/json"
	"maps"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/observability"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// keySinkExtraLogNames is the config key under test throughout this file.
const keySinkExtraLogNames = "sinkExtraLogNames"

// keyDisplayName and keyStackRef are the uptimeTargets entry keys the
// StackReference tests set.
const (
	keyDisplayName = "displayName"
	keyStackRef    = "stackRef"
)

// sharedStack is the stack two uptime targets reference in the shared-read test.
const sharedStack = "org/frontend/dev"

// frontendTarget is the display name of the uptime target on sharedStack.
const frontendTarget = "frontend"

// requiredConfig is the minimal placement config Apply needs when no
// Params and no governanceStack are supplied.
var requiredConfig = map[string]any{
	"projectName":    "obs-test",
	"folderID":       "folders/123",
	"billingAccount": "000000-000000-000000",
	"orgID":          "123456789012",
}

// sinkFilterMocks captures the Filter passed to whichever resource sets a
// "filter" input — in this graph, only the org log sink does.
type sinkFilterMocks struct {
	filter *string
}

//nolint:gocritic // signature is fixed by pulumi.MockResourceMonitor.
func (m sinkFilterMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outputs := args.Inputs.Copy()
	outputs["number"] = resource.NewStringProperty("123456789")
	outputs["writerIdentity"] = resource.NewStringProperty("serviceAccount:sink-writer@example.iam.gserviceaccount.com")

	if f, ok := args.Inputs["filter"]; ok {
		*m.filter = f.StringValue()
	}

	return args.Name + "-id", outputs, nil
}

func (sinkFilterMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

// setConfig marshals key/value pairs into PULUMI_CONFIG under the "project"
// namespace, matching config.New(ctx, "")'s default namespace when Apply
// runs under pulumi.WithMocks("project", "stack", ...).
func setConfig(t *testing.T, kv map[string]any) {
	t.Helper()

	flat := make(map[string]string, len(kv))
	for k, v := range kv {
		if s, ok := v.(string); ok {
			flat["project:"+k] = s

			continue
		}
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		flat["project:"+k] = string(b)
	}

	blob, err := json.Marshal(flat)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PULUMI_CONFIG", string(blob))
}

// runApply runs observability.Apply(ctx, nil) under mocks and returns the
// filter string the org log sink resource was created with (empty if Apply
// errored before the sink resource was registered) and Apply's error.
func runApply(t *testing.T, extraConfig map[string]any) (string, error) {
	t.Helper()

	cfg := make(map[string]any, len(requiredConfig)+len(extraConfig))
	maps.Copy(cfg, requiredConfig)
	maps.Copy(cfg, extraConfig)
	setConfig(t, cfg)

	var filter string
	mocks := sinkFilterMocks{filter: &filter}

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return observability.Apply(ctx, nil)
	}, pulumi.WithMocks("project", "stack", mocks))

	return filter, err
}

func TestApply_OmittedSinkExtraLogNamesYieldsTodaysSink(t *testing.T) {
	got, err := runApply(t, nil)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	const wantOriginalFilter = `logName:"logs/cloudaudit.googleapis.com"`
	if got != wantOriginalFilter {
		t.Errorf("sink filter with sinkExtraLogNames omitted = %q, want unchanged original %q", got, wantOriginalFilter)
	}
}

func TestApply_CallerSuppliedExtraLogNamesAreAddedToTheDefault(t *testing.T) {
	got, err := runApply(t, map[string]any{
		keySinkExtraLogNames: []string{"one.example.com", "two.example.com"},
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	const want = `logName:"logs/cloudaudit.googleapis.com" OR logName:"logs/one.example.com" OR logName:"logs/two.example.com"`
	if got != want {
		t.Errorf("sink filter with caller-supplied sinkExtraLogNames = %q, want %q", got, want)
	}
}

func TestApply_EmptyExtraLogNameEntryFailsClosed(t *testing.T) {
	_, err := runApply(t, map[string]any{
		keySinkExtraLogNames: []string{""},
	})
	if err == nil {
		t.Fatal("Apply with an empty sinkExtraLogNames entry: want error, got nil")
	}
}

func TestApply_QuoteBreakingExtraLogNameEntryFailsClosed(t *testing.T) {
	_, err := runApply(t, map[string]any{
		keySinkExtraLogNames: []string{`x" OR NOT logName:"nothing`},
	})
	if err == nil {
		t.Fatal("Apply with a quote-breaking sinkExtraLogNames entry: want error, got nil")
	}
}

// TestApply_PercentEncodedQuoteBreakingExtraLogNameEntryFailsClosed uses a
// payload built entirely from the allowed charset (letters, digits, %) so
// it passes the raw check, but percent-decodes to `" OR severity>=DEFAULT`.
// This only fails if the decoded form is validated too — a fail-closed
// guarantee that must hold regardless of whether any downstream layer
// (the Pulumi provider, the Cloud Logging API) ever decodes the filter.
func TestApply_PercentEncodedQuoteBreakingExtraLogNameEntryFailsClosed(t *testing.T) {
	_, err := runApply(t, map[string]any{
		keySinkExtraLogNames: []string{"%22%20OR%20severity%3E%3DDEFAULT"},
	})
	if err == nil {
		t.Fatal("Apply with a percent-encoded quote-breaking sinkExtraLogNames entry: want error, got nil")
	}
}

// stackRefMocks extends sinkFilterMocks with a countable StackReference
// read: every "pulumi:pulumi:StackReference" registration is tallied, and
// the read answers with a stack output map carrying the default URL key so
// the uptime target's host derivation has a value to transform.
type stackRefMocks struct {
	sinkFilterMocks
	stackRefs *int
}

//nolint:gocritic // signature is fixed by pulumi.MockResourceMonitor.
func (m stackRefMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if args.TypeToken != "pulumi:pulumi:StackReference" {
		return m.sinkFilterMocks.NewResource(args)
	}

	*m.stackRefs++
	outputs := args.Inputs.Copy()
	outputs["outputs"] = resource.NewObjectProperty(resource.PropertyMap{
		"frontendUrl": resource.NewStringProperty("https://app.example.com/"),
	})

	return args.Name, outputs, nil
}

// runApplyCountingStackRefs runs observability.Apply(ctx, nil) under mocks
// with the given uptime targets plus any extra config and returns how many
// StackReference resources the program registered, plus Apply's error.
func runApplyCountingStackRefs(t *testing.T, targets []map[string]any, extraConfig map[string]any) (int, error) {
	t.Helper()

	cfg := make(map[string]any, len(requiredConfig)+len(extraConfig)+1)
	maps.Copy(cfg, requiredConfig)
	maps.Copy(cfg, extraConfig)
	cfg["uptimeTargets"] = targets
	setConfig(t, cfg)

	var (
		filter    string
		stackRefs int
	)
	mocks := stackRefMocks{
		sinkFilterMocks: sinkFilterMocks{filter: &filter},
		stackRefs:       &stackRefs,
	}

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return observability.Apply(ctx, nil)
	}, pulumi.WithMocks("project", "stack", mocks))

	return stackRefs, err
}

func TestApply_SingleUptimeTargetReadsItsStackOnce(t *testing.T) {
	got, err := runApplyCountingStackRefs(t, []map[string]any{
		{keyDisplayName: frontendTarget, keyStackRef: sharedStack},
	}, nil)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got != 1 {
		t.Errorf("StackReference registrations for one uptime target = %d, want 1", got)
	}
}

// TestApply_UptimeTargetsSharingAStackReadItOnce guards the URN-uniqueness
// rule the SDK mocks do not enforce: a StackReference's logical name is the
// stack name, so two targets on the same stack must share one read — a
// second registration under the same name makes the real engine abort the
// whole update with a duplicate URN.
func TestApply_UptimeTargetsSharingAStackReadItOnce(t *testing.T) {
	got, err := runApplyCountingStackRefs(t, []map[string]any{
		{keyDisplayName: frontendTarget, keyStackRef: sharedStack},
		{keyDisplayName: "frontend api", keyStackRef: sharedStack, "urlOutputKey": "frontendUrl"},
		{keyDisplayName: "backend", keyStackRef: "org/backend/dev"},
	}, nil)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got != 2 {
		t.Errorf("StackReference registrations for three targets over two stacks = %d, want 2 (one per distinct stack)", got)
	}
}

// TestApply_GovernanceStackUsedAsUptimeTargetIsReadOnce extends the
// URN-uniqueness guard to the governance stack: placement resolution and
// an uptime target that name the same stack must share the one
// StackReference, otherwise the governance read and the target's read
// register the same name twice. The mock answers the governance read with
// no placement outputs, so placement falls back to the required config.
func TestApply_GovernanceStackUsedAsUptimeTargetIsReadOnce(t *testing.T) {
	got, err := runApplyCountingStackRefs(t, []map[string]any{
		{keyDisplayName: frontendTarget, keyStackRef: sharedStack},
	}, map[string]any{
		"governanceStack": sharedStack,
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got != 1 {
		t.Errorf("StackReference registrations for the governance stack also used as an uptime target = %d, want 1 (shared)", got)
	}
}
