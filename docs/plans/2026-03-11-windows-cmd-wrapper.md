# Windows .cmd/.bat command wrapper

## Overview

- On Windows, executables installed via npm (like `claude`, `codex`) create `.cmd` shim files that require `cmd /C` prefix to run correctly via `exec.Command`
- Create a `CommandFactory` type in `pkg/executor/` that wraps `exec.Command` and `exec.CommandContext`, automatically prepending `cmd /C` when running `.cmd`/`.bat` files on Windows
- Update all production `exec.Command`/`exec.CommandContext` call sites across the project to use the new wrapper

## Context (from discovery)

Files with `exec.Command`/`exec.CommandContext` calls (production code only):
- `pkg/executor/executor.go:63` — `exec.Command(name, args...)` for Claude CLI
- `pkg/executor/codex.go:37` — `exec.Command(name, args...)` for Codex CLI
- `pkg/executor/custom.go:29` — `exec.Command(script, promptFile)` for custom review scripts
- `pkg/git/external.go:31,57,127,149,173,299,413,479,490` — `exec.CommandContext(...)` for git operations
- `pkg/input/input.go:133,389` — `exec.CommandContext(...)` for fzf and editor
- `pkg/plan/plan.go:109` — `exec.CommandContext(...)` for fzf
- `pkg/notify/custom.go:28` — `exec.CommandContext(...)` for notification scripts

Existing platform-specific patterns:
- `pkg/executor/procgroup_unix.go` / `procgroup_windows.go` — build tag pairs for syscall-dependent code
- `pkg/progress/flock_unix.go` / `flock_windows.go` — build tag pairs for file locking
- This feature uses `runtime.GOOS` (no platform-specific imports needed), so a single file is appropriate

No import cycles: `pkg/executor` only imports `pkg/status`, so `pkg/git`, `pkg/input`, `pkg/plan`, and `pkg/notify` can safely import it.

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- Make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
  - tests are not optional - they are a required part of the checklist
  - write unit tests for new functions/methods
  - add new test cases for new code paths
  - tests cover both success and error scenarios
- **CRITICAL: all tests must pass before starting next task** - no exceptions
- **CRITICAL: update this plan file when scope changes during implementation**
- Run tests after each change
- Maintain backward compatibility

## Testing Strategy

- **Unit tests**: required for every task (see Development Approach above)
- Use table-driven tests with testify (project convention)
- Use `t.Helper()` in test helpers
- Override `runtime.GOOS` via package-level variable for cross-platform test coverage

## Progress Tracking

- Mark completed items with `[x]` immediately when done
- Add newly discovered tasks with ➕ prefix
- Document issues/blockers with ⚠️ prefix
- Update plan if implementation deviates from original scope
- Keep plan in sync with actual work done

## Implementation Steps

### Task 1: Create CommandFactory type with Command and CommandContext methods

- [x] create `pkg/executor/command.go` with `CommandFactory` struct
- [x] add package-level `var goos = runtime.GOOS` for test overriding
- [x] implement `isBatchFile(name string) bool` helper — checks `filepath.Ext` for `.cmd`/`.bat` (case-insensitive)
- [x] implement `Command(name string, args ...string) *exec.Cmd` method — on Windows + batch file: returns `exec.Command("cmd", "/C", name, args...)`, otherwise passthrough to `exec.Command(name, args...)`
- [x] implement `CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd` method — same logic with `exec.CommandContext`
- [x] add comments following project style (lowercase except godoc)
- [x] write table-driven tests in `pkg/executor/command_test.go` for `isBatchFile` (`.cmd`, `.bat`, `.exe`, `""`, mixed case)
- [x] write table-driven tests for `Command` method: Windows+.cmd wraps with cmd /C, Windows+.exe passes through, Linux+.cmd passes through
- [x] write table-driven tests for `CommandContext` method: same cases as Command, verify context is passed through
- [x] override `goos` variable in tests to test both Windows and non-Windows paths
- [x] run `make test` and `make lint` — must pass before next task

### Task 2: Update pkg/executor call sites (executor.go, codex.go, custom.go)

- [x] add `cmdFactory CommandFactory` field (or use zero-value directly) in relevant executor structs, or use package-level `CommandFactory{}` instance
- [x] update `pkg/executor/executor.go:63` — replace `exec.Command(name, args...)` with `CommandFactory{}.Command(name, args...)`
- [x] update `pkg/executor/codex.go:37` — replace `exec.Command(name, args...)` with `CommandFactory{}.Command(name, args...)`
- [x] update `pkg/executor/custom.go:29` — replace `exec.Command(script, promptFile)` with `CommandFactory{}.Command(script, promptFile)`
- [x] verify existing tests in `pkg/executor/` still pass (they use mocked runners, not real exec)
- [x] run `make test` and `make lint` — must pass before next task

