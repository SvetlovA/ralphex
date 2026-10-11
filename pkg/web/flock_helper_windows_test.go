//go:build windows

package web

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

// matches the sentinel-byte range used by pkg/progress/flock_windows.go.
const (
	testLockOffsetLow  uint32 = 0
	testLockOffsetHigh uint32 = 0x7fffffff
	testLockBytesLow   uint32 = 1
	testLockBytesHigh  uint32 = 0
)

// holdFileLockForTest opens path and acquires a blocking exclusive lock on the
// sentinel byte used by progress.IsActive, returning a release function that
// unlocks and closes the file. mirrors the unix helper so watcher tests that
// need IsActive(path) to report true can run on Windows now that LockFileEx
// is wired up.
func holdFileLockForTest(t *testing.T, path string) func() {
	t.Helper()
	f, err := os.OpenFile(path, os.O_RDWR, 0o600) //nolint:gosec // test-controlled path from t.TempDir
	require.NoError(t, err)
	ol := &windows.Overlapped{Offset: testLockOffsetLow, OffsetHigh: testLockOffsetHigh}
	require.NoError(t, windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK,
		0,
		testLockBytesLow, testLockBytesHigh,
		ol,
	))
	return func() {
		ol := &windows.Overlapped{Offset: testLockOffsetLow, OffsetHigh: testLockOffsetHigh}
		_ = windows.UnlockFileEx(
			windows.Handle(f.Fd()),
			0,
			testLockBytesLow, testLockBytesHigh,
			ol,
		)
		_ = f.Close()
	}
}
