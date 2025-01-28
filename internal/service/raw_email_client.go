package service

// RawEmailInput represents the raw email message input
type RawEmailInput struct {
	Data []byte
}

// RawEmailOutput represents the result of sending a raw email
type RawEmailOutput struct {
	MessageID string
}

// RawEmailClient defines the interface for sending raw emails
type RawEmailClient interface {
	SendRawEmail(input *RawEmailInput) (*RawEmailOutput, error)
}
