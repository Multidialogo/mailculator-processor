package service

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

// SendEMLFile reads an .EML file and sends it using AWS SES.
func SendEMLFile(filePath string) error {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %v", err)
	}

	// Create SES client
	client := ses.NewFromConfig(cfg)

	// Open the .EML file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open EML file: %v", err)
	}
	defer file.Close()

	// Read the file content into a byte slice
	emlContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read EML file: %v", err)
	}

	// Prepare the SES input with the raw message
	input := &ses.SendRawEmailInput{
		RawMessage: &types.RawMessage{
			Data: emlContent,
		},
	}

	// Send the email using SES
	result, err := client.SendRawEmail(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to send email via SES: %v", err)
	}
	if result == nil || result.MessageId == nil {
		return fmt.Errorf("failed to send email: no message ID returned")
	}

	return nil
}
