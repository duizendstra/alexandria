//go:build !windows

package procrun

import (
	"os"
	"path/filepath"
)

// findExecutable reports whether dir holds an executable file for name, and
// returns its path. On Unix an executable is a non-directory with any execute
// bit set.
func findExecutable(dir, name string) (string, bool) {
	path := filepath.Join(dir, name)

	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() || fi.Mode().Perm()&0o111 == 0 {
		return "", false
	}

	return path, true
}
