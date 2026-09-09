//go:build windows

package progress

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// active session detection on Windows uses LockFileEx on a sentinel byte at
// offset 2^63-1 (high uint32 of overlapped offset = 0x7fffffff). Windows file
// locks are mandatory: a lock on a byte range blocks reads/writes from any
// other handle to that range, including handles opened by the same process.
// locking the entire file would therefore block the Tailer (which uses a
// separate handle to read progress lines while the Logger holds the lock).
// the sentinel byte is well past any plausible file size, so locking it
// provides advisory-style coordination without interfering with content I/O.
const (
	lockSentinelOffsetLow  uint32 = 0
	lockSentinelOffsetHigh uint32 = 0x7fffffff
	lockSentinelBytesLow   uint32 = 1
	lockSentinelBytesHigh  uint32 = 0
)

func newSentinelOverlapped() *windows.Overlapped {
	return &windows.Overlapped{
		Offset:     lockSentinelOffsetLow,
		OffsetHigh: lockSentinelOffsetHigh,
	}
}

// lockFile acquires a blocking exclusive lock on a sentinel byte of the file.
// the lock is automatically released when the underlying file handle is closed.
func lockFile(f *os.File) error {
	if err := windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK,
		0,
		lockSentinelBytesLow, lockSentinelBytesHigh,
		newSentinelOverlapped(),
	); err != nil {
		return fmt.Errorf("LockFileEx: %w", err)
	}
	return nil
}

// unlockFile releases the sentinel-byte lock acquired by lockFile.
func unlockFile(f *os.File) error {
	if err := windows.UnlockFileEx(
		windows.Handle(f.Fd()),
		0,
		lockSentinelBytesLow, lockSentinelBytesHigh,
		newSentinelOverlapped(),
	); err != nil {
		return fmt.Errorf("UnlockFileEx: %w", err)
	}
	return nil
}

// TryLockFile attempts a non-blocking exclusive lock on the sentinel byte.
// Returns (true, nil) if lock acquired (and immediately released), (false, nil)
// if another handle already holds the lock.
func TryLockFile(f *os.File) (bool, error) {
	err := windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		lockSentinelBytesLow, lockSentinelBytesHigh,
		newSentinelOverlapped(),
	)
	if err != nil {
		// ERROR_LOCK_VIOLATION: another handle holds the lock.
		// ERROR_IO_PENDING: rare under LOCKFILE_FAIL_IMMEDIATELY but treated
		// the same — caller is told the file is locked by someone else.
		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) || errors.Is(err, windows.ERROR_IO_PENDING) {
			return false, nil
		}
		return false, fmt.Errorf("LockFileEx: %w", err)
	}
	// got the lock; release it immediately so the caller's TryLock semantics
	// match the unix flock(LOCK_EX|LOCK_NB) + LOCK_UN pattern.
	if unlockErr := windows.UnlockFileEx(
		windows.Handle(f.Fd()),
		0,
		lockSentinelBytesLow, lockSentinelBytesHigh,
		newSentinelOverlapped(),
	); unlockErr != nil {
		return true, fmt.Errorf("UnlockFileEx: %w", unlockErr)
	}
	return true, nil
}
