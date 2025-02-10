package file_locker

type Locker interface {
	TryLock() (bool, error)
	Unlock() (bool, error)
}
