package executor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestCommandFactory_Command(t *testing.T) {
	tests := []struct {
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
			name:       "windows with exe passes through",
			goos:       "windows",
			cmdName:    "git.exe",
			args:       []string{"status"},
			expectCmd:  "git.exe",
			expectArgs: []string{"status"},
		},
		{
			name:       "windows with no extension passes through",
			goos:       "windows",
			cmdName:    "claude",
			args:       []string{"-p", "test"},
			expectCmd:  "claude",
			expectArgs: []string{"-p", "test"},
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

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			old := goos
			goos = tc.goos
			t.Cleanup(func() { goos = old })

			f := CommandFactory{}
			cmd := f.Command(tc.cmdName, tc.args...)

			// cmd.Args[0] is the command name as passed, cmd.Path may be resolved via LookPath
			assert.Equal(t, tc.expectCmd, cmd.Args[0], "command name")
			assert.Equal(t, tc.expectArgs, cmd.Args[1:], "command arguments")
		})
	}
}

func TestCommandFactory_CommandContext(t *testing.T) {
	tests := []struct {
		name       string
		goos       string
		cmdName    string
		args       []string
		expectCmd  string   // expected cmd.Args[0]
		expectArgs []string // expected cmd.Args[1:]
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
			name:       "windows with exe passes through",
			goos:       "windows",
			cmdName:    "git.exe",
			args:       []string{"status"},
			expectCmd:  "git.exe",
			expectArgs: []string{"status"},
		},
		{
			name:       "windows with no extension passes through",
			goos:       "windows",
			cmdName:    "claude",
			args:       []string{"-p", "test"},
			expectCmd:  "claude",
			expectArgs: []string{"-p", "test"},
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

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			old := goos
			goos = tc.goos
			t.Cleanup(func() { goos = old })

			ctx := context.Background()
			f := CommandFactory{}
			cmd := f.CommandContext(ctx, tc.cmdName, tc.args...)

			// cmd.Args[0] is the command name as passed, cmd.Path may be resolved via LookPath
			assert.Equal(t, tc.expectCmd, cmd.Args[0], "command name")
			assert.Equal(t, tc.expectArgs, cmd.Args[1:], "command arguments")
		})
	}
}
