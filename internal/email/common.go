package email

type Email struct {
	Id              string
	Status          string
	To              string
	EmlFilePath     string
	SuccessCallback string
	FailureCallback string
}
