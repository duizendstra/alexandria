package observability

import "testing"

func TestHostFromURL(t *testing.T) {
	const host = "app.example.com"
	cases := map[string]string{
		"https://" + host + "/":         host,
		"https://" + host:               host,
		"http://x.test/":                "x.test",
		host:                            host,
		"https://svc-abc-uc.a.run.app/": "svc-abc-uc.a.run.app",
	}
	for in, want := range cases {
		if got := hostFromURL(in); got != want {
			t.Errorf("hostFromURL(%q) = %q, want %q", in, got, want)
		}
	}
}
