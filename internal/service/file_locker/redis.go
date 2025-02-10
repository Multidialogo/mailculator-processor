package file_locker

import (
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
)

type RedisLocker struct {
	redisMutex *redsync.Mutex
	fsLocker   *FSLocker
}

func NewRedisLocker(redisClient *redis.Client, filePath string) *RedisLocker {
	pool := goredis.NewPool(redisClient)
	rs := redsync.New(pool)

	mutex := rs.NewMutex("file-lock:" + filePath)

	return &RedisLocker{
		redisMutex: mutex,
		fsLocker:   NewFSLocker(filePath),
	}
}

func (rl *RedisLocker) TryLock() (bool, error) {
	if err := rl.redisMutex.Lock(); err != nil {
		return false, fmt.Errorf("failed to acquire Redis lock: %v", err)
	}

	fsLock, err := rl.fsLocker.TryLock()
	if err != nil {
		return false, err
	}
	if !fsLock {
		return false, nil
	}

	return true, nil
}

func (rl *RedisLocker) Unlock() (bool, error) {
	fsLock, err := rl.fsLocker.Unlock()
	if err != nil {
		return false, err
	}
	if !fsLock {
		return false, nil
	}

	_, err = l.redisMutex.Unlock()
	if err != nil {
		return false, err
	}

	return true, nil
}
