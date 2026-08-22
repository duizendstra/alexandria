package identity_test

import (
	"encoding/json"
	"maps"
	"strings"
	"sync"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/identity"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Optional config keys under test and the resource types they declare.
const (
	keySecrets         = "secrets"
	keyServiceAccounts = "serviceAccounts"
	keyImpersonators   = "impersonators"

	typeSecret         = "gcp:secretmanager/secret:Secret"
	typeServiceAccount = "gcp:serviceaccount/account:Account"
	typeSAIAMMember    = "gcp:serviceaccount/iAMMember:IAMMember"

	// malformedBlock is a truncated JSON array: present, not parseable.
	malformedBlock = `[{"name": "api-key"`
)

// requiredConfig is the minimal config Apply needs when no Params and
// no governanceStack are supplied.
var requiredConfig = map[string]any{
	"projectName":    "identity-test",
	"folderID":       "folders/123",
	"billingAccount": "000000-000000-000000",
	"consumerSAs":    []string{"consumer@example.iam.gserviceaccount.com"},
}

// wellFormedBlocks declares one of everything the optional keys can hold.
var wellFormedBlocks = map[string]any{
	keySecrets:         []map[string]string{{"name": "api-key", "ref": "example/api-key"}},
	keyServiceAccounts: []map[string]string{{"id": "worker", "displayName": "Worker"}},
	keyImpersonators:   []string{"group:deployers@example.com"},
}

// countingMocks echoes inputs back as outputs, fills the outputs the
// graph reads (project number, service-account email), and counts the
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

	outputs := args.Inputs.Copy()
	outputs["number"] = resource.NewStringProperty("123456789")
	outputs["email"] = resource.NewStringProperty(args.Name + "@example.iam.gserviceaccount.com")

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

// runApply runs identity.Apply under mocks with requiredConfig plus
// extraConfig, resolving every secret ref to a fixed value.
func runApply(t *testing.T, extraConfig map[string]any) (countingMocks, error) {
	t.Helper()

	cfg := make(map[string]any, len(requiredConfig)+len(extraConfig))
	maps.Copy(cfg, requiredConfig)
	maps.Copy(cfg, extraConfig)
	setConfig(t, cfg)

	mocks := countingMocks{mu: &sync.Mutex{}, count: map[string]int{}}
	params := &identity.Params{Resolver: func(ref string) (string, error) { return "value-of-" + ref, nil }}

	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return identity.Apply(ctx, params)
	}, pulumi.WithMocks("project", "stack", mocks))

	return mocks, err
}

func TestApply_OmittedOptionalBlocksApplyCleanly(t *testing.T) {
	mocks, err := runApply(t, nil)
	if err != nil {
		t.Fatalf("Apply with optional blocks omitted: %v", err)
	}

	for _, typeToken := range []string{typeSecret, typeServiceAccount, typeSAIAMMember} {
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
		typeSecret:         1,
		typeServiceAccount: 1,
		typeSAIAMMember:    3, // user, token creator, OIDC token creator.
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
	for _, key := range []string{keySecrets, keyServiceAccounts, keyImpersonators} {
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
