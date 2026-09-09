//go:build windows

package executor

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// gracefulShutdownDelay is the time to wait between the first-stage stop attempt
// and the unconditional job termination that backs it up.
const gracefulShutdownDelay = 100 * time.Millisecond

// jobTerminateExitCode is reported by every process the job kills.
const jobTerminateExitCode = 1

// processGroupCleanup manages process lifecycle for graceful shutdown on Windows.
//
// Windows has no session/process-group equivalent and no orphan reaping: a killed
// parent leaves its descendants running with a dangling ppid. An npm-installed CLI
// always sits behind at least one cmd.exe shim (claude.cmd -> cmd.exe -> node.exe),
// so terminating cmd.cmd.Process reaches the shim and nothing below it. The real
// worker plus everything it spawned (Git Bash, find/grep/head, MCP servers) survives.
//
// A Job Object is the platform's tree primitive. Every descendant is assigned to the
// job automatically at creation, so terminating the job reaches the whole tree in one
// call. JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE additionally makes the kernel tear the tree
// down if ralphex exits without cleaning up, which no user-space path can guarantee.
//
// job is windows.InvalidHandle when the job could not be created; every operation
// degrades to direct-process behavior in that case rather than failing the run.
type processGroupCleanup struct {
	cmd      *exec.Cmd
	job      windows.Handle
	done     chan struct{}
	once     sync.Once // guards cmd.Wait() idempotency
	killOnce sync.Once // guards killProcess() idempotency
	jobOnce  sync.Once // guards job handle close idempotency
	err      error
}

// setupProcessGroup is a no-op on Windows: the job object is created after Start
// (see newProcessGroupCleanup), and CREATE_NEW_PROCESS_GROUP is deliberately NOT
// set. Without it the child tree stays attached to ralphex's console and still
// receives the console's CTRL_C_EVENT on interactive Ctrl+C, which is the one path
// that already tore the tree down correctly. Setting the flag would suppress that.
func setupProcessGroup(_ *exec.Cmd) {
	// job object assignment happens post-Start in newProcessGroupCleanup
}

// newProcessGroupCleanup creates a cleanup handler for the given command.
// The command must already be started before calling this.
// Caller must eventually call Wait() to ensure proper resource cleanup.
func newProcessGroupCleanup(cmd *exec.Cmd, cancelCh <-chan struct{}) *processGroupCleanup {
	pg := &processGroupCleanup{
		cmd:  cmd,
		job:  windows.InvalidHandle,
		done: make(chan struct{}),
	}

	// assign the started process to a kill-on-close job so descendants are reachable.
	// failure is not fatal: the run continues with direct-process semantics.
	job, err := assignToNewJob(cmd)
	if err != nil {
		log.Printf("[executor] job object setup failed, descendants may be orphaned: %v", err)
	} else {
		pg.job = job
	}

	// monitor for cancellation in background
	go pg.watchForCancel(cancelCh)

	return pg
}

// assignToNewJob creates an anonymous kill-on-close job object and assigns the
// already-started process to it. Descendants spawned afterwards join the job
// automatically. Returns the job handle, which the caller owns and must close.
//
// There is a narrow race between cmd.Start() and the assignment: a descendant
// spawned in that window escapes the job. Creating the process suspended would
// close it, but os/exec exposes no thread handle to resume, so the window is
// accepted - it is microseconds against a node startup measured in milliseconds.
func assignToNewJob(cmd *exec.Cmd) (windows.Handle, error) {
	if cmd.Process == nil {
		return windows.InvalidHandle, errors.New("process not started")
	}

	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return windows.InvalidHandle, fmt.Errorf("create job object: %w", err)
	}

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	//nolint:gosec // G103: unsafe.Pointer is how the SetInformationJobObject ABI takes its struct
	if _, setErr := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); setErr != nil {
		_ = windows.CloseHandle(job)
		return windows.InvalidHandle, fmt.Errorf("set job limits: %w", setErr)
	}

	// os.Process keeps its handle unexported, so reopen by pid. the live os.Process
	// handle pins the pid, so this cannot race onto a recycled process.
	proc, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		return windows.InvalidHandle, fmt.Errorf("open process %d: %w", cmd.Process.Pid, err)
	}
	defer func() { _ = windows.CloseHandle(proc) }()

	if err := windows.AssignProcessToJobObject(job, proc); err != nil {
		_ = windows.CloseHandle(job)
		return windows.InvalidHandle, fmt.Errorf("assign process to job: %w", err)
	}

	return job, nil
}

