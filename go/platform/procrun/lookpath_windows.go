//go:build windows

package procrun

import (
	"os"
	"path/filepath"
	"strings"
)

// defaultPathExt mirrors the Windows default when PATHEXT is unset or empty.
const defaultPathExt = ".COM;.EXE;.BAT;.CMD"

// findExecutable reports whether dir holds an executable file for name, and
// returns its path. Windows has no execute bit: a file is executable when its
// extension appears in PATHEXT, so a bare name is tried with each PATHEXT
// extension appended, in order, the way cmd.exe resolves commands.
func findExecutable(dir, name string) (string, bool) {
	exts := pathExts()

	if hasPathExt(name, exts) {
		path := filepath.Join(dir, name)
		if isFile(path) {
			return path, true
		}

		return "", false
	}

	for _, ext := range exts {
		path := filepath.Join(dir, name+ext)
		if isFile(path) {
			return path, true
		}
	}

	return "", false
}

func isFile(path string) bool {
	fi, err := os.Stat(path)

	return err == nil && !fi.IsDir()
}

// pathExts returns the executable extensions from PATHEXT, falling back to
// the Windows default when the variable is unset or empty.
func pathExts() []string {
	raw := os.Getenv("PATHEXT")
	if raw == "" {
		raw = defaultPathExt
	}

	var exts []string

	for _, ext := range strings.Split(raw, ";") {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			continue
		}

		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}

		exts = append(exts, ext)
	}

	return exts
}

// hasPathExt reports whether name already ends in one of the executable
// extensions, compared case-insensitively as Windows does.
func hasPathExt(name string, exts []string) bool {
	for _, ext := range exts {
		if len(name) > len(ext) && strings.EqualFold(name[len(name)-len(ext):], ext) {
			return true
		}
	}

	return false
}
