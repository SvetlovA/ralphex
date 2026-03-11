package executor

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// goos is the current operating system, overridable in tests.
var goos = runtime.GOOS

// CommandFactory creates exec.Cmd instances with platform-appropriate adaptations.
// on Windows, .cmd and .bat files are automatically wrapped with cmd /C.
type CommandFactory struct{}

// Command creates an exec.Cmd for the given program and arguments.
// on Windows, if name has a .cmd or .bat extension, prepends "cmd /C".
func (f CommandFactory) Command(name string, args ...string) *exec.Cmd {
	if goos == "windows" && isBatchFile(name) {
		return exec.Command("cmd", append([]string{"/C", name}, args...)...)
	}
	return exec.Command(name, args...)
}

// CommandContext creates an exec.Cmd with context for the given program and arguments.
// on Windows, if name has a .cmd or .bat extension, prepends "cmd /C".
func (f CommandFactory) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	if goos == "windows" && isBatchFile(name) {
		return exec.CommandContext(ctx, "cmd", append([]string{"/C", name}, args...)...)
	}
	return exec.CommandContext(ctx, name, args...)
}

// isBatchFile checks if the file has a .cmd or .bat extension (case-insensitive).
func isBatchFile(name string) bool {
	ext := filepath.Ext(name)
	return strings.EqualFold(ext, ".cmd") || strings.EqualFold(ext, ".bat")
}
