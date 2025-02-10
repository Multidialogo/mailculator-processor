package file_locker

import (
	"github.com/redis/go-redis/v9"
)

type Factory struct {
	driver string
}

func NewFactory(driver string) *Factory {
	return &Factory{driver: driver}
}

func (F *Factory) GetInstance(filePath string, redisClient *redis.Client) Locker {
	if F.driver == "REDIS" {
		return NewRedisLocker(redisClient, filePath)
	} else {
		return NewFSLocker(filePath)
	}
}
