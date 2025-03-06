package email

type Email struct {
	Id              string
	Status          string
	EmlFilePath     string
	SuccessCallback string
	FailureCallback string
}

func (email Email) GetId() string {
	return email.Id
}

type Lock struct {
	Id string
}

func (lock Lock) GetId() string {
	return lock.Id
}
