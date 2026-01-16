package email

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
)

type Payload struct {
	Id            string            `json:"id" validate:"required,uuid"`
	From          string            `json:"from" validate:"required,email"`
	ReplyTo       string            `json:"reply_to" validate:"required,email"`
	To            string            `json:"to" validate:"required,email"`
	Subject       string            `json:"subject" validate:"required"`
	BodyHTML      string            `json:"body_html" validate:"required_without=BodyText"`
	BodyText      string            `json:"body_text" validate:"required_without=BodyHTML"`
	Attachments   []string          `json:"attachments" validate:"dive,uri"`
	CustomHeaders map[string]string `json:"custom_headers"`
}

func LoadPayload(path string) (Payload, error) {
	payloadData, err := os.ReadFile(path)
	if err != nil {
		return Payload{}, fmt.Errorf("failed to read payload file %s: %w", path, err)
	}

	var payload Payload
	if err := json.Unmarshal(payloadData, &payload); err != nil {
		return Payload{}, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(payload); err != nil {
		return Payload{}, fmt.Errorf("payload validation failed: %w", err)
	}

	return payload, nil
}
