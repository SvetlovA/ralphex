// Package command creates exec.Cmd instances with platform-specific adaptations.
package command

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// goos is the operating system the wrapping decision is made against.
// production always uses runtime.GOOS; tests override it to cover every platform
// branch on any host.
var goos = runtime.GOOS

// Command creates an exec.Cmd for the given program and arguments, mirroring
// exec.Command. on Windows a .cmd or .bat target is invoked through "cmd /C";
// every other target, and every other platform, reaches exec.Command unchanged.
func Command(name string, args ...string) *exec.Cmd {
	name, args = adapt(name, args)
	return exec.Command(name, args...) //nolint:noctx // callers handle cancellation by killing the process group
}

// CommandContext is Command with a ctx bound to the command's lifetime,
// mirroring exec.CommandContext.
func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	name, args = adapt(name, args)
	return exec.CommandContext(ctx, name, args...)
}

// adapt returns the program and arguments to hand to os/exec, prefixing "cmd /C"
// when the target is a Windows batch file. only an explicit .cmd or .bat path is
// wrapped: a bare name such as "claude" is resolved by exec.LookPath through
// PATHEXT and executed directly, and a name that resolves to nothing must surface
// exec.ErrNotFound rather than a cmd.exe exit code.
func adapt(name string, args []string) (string, []string) {
	if goos != "windows" || !isBatchFile(name) {
		return name, args
	}
	return "cmd", append([]string{"/C", name}, args...)
}

// isBatchFile reports whether name has a .cmd or .bat extension (case-insensitive).
func isBatchFile(name string) bool {
	ext := filepath.Ext(name)
	return strings.EqualFold(ext, ".cmd") || strings.EqualFold(ext, ".bat")
}
