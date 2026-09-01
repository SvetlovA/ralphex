package execx

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setGOOS overrides the package-level goos for the duration of the test.
func setGOOS(t *testing.T, os string) {
	t.Helper()
	orig := goos
	goos = os
	t.Cleanup(func() { goos = orig })
}

func TestGOOSDefault(t *testing.T) {
	assert.Equal(t, runtime.GOOS, goos, "production must decide against the real platform")
}

func TestIsBatchFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"cmd extension", "claude.cmd", true},
		{"bat extension", "script.bat", true},
		{"CMD uppercase", "claude.CMD", true},
		{"BAT uppercase", "script.BAT", true},
		{"Cmd mixed case", "claude.Cmd", true},
		{"exe extension", "git.exe", false},
		{"no extension", "claude", false},
		{"empty string", "", false},
		{"dot in directory", "path.with.dots/claude.cmd", true},
		{"dot in directory no batch ext", "path.with.dots/claude.exe", false},
		{"cmd in path but not extension", "cmd/claude", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isBatchFile(tc.input))
		})
	}
}

// commandCases covers the wrapping decision; Command and CommandContext share it
// because they must make identical choices.
var commandCases = []struct { //nolint:gochecknoglobals // shared table for Command and CommandContext
	name       string
	goos       string
	cmdName    string
	args       []string
	expectCmd  string   // expected cmd.Args[0] (command name as passed)
	expectArgs []string // expected cmd.Args[1:] (arguments)
}{
	{
		name:       "windows with cmd file wraps with cmd /C",
		goos:       "windows",
		cmdName:    "claude.cmd",
		args:       []string{"--verbose", "-p", "hello"},
		expectCmd:  "cmd",
		expectArgs: []string{"/C", "claude.cmd", "--verbose", "-p", "hello"},
	},
	{
		name:       "windows with bat file wraps with cmd /C",
		goos:       "windows",
		cmdName:    "script.bat",
		args:       []string{"arg1"},
		expectCmd:  "cmd",
		expectArgs: []string{"/C", "script.bat", "arg1"},
	},
	{
		name:       "windows with absolute cmd path wraps with cmd /C",
		goos:       "windows",
		cmdName:    `C:\Program Files\nodejs\claude.cmd`,
		args:       []string{"-p"},
		expectCmd:  "cmd",
		expectArgs: []string{"/C", `C:\Program Files\nodejs\claude.cmd`, "-p"},
	},
	{
		name:       "windows with uppercase extension wraps with cmd /C",
		goos:       "windows",
		cmdName:    "claude.CMD",
		args:       nil,
		expectCmd:  "cmd",
		expectArgs: []string{"/C", "claude.CMD"},
	},
	{
		name:       "windows with exe passes through",
		goos:       "windows",
		cmdName:    "git.exe",
		args:       []string{"status"},
		expectCmd:  "git.exe",
		expectArgs: []string{"status"},
	},
	{
		// a bare name is resolved by exec.LookPath through PATHEXT, so an npm shim
		// installed as claude.cmd is found and executed without the cmd /C prefix
		name:       "windows with no extension passes through",
		goos:       "windows",
		cmdName:    "claude",
		args:       []string{"-p", "test"},
		expectCmd:  "claude",
		expectArgs: []string{"-p", "test"},
	},
	{
		// an unresolvable name must reach exec.Command so Start reports
		// exec.ErrNotFound instead of a cmd.exe "not recognized" exit code
		name:       "windows with unknown program passes through",
		goos:       "windows",
		cmdName:    "definitely-not-installed-tool",
		args:       []string{"arg1"},
		expectCmd:  "definitely-not-installed-tool",
		expectArgs: []string{"arg1"},
	},
	{
		name:       "linux with cmd file passes through",
		goos:       "linux",
		cmdName:    "claude.cmd",
		args:       []string{"--verbose"},
		expectCmd:  "claude.cmd",
		expectArgs: []string{"--verbose"},
	},
	{
		name:       "darwin with bat file passes through",
		goos:       "darwin",
		cmdName:    "script.bat",
		args:       []string{"arg1"},
		expectCmd:  "script.bat",
		expectArgs: []string{"arg1"},
	},
	{
		name:       "windows with no args",
		goos:       "windows",
		cmdName:    "claude.cmd",
		args:       nil,
		expectCmd:  "cmd",
		expectArgs: []string{"/C", "claude.cmd"},
	},
}

func TestCommand(t *testing.T) {
	for _, tc := range commandCases {
		t.Run(tc.name, func(t *testing.T) {
			setGOOS(t, tc.goos)
			cmd := Command(tc.cmdName, tc.args...)

			// cmd.Args[0] is the command name as passed, cmd.Path may be resolved via LookPath
			assert.Equal(t, tc.expectCmd, cmd.Args[0], "command name")
			assert.Equal(t, tc.expectArgs, cmd.Args[1:], "command arguments")
		})
	}
}

func TestCommandContext(t *testing.T) {
	for _, tc := range commandCases {
		t.Run(tc.name, func(t *testing.T) {
			setGOOS(t, tc.goos)
			cmd := CommandContext(t.Context(), tc.cmdName, tc.args...)

			assert.Equal(t, tc.expectCmd, cmd.Args[0], "command name")
			assert.Equal(t, tc.expectArgs, cmd.Args[1:], "command arguments")
		})
	}
}

func TestCommandDoesNotMutateCallerArgs(t *testing.T) {
	setGOOS(t, "windows")

	args := []string{"--verbose", "-p"}
	cmd := Command("claude.cmd", args...)

	require.Equal(t, []string{"--verbose", "-p"}, args, "caller slice must be untouched")
	assert.Equal(t, []string{"cmd", "/C", "claude.cmd", "--verbose", "-p"}, cmd.Args)
}
