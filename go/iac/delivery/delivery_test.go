package delivery_test

import (
	"encoding/json"
	"maps"
	"sync"
	"testing"

	delivery "github.com/duizendstra/alexandria/go/iac/delivery"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Shared fixture values.
const (
	projectNumber = "123456789012"
	writerRole    = "roles/artifactregistry.writer"

	legacyBuildMember  = "serviceAccount:" + projectNumber + "@cloudbuild.gserviceaccount.com"
	computeBuildMember = "serviceAccount:" + projectNumber + "-compute@developer.gserviceaccount.com"

	legacyGrantName  = "build-ar-writer"
	computeGrantName = "build-ar-writer-compute"
)

// applyMocks answers every resource with its inputs (plus a project
// number) and records each Artifact Registry writer grant by logical
// name so a test can assert which identities the blueprint granted.
type applyMocks struct {
	mu      sync.Mutex
	writers map[string]string
}

//nolint:gocritic // signature is fixed by pulumi.MockResourceMonitor.
func (m *applyMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outputs := args.Inputs.Copy()
	outputs["number"] = resource.NewStringProperty(projectNumber)

	if role, ok := args.Inputs["role"]; ok && role.IsString() && role.StringValue() == writerRole {
		member := ""
		if v, ok := args.Inputs["member"]; ok && v.IsString() {
			member = v.StringValue()
		}

		m.mu.Lock()
		m.writers[args.Name] = member
		m.mu.Unlock()
	}

	return args.Name + "-id", outputs, nil
}

func (m *applyMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

// setConfig marshals the given key/value pairs into the PULUMI_CONFIG
// environment variable the Pulumi test runtime reads config from.
func setConfig(t *testing.T, kv map[string]string) {
	t.Helper()

	flat := make(map[string]string, len(kv))
	for k, v := range kv {
		flat["project:"+k] = v
	}

	blob, err := json.Marshal(flat)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PULUMI_CONFIG", string(blob))
}

// baseConfig is the smallest config that reaches the registry grants:
// the GitHub connection is left unconfigured so Apply stops right after
// the registry, which keeps the test on the code under scrutiny.
func baseConfig() map[string]string {
	return map[string]string{
		"projectName":    "delivery-test",
		"registryId":     "images",
		"folderID":       "folders/123",
		"billingAccount": "01ABCD-234567-89EFGH",
	}
}

// runApply runs the blueprint against the mocks and returns the writer
// grants it created, keyed by logical resource name.
func runApply(t *testing.T, cfg map[string]string) map[string]string {
	t.Helper()
	setConfig(t, cfg)

	mocks := &applyMocks{writers: map[string]string{}}
	if err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return delivery.Apply(ctx, nil)
	}, pulumi.WithMocks("project", "stack", mocks)); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	return mocks.writers
}

func TestApply_WriterGrantCoversBothDefaultBuildIdentities(t *testing.T) {
	got := runApply(t, baseConfig())

	want := map[string]string{
		legacyGrantName:  legacyBuildMember,
		computeGrantName: computeBuildMember,
	}
	if !maps.Equal(got, want) {
		t.Fatalf("writer grants without buildServiceAccount:\n got %v\nwant %v", got, want)
	}
}

func TestApply_WriterGrantLandsOnConfiguredBuildServiceAccount(t *testing.T) {
	cfg := baseConfig()
	cfg["buildServiceAccount"] = "builder@delivery-test.iam.gserviceaccount.com"

	got := runApply(t, cfg)

	want := map[string]string{
		legacyGrantName: "serviceAccount:builder@delivery-test.iam.gserviceaccount.com",
	}
	if !maps.Equal(got, want) {
		t.Fatalf("writer grants with buildServiceAccount:\n got %v\nwant %v", got, want)
	}
}
