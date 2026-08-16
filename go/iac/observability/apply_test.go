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
