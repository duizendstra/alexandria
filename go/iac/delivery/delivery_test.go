package delivery_test

import (
	"encoding/json"
	"maps"
	"strings"
	"sync"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/delivery"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Shared fixture values for the writer-grant tests.
const (
	projectNumber = "123456789012"
	writerRole    = "roles/artifactregistry.writer"

	legacyBuildMember  = "serviceAccount:" + projectNumber + "@cloudbuild.gserviceaccount.com"
	computeBuildMember = "serviceAccount:" + projectNumber + "-compute@developer.gserviceaccount.com"

	legacyGrantName  = "build-ar-writer"
	computeGrantName = "build-ar-writer-compute"
)

// Optional config keys under test and the resource types they declare.
const (
	keyRepositories = "repositories"
	keyConsumers    = "consumerWorkloadStacks"

	typeRepository     = "gcp:cloudbuildv2/repository:Repository"
	typeTrigger        = "gcp:cloudbuild/trigger:Trigger"
	typeRegistryMember = "gcp:artifactregistry/repositoryIamMember:RepositoryIamMember"
	typeStackReference = "pulumi:pulumi:StackReference"

	// malformedBlock is a truncated JSON array: present, not parseable.
	malformedBlock = `[{"name": "app"`
)

// requiredConfig is the minimal config Apply needs to reach the
// repository and consumer-grant steps: placement, a registry, and a
// configured Git hosting connection.
var requiredConfig = map[string]any{
	"projectName":              "delivery-test",
	"registryId":               "images",
	"folderID":                 "folders/123",
	"billingAccount":           "000000-000000-000000",
	"githubAppInstallationId":  12345,
	"githubOAuthSecretVersion": "projects/delivery-test/secrets/github-token/versions/1",
}

// wellFormedBlocks declares one repository with one trigger and one
// consumer workload stack.
var wellFormedBlocks = map[string]any{
	keyRepositories: []map[string]any{{
		"name":      "app",
		"remoteURI": "https://example.com/org/app.git",
		"triggers": []map[string]any{{
			"name":       "release",
			"tagPattern": "v.*",
			"configFile": "cloudbuild.yaml",
		}},
	}},
	keyConsumers: []string{"org/workloads/dev"},
}

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

// countingMocks echoes inputs back as outputs, fills the outputs the
// graph reads (project number, workload stack exports), and counts the
// resources registered per type token.
type countingMocks struct {
	mu    *sync.Mutex
	count map[string]int
}

//nolint:gocritic // signature is fixed by pulumi.MockResourceMonitor.
func (m countingMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	m.mu.Lock()
	m.count[args.TypeToken]++
	m.mu.Unlock()

	if args.TypeToken == typeStackReference {
		return args.Name, resource.PropertyMap{
			"outputs": resource.NewObjectProperty(resource.PropertyMap{
				"computeProjectNumber": resource.NewStringProperty("987654321"),
			}),
		}, nil
	}

	outputs := args.Inputs.Copy()
	outputs["number"] = resource.NewStringProperty(projectNumber)

	return args.Name + "-id", outputs, nil
}

func (countingMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func (m countingMocks) created(typeToken string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.count[typeToken]
}

// setConfig marshals key/value pairs into PULUMI_CONFIG under the "project"
// namespace, matching config.New(ctx, "")'s default namespace when Apply
// runs under pulumi.WithMocks("project", "stack", ...). Strings are passed
// through verbatim, so a malformed JSON string reaches TryObject as-is.
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

// baseConfig is the smallest config that reaches the registry grants:
// the GitHub connection is left unconfigured so Apply stops right after
// the registry, which keeps the test on the code under scrutiny.
func baseConfig() map[string]any {
	return map[string]any{
		"projectName":    "delivery-test",
		"registryId":     "images",
		"folderID":       "folders/123",
		"billingAccount": "01ABCD-234567-89EFGH",
	}
}

// runApplyGrants runs the blueprint against the recording mocks and
// returns the writer grants it created, keyed by logical resource name.
func runApplyGrants(t *testing.T, cfg map[string]any) map[string]string {
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

// runApply runs delivery.Apply(ctx, nil) under the counting mocks with
// requiredConfig plus extraConfig.
func runApply(t *testing.T, extraConfig map[string]any) (countingMocks, error) {
	t.Helper()

	cfg := make(map[string]any, len(requiredConfig)+len(extraConfig))
	maps.Copy(cfg, requiredConfig)
	maps.Copy(cfg, extraConfig)
	setConfig(t, cfg)

	mocks := countingMocks{mu: &sync.Mutex{}, count: map[string]int{}}

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return delivery.Apply(ctx, nil)
	}, pulumi.WithMocks("project", "stack", mocks))

	return mocks, err
}

func TestApply_WriterGrantCoversBothDefaultBuildIdentities(t *testing.T) {
	got := runApplyGrants(t, baseConfig())

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

	got := runApplyGrants(t, cfg)

	want := map[string]string{
		legacyGrantName: "serviceAccount:builder@delivery-test.iam.gserviceaccount.com",
	}
	if !maps.Equal(got, want) {
		t.Fatalf("writer grants with buildServiceAccount:\n got %v\nwant %v", got, want)
	}
}

func TestApply_OmittedOptionalBlocksApplyCleanly(t *testing.T) {
	mocks, err := runApply(t, nil)
	if err != nil {
		t.Fatalf("Apply with optional blocks omitted: %v", err)
	}

	for _, typeToken := range []string{typeRepository, typeTrigger, typeStackReference} {
		if n := mocks.created(typeToken); n != 0 {
			t.Errorf("%s resources with optional blocks omitted = %d, want 0", typeToken, n)
		}
	}
}

func TestApply_WellFormedOptionalBlocksCreateResources(t *testing.T) {
	mocks, err := runApply(t, wellFormedBlocks)
	if err != nil {
		t.Fatalf("Apply with well-formed optional blocks: %v", err)
	}

	want := map[string]int{
		typeRepository:     1,
		typeTrigger:        1,
		typeStackReference: 1,
		// Two build writer grants — the legacy Cloud Build SA and the
		// Compute default SA are both granted when buildServiceAccount
		// is unset — plus one consumer reader grant.
		typeRegistryMember: 3,
	}
	for typeToken, n := range want {
		if got := mocks.created(typeToken); got != n {
			t.Errorf("%s resources = %d, want %d", typeToken, got, n)
		}
	}
}

// A block that is present but does not parse must abort the update:
// read as empty it would retire every resource it had declared.
func TestApply_MalformedOptionalBlockFails(t *testing.T) {
	for _, key := range []string{keyRepositories, keyConsumers} {
		t.Run(key, func(t *testing.T) {
			cfg := maps.Clone(wellFormedBlocks)
			cfg[key] = malformedBlock

			_, err := runApply(t, cfg)
			if err == nil {
				t.Fatalf("Apply with malformed %s: want error, got nil", key)
			}
			if want := `config key "` + key + `"`; !strings.Contains(err.Error(), want) {
				t.Fatalf("error %v does not contain %q", err, want)
			}
		})
	}
}
