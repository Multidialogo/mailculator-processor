package email_client

// RawEmailInput represents the raw email message input
type RawEmailInput struct {
	Data []byte
}

// RawEmailOutput represents the result of sending a raw email
type RawEmailOutput struct {
	MessageID string
}

// EmailClient defines the interface for sending raw emails
type EmailClient interface {
	SendRawEmail(input *RawEmailInput) (*RawEmailOutput, error)
}
