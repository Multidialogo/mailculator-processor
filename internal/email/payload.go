package email

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-playground/validator/v10"
)

type AttachmentList []Attachment

type Attachment struct {
	Path string `json:"path" validate:"required,uri"`
	Name string `json:"name" validate:"required"`
}

func (a *AttachmentList) UnmarshalJSON(data []byte) error {
	var strings []string
	if err := json.Unmarshal(data, &strings); err == nil {
		*a = make([]Attachment, len(strings))
		for i, s := range strings {
			(*a)[i] = Attachment{
				Path: s,
				Name: filepath.Base(s),
			}
		}
		return nil
	}
	var attachments []Attachment
	if err := json.Unmarshal(data, &attachments); err != nil {
		return fmt.Errorf("attachments must be either array of strings or array of objects: %w", err)
	}
	*a = attachments
	return nil
}

type Payload struct {
	Id            string            `json:"id" validate:"required,uuid"`
	From          string            `json:"from" validate:"required,email"`
	ReplyTo       string            `json:"reply_to" validate:"required,email"`
	To            string            `json:"to" validate:"required,email"`
	Subject       string            `json:"subject" validate:"required"`
	BodyHTML      string            `json:"body_html" validate:"required_without=BodyText"`
	BodyText      string            `json:"body_text" validate:"required_without=BodyHTML"`
	Attachments   AttachmentList    `json:"attachments" validate:"dive"`
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