// watchForCancel monitors the cancel channel and kills the process tree if triggered.
func (pg *processGroupCleanup) watchForCancel(cancelCh <-chan struct{}) {
	select {
	case <-cancelCh:
		pg.killOnce.Do(pg.killProcess)
	case <-pg.done:
		// process completed normally, goroutine exits
	}
}

// killProcess terminates the child and every descendant it spawned, in two stages
// mirroring the Unix SIGTERM-then-SIGKILL shape.
//
// Windows has no signal that asks a process to shut down: TerminateJobObject is an
// unconditional TerminateProcess on every member, with no unwind and no flush. The
// closest thing to a request is dropping the direct child, which breaks the stdio
// pipes the tree writes through - node exits on the resulting EPIPE, giving claude a
// chance to finish its session transcript and release its ~/.claude locks. The job
// termination then backs that up for anything that ignored the hint.
//
// Only reached on abnormal termination (idle_timeout, session_timeout, cancellation),
// so the delay is not paid per iteration - Wait's post-exit reap calls terminateJob
// directly. Safe when the job is InvalidHandle and when the process has already
// exited; called at most once, guarded by killOnce.
func (pg *processGroupCleanup) killProcess() {
	// stage 1: drop the direct child, breaking the tree's stdio pipes
	if pg.cmd.Process != nil {
		_ = pg.cmd.Process.Kill()
	}

	if pg.job == windows.InvalidHandle {
		return // no job: the direct kill was all we had
	}

	// give the tree a beat to notice and unwind on its own
	time.Sleep(gracefulShutdownDelay)

	// stage 2: unconditional backstop for whatever is left
	pg.terminateJob()
}

// terminateJob kills every process still in the job, with no grace period. Used
// directly by the post-exit reap, where the direct child has already exited and a
// grace stage would buy nothing but latency on every iteration. No-op when the job
// could not be created.
func (pg *processGroupCleanup) terminateJob() {
	if pg.job == windows.InvalidHandle {
		return
	}
	if err := windows.TerminateJobObject(pg.job, jobTerminateExitCode); err != nil {
		log.Printf("[executor] terminate job object: %v", err)
	}
}

// closeJob releases the job handle. KILL_ON_JOB_CLOSE means any member still alive
// at this point is killed by the kernel, which is the backstop for descendants that
// outlived the direct child on a normal exit.
func (pg *processGroupCleanup) closeJob() {
	pg.jobOnce.Do(func() {
		if pg.job == windows.InvalidHandle {
			return
		}
		if err := windows.CloseHandle(pg.job); err != nil {
			log.Printf("[executor] closing job object handle: %v", err)
		}
		pg.job = windows.InvalidHandle
	})
}

// Wait waits for the command to complete and cleans up resources.
// It is safe to call multiple times - subsequent calls return the cached result.
// Callers must eventually call Wait to avoid leaking resources.
//
// After the command exits, the job is torn down to reap descendants that survived
// the direct child (node subagents, MCP servers, Git Bash and its find/grep/head
// children). This mirrors the post-exit reap the Unix implementation performs; the
// previous Windows code skipped it on the reasoning that cmd.Wait() returning made a
// further kill a no-op, which holds for the direct process but not for its tree.
func (pg *processGroupCleanup) Wait() error {
	pg.once.Do(func() {
		pg.err = pg.cmd.Wait()
		close(pg.done)
		// the direct child has already exited, so killProcess's grace stage would add
		// latency to every iteration for no benefit. reap the tree directly instead.
		// sharing killOnce means a concurrent cancellation runs exactly one of the two:
		// whichever wins, the other sees the tree already terminated.
		pg.killOnce.Do(pg.terminateJob)
		pg.closeJob()
		if pg.err != nil {
			pg.err = fmt.Errorf("command wait: %w", pg.err)
		}
	})
	return pg.err
}
