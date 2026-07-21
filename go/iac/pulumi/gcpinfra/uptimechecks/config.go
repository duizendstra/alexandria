package uptimechecks

import (
	"errors"
	"fmt"
)

var (
	// ErrDisplayNameRequired means the uptime check has no display name.
	ErrDisplayNameRequired = errors.New("uptimechecks: DisplayName is required")
	// ErrHostRequired means the uptime check has no host to monitor.
	ErrHostRequired = errors.New("uptimechecks: Host is required")
	// ErrPortOutOfRange means Port is set outside the valid 1-65535 range.
	ErrPortOutOfRange = errors.New("uptimechecks: Port must be between 1 and 65535")
	// ErrPeriodInvalid means Period is not one of the values GCP accepts.
	ErrPeriodInvalid = errors.New("uptimechecks: Period must be one of 60s, 300s, 600s, 900s")
	// ErrStatusClassInvalid means an AcceptedStatusClasses entry is not a known class.
	ErrStatusClassInvalid = errors.New("uptimechecks: AcceptedStatusClasses entries must be one of 1xx, 2xx, 3xx, 4xx, 5xx, any")
)

// Response status classes accepted in Config.AcceptedStatusClasses.
const (
	Class1xx = "1xx"
	Class2xx = "2xx"
	Class3xx = "3xx"
	Class4xx = "4xx"
	Class5xx = "5xx"
	ClassAny = "any"
)

// Defaults applied by Apply when the corresponding field is left zero.
const (
	defaultPath    = "/"
	defaultPort    = 443
	defaultPeriod  = "60s"
	defaultTimeout = "10s"
)

// statusClassEnum maps a friendly status class to the GCP StatusClass enum,
// reporting whether the class is a known one.
func statusClassEnum(friendly string) (string, bool) {
	switch friendly {
	case Class1xx:
		return "STATUS_CLASS_1XX", true
	case Class2xx:
		return "STATUS_CLASS_2XX", true
	case Class3xx:
		return "STATUS_CLASS_3XX", true
	case Class4xx:
		return "STATUS_CLASS_4XX", true
	case Class5xx:
		return "STATUS_CLASS_5XX", true
	case ClassAny:
		return "STATUS_CLASS_ANY", true
	default:
		return "", false
	}
}

func validPeriod(p string) bool {
	switch p {
	case "60s", "300s", "600s", "900s":
		return true
	default:
		return false
	}
}

// Config defines an HTTPS uptime check and its failure alert policy.
//
// It is deliberately serializable and free of Pulumi types so a composition
// root can drive it from JSON. The runtime inputs an uptime check needs —
// the project ID and the notification channel IDs the alert routes to — are
// passed to Apply as Pulumi values, not carried here.
type Config struct {
	// DisplayName is the uptime check (and derived alert) display name.
	DisplayName string `json:"displayName"`
	// Host is the hostname to probe, e.g. "app.example.com".
	Host string `json:"host"`
	// Path is the request path (default: "/").
	Path string `json:"path"`
	// Port is the TCP port to probe (default: 443).
	Port int `json:"port"`
	// AcceptedStatusClasses are the response classes treated as healthy
	// (default: [Class2xx]). Add Class3xx for IAP-protected endpoints, whose
	// unauthenticated probes are redirected to the sign-in page.
	AcceptedStatusClasses []string `json:"acceptedStatusClasses"`
	// Period is how often the check runs; one of 60s, 300s, 600s, 900s
	// (default: 60s).
	Period string `json:"period"`
	// Timeout is the per-probe timeout (default: 10s).
	Timeout string `json:"timeout"`
}

// Validate checks that the uptime check configuration is complete and well-formed.
func (c *Config) Validate() error {
	if c.DisplayName == "" {
		return ErrDisplayNameRequired
	}
	if c.Host == "" {
		return ErrHostRequired
	}
	if c.Port < 0 || c.Port > 65535 {
		return ErrPortOutOfRange
	}
	if c.Period != "" && !validPeriod(c.Period) {
		return ErrPeriodInvalid
	}
	for _, sc := range c.AcceptedStatusClasses {
		if _, ok := statusClassEnum(sc); !ok {
			return fmt.Errorf("%w: %q", ErrStatusClassInvalid, sc)
		}
	}

	return nil
}

// resolved returns the effective values Apply provisions from, with defaults
// filled in. It stays Pulumi-free so it can be unit-tested directly.
func (c *Config) resolved() resolvedConfig {
	r := resolvedConfig{
		displayName: c.DisplayName,
		host:        c.Host,
		path:        orString(c.Path, defaultPath),
		port:        c.Port,
		period:      orString(c.Period, defaultPeriod),
		timeout:     orString(c.Timeout, defaultTimeout),
	}
	if r.port == 0 {
		r.port = defaultPort
	}

	classes := c.AcceptedStatusClasses
	if len(classes) == 0 {
		classes = []string{Class2xx}
	}
	for _, sc := range classes {
		enum, _ := statusClassEnum(sc)
		r.statusClasses = append(r.statusClasses, enum)
	}

	return r
}

type resolvedConfig struct {
	displayName   string
	host          string
	path          string
	port          int
	period        string
	timeout       string
	statusClasses []string
}

func orString(v, fallback string) string {
	if v == "" {
		return fallback
	}

	return v
}
