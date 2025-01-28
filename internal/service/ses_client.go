package service

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

// SESClient is a specific implementation of RawEmailClient for SES
type SESClient struct {
	client *ses.Client
}

// NewSESClient initializes and returns a new SESClient
func NewSESClient() (*SESClient, error) {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config, %v", err)
	}

	// Create SES client from AWS config
	client := ses.NewFromConfig(cfg)

	return &SESClient{client: client}, nil
}

func (s *SESClient) SendRawEmail(input *RawEmailInput) (*RawEmailOutput, error) {
	// Prepare the SES input with the raw message
	sesInput := &ses.SendRawEmailInput{
		RawMessage: &types.RawMessage{
			Data: input.Data,
		},
	}

	// Send the email using SES
	result, err := s.client.SendRawEmail(context.TODO(), sesInput)
	if err != nil {
		return nil, err
	}

	// Return the result as RawEmailOutput
	return &RawEmailOutput{
		MessageID: *result.MessageId,
	}, nil
}
