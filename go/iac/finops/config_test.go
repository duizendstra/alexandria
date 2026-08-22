package finops

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// keyThresholds is the optional config key under test.
const keyThresholds = "thresholds"

// malformedThresholds is present but not parseable as a float64 array:
// a trailing comma is the typo the issue reports.
const malformedThresholds = `[0.5, 0.75,]`

// recordingMocks echoes inputs back as outputs and keeps the inputs of
// every registered resource by type token so tests can inspect them.
type recordingMocks struct {
	mu     *sync.Mutex
	inputs map[string][]resource.PropertyMap
}

func newRecordingMocks() recordingMocks {
	return recordingMocks{mu: &sync.Mutex{}, inputs: map[string][]resource.PropertyMap{}}
}

//nolint:gocritic // signature is fixed by pulumi.MockResourceMonitor.
func (m recordingMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	m.mu.Lock()
	m.inputs[args.TypeToken] = append(m.inputs[args.TypeToken], args.Inputs.Copy())
	m.mu.Unlock()

	outputs := args.Inputs.Copy()
	outputs["number"] = resource.NewStringProperty("123456789")

	return args.Name + "-id", outputs, nil
}

func (recordingMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func (m recordingMocks) registered(typeToken string) []resource.PropertyMap {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.inputs[typeToken]
}

// setConfig marshals key/value pairs into PULUMI_CONFIG under the "project"
// namespace, matching config.New(ctx, "")'s default namespace when the
// program runs under pulumi.WithMocks("project", "stack", ...). Strings
// are passed through verbatim, so a malformed JSON string reaches
// TryObject as-is.
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

// readThresholds runs optionalObject for the thresholds key against a
// stack whose config is exactly kv, returning what it unmarshalled and
// the error it reported.
func readThresholds(t *testing.T, kv map[string]any) ([]float64, error) {
	t.Helper()
	setConfig(t, kv)

	var out []float64
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		return optionalObject(config.New(ctx, ""), keyThresholds, &out)
	}, pulumi.WithMocks("project", "stack", newRecordingMocks()))

	return out, err
}

func TestOptionalObject_AbsentKeyLeavesOutUntouched(t *testing.T) {
	out, err := readThresholds(t, map[string]any{})
	if err != nil {
		t.Fatalf("optionalObject with key absent: %v", err)
	}
	if out != nil {
		t.Fatalf("out = %v, want nil (untouched)", out)
	}
}

func TestOptionalObject_ValidKeyUnmarshals(t *testing.T) {
	out, err := readThresholds(t, map[string]any{keyThresholds: []float64{0.8, 1.0}})
	if err != nil {
		t.Fatalf("optionalObject with valid key: %v", err)
	}
	if len(out) != 2 || out[0] != 0.8 || out[1] != 1.0 {
		t.Fatalf("out = %v, want [0.8 1]", out)
	}
}

func TestOptionalObject_EmptyArrayIsEmptyNotError(t *testing.T) {
	out, err := readThresholds(t, map[string]any{keyThresholds: []float64{}})
	if err != nil {
		t.Fatalf("optionalObject with empty array: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("out = %v, want empty", out)
	}
}

// A key that is present but does not parse must be reported, and the
// report must name the key and keep the cause reachable.
func TestOptionalObject_MalformedKeyFails(t *testing.T) {
	for name, value := range map[string]string{
		"trailing comma": malformedThresholds,
		"scalar string":  `"0.9"`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := readThresholds(t, map[string]any{keyThresholds: value})
			if err == nil {
				t.Fatal("optionalObject with malformed key: want error, got nil")
			}
			if errors.Is(err, config.ErrMissingVar) {
				t.Fatalf("error %v reports the key as missing", err)
			}
			if want := `config key "` + keyThresholds + `"`; !strings.Contains(err.Error(), want) {
				t.Fatalf("error %v does not contain %q", err, want)
			}
		})
	}
}
