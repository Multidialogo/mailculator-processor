package outbox

const (
	StatusAccepted              = "ACCEPTED"
	StatusIntaking              = "INTAKING"
	StatusReady                 = "READY"
	StatusProcessing            = "PROCESSING"
	StatusSent                  = "SENT"
	StatusFailed                = "FAILED"
	StatusInvalid               = "INVALID"
	StatusCallingSentCallback   = "CALLING-SENT-CALLBACK"
	StatusCallingFailedCallback = "CALLING-FAILED-CALLBACK"
	StatusSentAcknowledged      = "SENT-ACKNOWLEDGED"
	StatusFailedAcknowledged    = "FAILED-ACKNOWLEDGED"
)

type Email struct {
	Id              string
	Status          string
	EmlFilePath     string
	PayloadFilePath string
	UpdatedAt       string
	Reason          string
	TTL             *int64
	Version         int
}
