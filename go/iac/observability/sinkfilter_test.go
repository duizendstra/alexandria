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
