package file_locker

type RedisLocker struct {
}

func NewRedisLocker(filePath string) *RedisLocker {
	return &RedisLocker{}
}

func (L *RedisLocker) TryLock() (bool, error) {
	// TODO: implement

	return true, nil
}

func (L *RedisLocker) Unlock() (bool, error) {
	// TODO: implement

	return true, nil
}
