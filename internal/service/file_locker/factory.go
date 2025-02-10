package file_locker

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
