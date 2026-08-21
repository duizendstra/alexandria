package runstate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrBadSubject reports a subject that cannot be part of a file name.
var ErrBadSubject = errors.New("subject may not contain a path separator or a parent reference")

// dirPerm and filePerm keep run state readable by its owner only.
const (
	dirPerm  = 0o700
	filePerm = 0o600
)

// checkSubject rejects a subject that would escape the state directory. A
// subject is usually an identifier the caller already trusts — an account, a
// job name — but it becomes part of a path, so it is checked all the same.
func checkSubject(subject string) error {
	switch {
	case subject == "", subject == "." || subject == "..":
		return fmt.Errorf("%q: %w", subject, ErrBadSubject)
	case strings.ContainsRune(subject, os.PathSeparator), strings.ContainsRune(subject, '/'), strings.ContainsRune(subject, '\\'):
		return fmt.Errorf("%q: %w", subject, ErrBadSubject)
	case strings.Contains(subject, ".."):
		return fmt.Errorf("%q: %w", subject, ErrBadSubject)
	}

	return nil
}

// pathFor composes <dir>/<prefix><subject><suffix> after checking the subject.
func pathFor(dir, prefix, subject, suffix string) (string, error) {
	if err := checkSubject(subject); err != nil {
		return "", err
	}

	return filepath.Join(dir, prefix+subject+suffix), nil
}
