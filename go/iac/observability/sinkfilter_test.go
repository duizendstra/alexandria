package observability

import "testing"

func TestSinkFilter(t *testing.T) {
	cases := map[string]struct {
		names []string
		want  string
	}{
		"default log name matches the original hardcoded filter": {
			names: defaultSinkLogNames(),
			want:  `logName:"logs/cloudaudit.googleapis.com"`,
		},
		"single caller-supplied name": {
			names: []string{"example.googleapis.com"},
			want:  `logName:"logs/example.googleapis.com"`,
		},
		"multiple caller-supplied names are OR'd": {
			names: []string{"one.example.com", "two.example.com"},
			want:  `logName:"logs/one.example.com" OR logName:"logs/two.example.com"`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := sinkFilter(tc.names); got != tc.want {
				t.Errorf("sinkFilter(%v) = %q, want %q", tc.names, got, tc.want)
			}
		})
	}
}

func TestValidateSinkLogName(t *testing.T) {
	valid := []string{
		"cloudaudit.googleapis.com",
		"example.googleapis.com%2Frequests",
		"a-b_c.d/e",
	}
	for _, name := range valid {
		if err := validateSinkLogName(name); err != nil {
			t.Errorf("validateSinkLogName(%q) = %v, want nil", name, err)
		}
	}

	invalid := []string{
		"",
		`x" OR NOT logName:"nothing`,
		"has space",
		"semi;colon",
	}
	for _, name := range invalid {
		if err := validateSinkLogName(name); err == nil {
			t.Errorf("validateSinkLogName(%q) = nil, want error", name)
		}
	}
}
