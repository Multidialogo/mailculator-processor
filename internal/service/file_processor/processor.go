package file_processor

import (
	"fmt"
	"os"

	"mailculator-processor/internal/service/email_client"
	"mailculator-processor/internal/service/file_locker"
)

type FileProcessor struct {
	emailClient       email_client.EmailClient
	fileLockerFactory *file_locker.Factory
}

func NewFileProcessor(emailClient email_client.EmailClient, fileLockerFactory *file_locker.Factory) *FileProcessor {
	return &FileProcessor{
		emailClient:       emailClient,
		fileLockerFactory: fileLockerFactory,
	}
}

func (fp *FileProcessor) SendRawEmail(filePath string) (error, *email_client.RawEmailOutput) {
	// Create a flock instance for the lock file
	lock := fp.fileLockerFactory.GetInstance(filePath)

	// Try to acquire an exclusive lock on the file
	locked, err := lock.TryLock()
	if err != nil {
		// If an error occurred, print it
		return fmt.Errorf("Error locking file: %v", err), nil
	}

	// If the file is locked, skip processing
	if !locked {
		return fmt.Errorf("File locked: %v", err), nil
	}

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
	input := &email_client.RawEmailInput{
		Data: emlContent,
	}

	// Send the email using the provided client
	result, err := fp.emailClient.SendRawEmail(input)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err), nil
	}

	lock.Unlock()

	return nil, result
}