### Task 3: Update pkg/git/external.go call sites

- [x] import `pkg/executor` in `pkg/git/external.go`
- [x] add `cmdFactory executor.CommandFactory` field to `externalBackend` struct (or use zero-value inline)
- [x] update `newExternalBackend` (line 31) — replace `exec.CommandContext(...)` with `CommandFactory{}.CommandContext(...)`
- [x] update `run` method (line 57) — replace `exec.CommandContext(...)` with `CommandFactory{}.CommandContext(...)`
- [x] update `hasCommits` (line 127) — same replacement
- [x] update `currentBranch` (line 149) — same replacement
- [x] update `getDefaultBranch` (line 173) — same replacement
- [x] update `isIgnored` (line 299) — same replacement
- [x] update `diffStats` (line 413) — same replacement
- [x] update `resolveRef` fallback (line 479) — same replacement
- [x] update `refExists` (line 490) — same replacement
- [x] verify existing tests in `pkg/git/` still pass
- [x] run `make test` and `make lint` — must pass before next task

### Task 4: Update pkg/input, pkg/plan, pkg/notify call sites

- [x] update `pkg/input/input.go:133` (fzf) — replace `exec.CommandContext(ctx, "fzf", ...)` with `CommandFactory{}.CommandContext(ctx, "fzf", ...)`
- [x] update `pkg/input/input.go:389` (editor) — replace `exec.CommandContext(ctx, editorPath, args...)` with `CommandFactory{}.CommandContext(ctx, editorPath, args...)`
- [x] update `pkg/plan/plan.go:109` (fzf) — replace `exec.CommandContext(ctx, "fzf", ...)` with `CommandFactory{}.CommandContext(ctx, "fzf", ...)`
- [x] update `pkg/notify/custom.go:28` (script) — replace `exec.CommandContext(ctx, c.scriptPath)` with `CommandFactory{}.CommandContext(ctx, c.scriptPath)`
- [x] verify existing tests in `pkg/input/`, `pkg/plan/`, `pkg/notify/` still pass
- [x] run `make test` and `make lint` — must pass before next task

### Task 5: Verify acceptance criteria

- [x] verify all `exec.Command` and `exec.CommandContext` production call sites are updated (grep to confirm none remain outside tests)
- [x] verify edge cases are handled: empty extension, uppercase `.CMD`, path with dots in directory name
- [x] run full test suite: `make test`
- [x] run linter: `make lint` — all issues must be fixed
- [x] run `make fmt` — code is properly formatted
- [x] cross-compile verify: `GOOS=windows GOARCH=amd64 go build ./...`
- [x] verify test coverage for `command.go` meets 80%+

### Task 6: [Final] Update documentation

- [ ] update `CLAUDE.md` Platform Support section to mention `.cmd`/`.bat` wrapper
- [ ] add `CommandFactory` to the Key Patterns section in `CLAUDE.md` if appropriate

## Technical Details

**Type design:**
```go
// CommandFactory creates exec.Cmd instances with platform-appropriate adaptations.
// on Windows, .cmd and .bat files are automatically wrapped with cmd /C.
type CommandFactory struct{}
```

**Adaptation logic:**
- Check `runtime.GOOS == "windows"` (via overridable package var for testing)
- Check `filepath.Ext(name)` for `.cmd` or `.bat` (case-insensitive via `strings.EqualFold`)
- If both conditions met: `exec.Command("cmd", append([]string{"/C", name}, args...)...)`
- Otherwise: passthrough to `exec.Command(name, args...)`

**Test override pattern:**
```go
var goos = runtime.GOOS // package-level, overridden in tests

func TestCommand_WindowsBatch(t *testing.T) {
    old := goos
    goos = "windows"
    t.Cleanup(func() { goos = old })
    // ...
}
```

**Files not updated (test helpers only):**
- `pkg/git/external_test.go` — `runGit` test helper (runs real git in temp dirs)
- `cmd/ralphex/main_test.go` — `runGit` and `branchExists` test helpers
- `e2e/e2e_test.go` — `buildBinary` and `startServer` test helpers

## Post-Completion

**Manual verification:**
- Test on actual Windows machine with npm-installed `claude.cmd` to verify the wrapper works end-to-end
- Verify that non-.cmd executables (e.g., `git.exe`, `fzf.exe`) are not affected by the wrapper