package file_locker

type Factory struct {
	driver string
}

func NewFactory(driver string) *Factory {
	return &Factory{driver: driver}
}

func (F *Factory) GetInstance(filePath string) Locker {
	if F.driver == "REDIS" {
		return NewRedisLocker(filePath)
	} else {
		return NewFSLocker(filePath)
	}
}
