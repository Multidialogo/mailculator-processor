package service

import (
	"fmt"
	"os"
)

// SendRawEmail sends a raw email using the provided RawEmailClient
func SendRawEmail(filePath string, client RawEmailClient) (error, *RawEmailOutput) {
	// Open the .EML file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open EML file: %v", err), nil
	}
	defer file.Close()

	// Read the file content into a byte slice
	emlContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read EML file: %v", err), nil
	}

	// Prepare the input for the raw email client
	input := &RawEmailInput{
		Data: emlContent,
	}

	// Send the email using the provided client
	result, err := client.SendRawEmail(input)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err), nil
	}

	return nil, result
}
