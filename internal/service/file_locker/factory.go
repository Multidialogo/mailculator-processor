package file_locker

import (
	"github.com/redis/go-redis/v9"
)

type Factory struct {
	driver      string
	redisClient *redis.Client
}

func NewFactory(driver string, redisClient *redis.Client) *Factory {
	return &Factory{
		driver:      driver,
		redisClient: redisClient,
	}
}

func (F *Factory) GetInstance(filePath string) Locker {
	if F.driver == "REDIS" {
		return NewRedisLocker(F.redisClient, filePath)
	} else {
		return NewFSLocker(filePath)
	}
}
