package passstore

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const passTimeout = 5 * time.Second

// Show reads a value from the local pass store.
func Show(path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), passTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "pass", "show", path).Output() //nolint:gosec // path is hardcoded by caller.
	if err != nil {
		return "", fmt.Errorf("pass show %s: %w", path, err)
	}

	return strings.TrimSpace(string(out)), nil
}

// MustShow reads from pass or panics. Use only in Pulumi programs
// where a missing credential should abort the deploy.
func MustShow(path string) string {
	v, err := Show(path)
	if err != nil {
		panic(fmt.Sprintf("pass show %s: %v", path, err))
	}

	return v
}
