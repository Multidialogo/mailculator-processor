package file_locker

import (
	"fmt"

	"github.com/gofrs/flock"
)

type FSLocker struct {
	lock *flock.Flock
}

func NewFSLocker(filePath string) *FSLocker {
	return &FSLocker{
		lock: flock.New(filePath),
	}
}

func (L *FSLocker) TryLock() (bool, error) {
	// Try to acquire an exclusive lock on the file
	locked, err := L.lock.TryLock()
	if err != nil {
		// If an error occurred, print it
		return false, fmt.Errorf("Error locking file: %v", err)
	}

	// If the file is locked, skip processing
	if !locked {
		return false, nil
	}

	return true, nil
}

func (L *FSLocker) Unlock() (bool, error) {
	// FIXME: check for error on Unlock
	L.lock.Unlock()

	return true, nil
}
