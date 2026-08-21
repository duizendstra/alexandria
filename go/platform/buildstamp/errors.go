package buildstamp

import "errors"

// Sentinels for the refusals. They are wrapped with context by the callers, so
// a caller can both show the operator a specific message and test the class of
// problem with errors.Is.
var (
	// ErrMalformedStamp means the stamp line could not be read.
	ErrMalformedStamp = errors.New("malformed build stamp")

	// ErrUnknownCommit means the binary does not know which commit it is.
	ErrUnknownCommit = errors.New("binary has no known commit")

	// ErrCommitMismatch means the binary is a different commit than expected.
	ErrCommitMismatch = errors.New("binary was built from an unexpected commit")

	// ErrDirtyBuild means the binary was built from uncommitted changes and is
	// therefore not reproducible.
	ErrDirtyBuild = errors.New("binary was built from a dirty working tree")

	// ErrDependencyUnclean means a recorded dependency revision was dirty or
	// unknown, so the binary is not actually pinned.
	ErrDependencyUnclean = errors.New("dependency revision is not clean")

	// ErrDependencyMissing means the build did not record a dependency the
	// caller requires.
	ErrDependencyMissing = errors.New("build stamp does not record the dependency")
)
