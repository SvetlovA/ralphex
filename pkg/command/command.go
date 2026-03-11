// Package command provides platform-aware exec.Cmd creation.
package command

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Factory creates exec.Cmd instances with platform-appropriate adaptations.
// on Windows, .cmd and .bat files are automatically wrapped with cmd /C.
// zero value is ready to use (defaults to runtime.GOOS).
type Factory struct {
	goos string // operating system override for testing; empty means runtime.GOOS
}

// os returns the effective operating system string.
func (f Factory) os() string {
	if f.goos != "" {
		return f.goos
	}
	return runtime.GOOS
}

// Command creates an exec.Cmd for the given program and arguments.
// on Windows, if name has a .cmd or .bat extension, prepends "cmd /C".
func (f Factory) Command(name string, args ...string) *exec.Cmd {
	if f.os() == "windows" && isBatchFile(name) {
		return exec.Command("cmd", append([]string{"/C", name}, args...)...)
	}
	return exec.Command(name, args...)
}

// CommandContext creates an exec.Cmd with context for the given program and arguments.
// on Windows, if name has a .cmd or .bat extension, prepends "cmd /C".
func (f Factory) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	if f.os() == "windows" && isBatchFile(name) {
		return exec.CommandContext(ctx, "cmd", append([]string{"/C", name}, args...)...)
	}
	return exec.CommandContext(ctx, name, args...)
}

// isBatchFile checks if the file has a .cmd or .bat extension (case-insensitive).
func isBatchFile(name string) bool {
	ext := filepath.Ext(name)
	return strings.EqualFold(ext, ".cmd") || strings.EqualFold(ext, ".bat")
}
