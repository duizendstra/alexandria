package buildstamp

import (
	"fmt"
	"maps"
	"regexp"
	"runtime/debug"
	"slices"
	"strings"
)

// Values a build script can set with -ldflags, e.g.
//
//	go build -ldflags "-X github.com/duizendstra/alexandria/go/platform/buildstamp.Commit=$(git rev-parse HEAD)"
//
// They take precedence over Go's embedded VCS settings, which are unavailable
// in some build layouts (notably a module graph using a local replace).
//
//nolint:gochecknoglobals // -ldflags -X can only write to package-level vars.
var (
	Version  = ""
	Commit   = ""
	Modified = "" // trueValue or "false".
	BuiltAt  = ""
)

const (
	// Unknown is the placeholder used when the commit could not be
	// determined. It never satisfies Matches.
	Unknown = "unknown"

	// trueValue is the -ldflags spelling of a dirty working tree.
	trueValue = "true"

	// shortSHALen is how much of a SHA Short renders — enough to recognise,
	// never enough to verify. Matches always requires the full SHA.
	shortSHALen = 7

	// dirtySuffix marks a dependency revision built from uncommitted changes.
	dirtySuffix = "-dirty"

	// fullSHALen is the length of a complete git object name.
	fullSHALen = 40
)

// fullSHA matches a complete git object name. Anything shorter is refused:
// abbreviated SHAs are ambiguous, and ambiguity is the thing a stamp exists to
// remove.
var fullSHA = regexp.MustCompile(fmt.Sprintf(`^[0-9a-f]{%d}$`, fullSHALen))

// Stamp is the build identity of one binary.
type Stamp struct {
	// Name of the binary, so several tools built from one repository can be
	// told apart in logs.
	Name string `json:"name,omitempty"`

	// Version is an optional human-facing release name.
	Version string `json:"version,omitempty"`

	// Commit is the full 40-character SHA, or Unknown.
	Commit string `json:"commit"`

	// Dirty reports whether the working tree had uncommitted changes.
	Dirty bool `json:"dirty"`

	// BuiltAt is the build timestamp, normally RFC 3339.
	BuiltAt string `json:"built_at,omitempty"`

	// Module is the main module path, when available.
	Module string `json:"module,omitempty"`

	// Deps records provenance the module graph does not pin — typically a
	// dependency consumed through a local replace. Values follow the same
	// shape as Commit: a SHA, optionally suffixed "-dirty", or Unknown.
	Deps map[string]string `json:"deps,omitempty"`
}

// Get returns the stamp of the running binary. Values set via -ldflags win;
// anything still missing is filled from Go's embedded VCS settings. deps
// records extra provenance the caller tracks itself, and is copied.
func Get(name string, deps map[string]string) Stamp {
	s := Stamp{
		Name:    name,
		Version: Version,
		Commit:  Commit,
		Dirty:   Modified == trueValue,
		BuiltAt: BuiltAt,
	}
	if len(deps) > 0 {
		s.Deps = make(map[string]string, len(deps))
		maps.Copy(s.Deps, deps)
	}
	s.fillFromBuildInfo()

	if s.Commit == "" {
		s.Commit = Unknown
	}

	return s
}

// String renders the stamp as one line, which is also what ParseStamp reads:
//
//	tool 1.4.0 commit=<sha> dirty=false built=2026-01-02T15:04:05Z lib=abc1234
//
// Dependency stamps are appended in sorted order so the line is stable between
// runs and can be compared byte for byte.
func (s *Stamp) String() string {
	var b strings.Builder
	if s.Name != "" {
		b.WriteString(s.Name)
	}
	if s.Version != "" {
		b.WriteString(" " + s.Version)
	}
	fmt.Fprintf(&b, " commit=%s dirty=%t", s.Commit, s.Dirty)

	if s.BuiltAt != "" {
		b.WriteString(" built=" + s.BuiltAt)
	}
	for _, k := range slices.Sorted(maps.Keys(s.Deps)) {
		b.WriteString(" " + k + "=" + s.Deps[k])
	}

	return strings.TrimSpace(b.String())
}

// Short renders "abc1234" or "abc1234-dirty", with the version when set. It is
// for humans reading logs; verification always uses the full SHA.
func (s *Stamp) Short() string {
	c := s.Commit
	if len(c) > shortSHALen {
		c = c[:shortSHALen]
	}
	if s.Dirty {
		c += dirtySuffix
	}
	if s.Version != "" {
		return s.Version + " (" + c + ")"
	}

	return c
}

