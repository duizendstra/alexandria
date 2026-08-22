//go:build windows

package procrun_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/duizendstra/alexandria/go/platform/procrun"
)

// plainFile writes a non-empty file with no execute semantics; on Windows
// executability comes from the name alone.
func plainFile(t *testing.T, dir, name string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("stub"), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}

	return path
}

func TestLookPathAppendsPathExtExtensions(t *testing.T) {
	t.Setenv("PATHEXT", ".COM;.EXE;.BAT;.CMD")

	dir := t.TempDir()
	want := plainFile(t, dir, "tool.exe")

	got, err := (&procrun.Runner{Path: dir}).LookPath("tool")
	if err != nil {
		t.Fatalf("LookPath: %v", err)
	}

	if got != want {
		t.Fatalf("LookPath = %q, want %q", got, want)
	}
}

func TestLookPathPrefersEarlierPathExtEntries(t *testing.T) {
	t.Setenv("PATHEXT", ".COM;.EXE")

	dir := t.TempDir()
	want := plainFile(t, dir, "tool.com")
	plainFile(t, dir, "tool.exe")

	got, err := (&procrun.Runner{Path: dir}).LookPath("tool")
	if err != nil || got != want {
		t.Fatalf("LookPath = %q, %v, want %q", got, err, want)
	}
}

func TestLookPathAcceptsAnExplicitExecutableExtension(t *testing.T) {
	t.Setenv("PATHEXT", ".COM;.EXE;.BAT;.CMD")

	dir := t.TempDir()
	want := plainFile(t, dir, "tool.bat")

	got, err := (&procrun.Runner{Path: dir}).LookPath("tool.bat")
	if err != nil || got != want {
		t.Fatalf("LookPath = %q, %v, want %q", got, err, want)
	}
}

func TestLookPathRejectsANonExecutableExtension(t *testing.T) {
	t.Setenv("PATHEXT", ".EXE")

	dir := t.TempDir()
	plainFile(t, dir, "tool.txt")

	if _, err := (&procrun.Runner{Path: dir}).LookPath("tool.txt"); !errors.Is(err, procrun.ErrNotOnPath) {
		t.Fatalf("err = %v, want ErrNotOnPath", err)
	}
}

func TestLookPathDefaultsPathExtWhenUnset(t *testing.T) {
	t.Setenv("PATHEXT", "")

	dir := t.TempDir()
	want := plainFile(t, dir, "tool.exe")

	got, err := (&procrun.Runner{Path: dir}).LookPath("tool")
	if err != nil || got != want {
		t.Fatalf("LookPath = %q, %v, want %q", got, err, want)
	}
}
