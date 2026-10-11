//go:build windows

package executor

import (
	"bufio"
	"context"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

// stillActive is the exit code GetExitCodeProcess reports for a running process.
const stillActive = 259

func TestExecClaudeRunner_KillsDescendantsOnCancel(t *testing.T) {
	// on Windows every npm-installed CLI sits behind a cmd.exe shim, so the direct
	// child is never the process doing the work. verify that canceling the context
	// reaches a grandchild the direct child spawned, not just the direct child.

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	runner := &execClaudeRunner{}

	// powershell spawns a detached grandchild, prints its PID, then blocks.
	// Start-Process makes the grandchild survive the direct child's death, so only
	// job-object termination can reach it.
	stdout, wait, err := runner.Run(ctx, "powershell", "-NoProfile", "-Command", grandchildScript+"Start-Sleep 300")
	require.NoError(t, err)

	childPID := readWindowsChildPID(t, stdout)
	require.NotZero(t, childPID, "should capture grandchild PID from output")
	t.Cleanup(func() { killPID(childPID) })

	require.True(t, processAlive(childPID), "grandchild should be running before cancel")

	cancel()

	// wait errors because the process was killed, which is expected
	_ = wait()

	require.Eventually(t, func() bool {
		return !processAlive(childPID)
	}, 5*time.Second, 50*time.Millisecond,
		"grandchild (PID %d) should be killed when the job object is terminated", childPID)
}

func TestExecClaudeRunner_KillsOrphansOnNormalExit(t *testing.T) {
	// the pre-job implementation skipped the post-exit reap on the reasoning that
	// cmd.Wait() returning made a further kill a no-op. that holds for the direct
	// process and is false for its tree, which is how stale find/grep/head processes
	// accumulated across a run. verify Wait() now reaps the orphan.

	runner := &execClaudeRunner{}

	// spawn the detached grandchild, print its PID, then exit immediately, leaving
	// the grandchild orphaned.
	stdout, wait, err := runner.Run(t.Context(), "powershell", "-NoProfile", "-Command", grandchildScript+"exit 0")
	require.NoError(t, err)

	childPID := readWindowsChildPID(t, stdout)
	require.NotZero(t, childPID, "should capture grandchild PID from output")
	t.Cleanup(func() { killPID(childPID) })

	require.True(t, processAlive(childPID), "grandchild should be running before wait")

	require.NoError(t, wait(), "parent should exit cleanly")

	require.Eventually(t, func() bool {
		return !processAlive(childPID)
	}, 5*time.Second, 50*time.Millisecond,
		"orphaned grandchild (PID %d) should be reaped after normal parent exit", childPID)
}

func TestProcessGroupCleanup_WaitIdempotent(t *testing.T) {
	// Wait() closes the job handle and terminates the tree; verify repeated calls
	// neither panic nor double-close.

	runner := &execClaudeRunner{}

	stdout, wait, err := runner.Run(t.Context(), "cmd", "/c", "echo hello")
	require.NoError(t, err)
	_, _ = io.ReadAll(stdout)

	err1, err2, err3 := wait(), wait(), wait()

	assert.Equal(t, err1, err2, "repeated Wait() calls should return same error")
	assert.Equal(t, err2, err3, "repeated Wait() calls should return same error")
}

// grandchildScript spawns a detached ping that outlives its parent and prints its
// PID in the CHILD_PID:<n> form readWindowsChildPID looks for. Callers append the
// statement that decides whether the parent blocks or exits.
const grandchildScript = `$p = Start-Process ping -ArgumentList '-n','300','127.0.0.1' -PassThru -WindowStyle Hidden; ` +
	`Write-Output "CHILD_PID:$($p.Id)"; `

// readWindowsChildPID scans output for the CHILD_PID:<n> marker and returns the pid.
func readWindowsChildPID(t *testing.T, r io.Reader) int {
	t.Helper()

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		pidStr, ok := strings.CutPrefix(line, "CHILD_PID:")
		if !ok {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(pidStr))
		require.NoError(t, err, "parse child PID from %q", line)
		return pid
	}
	return 0
}

// processAlive reports whether pid names a process that has not yet exited.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer func() { _ = windows.CloseHandle(h) }()

	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == stillActive
}

// killPID is a best-effort cleanup so a failed assertion does not leak the grandchild.
func killPID(pid int) {
	if pid <= 0 {
		return
	}
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return
	}
	defer func() { _ = windows.CloseHandle(h) }()
	_ = windows.TerminateProcess(h, 1)
}
