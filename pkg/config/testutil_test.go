package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// skipIfNoChmod skips the test on platforms where chmod 0o000 does not
// prevent reading (e.g. Windows, which ignores POSIX permission bits).
func skipIfNoChmod(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("chmod 0o000 does not prevent reading on Windows")
	}
}

// skipIfNoSymlink skips the test when the OS does not allow creating symlinks
// without elevated privileges (Windows without developer mode).
func skipIfNoSymlink(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.Mkdir(target, 0o750); err != nil {
		t.Skip("cannot create symlink test target:", err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks not supported:", err)
	}
}
