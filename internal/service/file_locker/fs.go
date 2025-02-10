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

func (fsl *FSLocker) TryLock() (bool, error) {
	locked, err := fsl.lock.TryLock()
	if err != nil {
		// If an error occurred, print it
		return false, fmt.Errorf("Error locking file: %v", err)
	}

	if !locked {
		return false, nil
	}

	return true, nil
}

func (fsl *FSLocker) Unlock() (bool, error) {
	// TODO: check for error on Unlock if needed
	fsl.lock.Unlock()

	return true, nil
}