// ParseStamp reads a stamp back out of a line produced by String.
//
// It is lenient about the leading words — a tool may print its own name and
// version however it likes — and strict about the key=value fields, because
// those are what Matches then judges. Unrecognised key=value pairs are kept as
// dependency stamps rather than discarded: dropping provenance silently is how
// a supervising check ends up verifying less than it claims.
func ParseStamp(line string) (Stamp, error) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return Stamp{}, fmt.Errorf("%w: empty line", ErrMalformedStamp)
	}

	var (
		s       Stamp
		leading []string
		seenKV  bool
	)

	for _, f := range fields {
		key, value, isKV := strings.Cut(f, "=")
		if !isKV {
			if seenKV {
				// A bare word after the key=value fields is not a shape we
				// produce; guessing would hide a malformed line.
				return Stamp{}, fmt.Errorf("%w: unexpected bare word %q after key=value fields", ErrMalformedStamp, f)
			}
			leading = append(leading, f)

			continue
		}
		seenKV = true

		if err := s.absorb(key, value); err != nil {
			return Stamp{}, err
		}
	}

	if s.Commit == "" {
		return Stamp{}, fmt.Errorf("%w: no commit= field", ErrMalformedStamp)
	}
	if len(leading) > 0 {
		s.Name = leading[0]
	}
	if len(leading) > 1 {
		s.Version = strings.Join(leading[1:], " ")
	}

	return s, nil
}

// Matches reports whether this stamp is acceptable for running against the
// given expected commit, which is normally the tip of the release branch.
//
// It returns a descriptive error rather than a bool, because the caller almost
// always wants to tell the operator what to rebuild and why. A nil error means
// the binary is exactly the expected commit, built from a clean tree, with
// clean dependency provenance.
func (s *Stamp) Matches(expectedCommit string) error {
	if !fullSHA.MatchString(expectedCommit) {
		return fmt.Errorf("%w: expected commit %q is not a full %d-character SHA",
			ErrMalformedStamp, expectedCommit, fullSHALen)
	}
	if s.Commit == "" || s.Commit == Unknown {
		return fmt.Errorf("%w — rebuild from a git checkout", ErrUnknownCommit)
	}
	if !fullSHA.MatchString(s.Commit) {
		return fmt.Errorf("%w: binary commit %q is not a full %d-character SHA",
			ErrMalformedStamp, s.Commit, fullSHALen)
	}
	if s.Commit != expectedCommit {
		return fmt.Errorf("%w: built from %s, expected %s — rebuild",
			ErrCommitMismatch, s.Commit, expectedCommit)
	}
	if s.Dirty {
		return fmt.Errorf("%w — rebuild from a clean checkout", ErrDirtyBuild)
	}

	for _, name := range slices.Sorted(maps.Keys(s.Deps)) {
		if err := checkDep(name, s.Deps[name]); err != nil {
			return err
		}
	}

	return nil
}

// RequireDeps reports whether every named dependency stamp is present and
// clean. Use it when a build is only trustworthy if specific provenance was
// recorded: Matches can only judge the stamps that are there, so a dependency
// the build forgot to record would otherwise pass unnoticed.
func (s *Stamp) RequireDeps(names ...string) error {
	for _, name := range names {
		value, ok := s.Deps[name]
		if !ok {
			return fmt.Errorf("%w: %q — build via the release script", ErrDependencyMissing, name)
		}
		if err := checkDep(name, value); err != nil {
			return err
		}
	}

	return nil
}

// fillFromBuildInfo completes the stamp from Go's embedded VCS settings,
// without overriding anything -ldflags already provided.
func (s *Stamp) fillFromBuildInfo() {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	s.Module = bi.Main.Path

	for _, setting := range bi.Settings {
		switch setting.Key {
		case "vcs.revision":
			if s.Commit == "" {
				s.Commit = setting.Value
			}
		case "vcs.time":
			if s.BuiltAt == "" {
				s.BuiltAt = setting.Value
			}
		case "vcs.modified":
			if Modified == "" {
				s.Dirty = setting.Value == trueValue
			}
		}
	}
}

// absorb applies one key=value field from a stamp line.
func (s *Stamp) absorb(key, value string) error {
	switch key {
	case "commit":
		s.Commit = value
	case "dirty":
		switch value {
		case trueValue:
			s.Dirty = true
		case "false":
			s.Dirty = false
		default:
			return fmt.Errorf("%w: dirty must be true or false, got %q", ErrMalformedStamp, value)
		}
	case "built":
		s.BuiltAt = value
	default:
		if s.Deps == nil {
			s.Deps = map[string]string{}
		}
		s.Deps[key] = value
	}

	return nil
}

// checkDep applies the cleanliness rules to one recorded dependency revision.
func checkDep(name, value string) error {
	switch {
	case value == "" || value == Unknown:
		return fmt.Errorf("%w: %q has an unknown revision — rebuild against a clean checkout",
			ErrDependencyUnclean, name)
	case strings.HasSuffix(value, dirtySuffix):
		return fmt.Errorf("%w: %q was dirty at build time (%s) — a binary built against uncommitted changes is not reproducible",
			ErrDependencyUnclean, name, value)
	}

	return nil
}
