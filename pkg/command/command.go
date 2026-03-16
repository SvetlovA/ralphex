// Package command provides platform-aware exec.Cmd creation.
package command

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Wrapper creates exec.Cmd instances with platform-appropriate adaptations.
// on Windows, .cmd and .bat files are automatically wrapped with cmd /C.
// zero value is ready to use (defaults to runtime.GOOS).
type Wrapper struct {
	goos string // operating system override for testing; empty means runtime.GOOS
}

// os returns the effective operating system string.
func (w Wrapper) os() string {
	if w.goos != "" {
		return w.goos
	}
	return runtime.GOOS
}

// Command creates an exec.Cmd for the given program and arguments.
// on Windows, if name has a .cmd or .bat extension, prepends "cmd /C".
func (w Wrapper) Command(name string, args ...string) *exec.Cmd {
	if w.needCmdWrap(name) {
		return exec.Command("cmd", append([]string{"/C", name}, args...)...) //nolint:noctx // intentional: we handle context cancellation via process group kill
	}
	return exec.Command(name, args...) //nolint:noctx // intentional: we handle context cancellation via process group kill
}

// CommandContext creates an exec.Cmd with context for the given program and arguments.
// on Windows, if name has a .cmd or .bat extension, prepends "cmd /C".
func (w Wrapper) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	if w.needCmdWrap(name) {
		return exec.CommandContext(ctx, "cmd", append([]string{"/C", name}, args...)...)
	}
	return exec.CommandContext(ctx, name, args...)
}

func (w Wrapper) needCmdWrap(name string) bool {
	return w.os() == "windows" && (isBatchFile(name) || !isExecutable(name))
}

// isBatchFile checks if the file has a .cmd or .bat extension (case-insensitive).
func isBatchFile(name string) bool {
	ext := filepath.Ext(name)
	return strings.EqualFold(ext, ".cmd") || strings.EqualFold(ext, ".bat")
}

func isExecutable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
