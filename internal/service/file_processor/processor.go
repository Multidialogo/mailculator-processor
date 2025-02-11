package file_processor

import (
	"fmt"
	"io"
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

func (fp *FileProcessor) SendRawEmail(filePath string) (*email_client.RawEmailOutput, error) {
	// Ensure file exists before proceeding
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	// Create and attempt to acquire file lock
	lock := fp.fileLockerFactory.GetInstance(filePath)
	locked, lockErr := lock.TryLock()
	if lockErr != nil {
		return nil, fmt.Errorf("error locking file: %w", lockErr)
	}
	if !locked {
		return nil, fmt.Errorf("file is already locked: %s", filePath)
	}
	// Ensure the lock is released when the function exits
	defer lock.Unlock()

	// Open the .EML file
	file, openErr := os.Open(filePath)
	if openErr != nil {
		return nil, fmt.Errorf("failed to open EML file: %w", openErr)
	}
	defer file.Close()

	// Read file contents efficiently
	emlContent, readErr := io.ReadAll(file)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read EML file: %w", readErr)
	}

	// Send the email using the provided client
	result, sendErr := fp.emailClient.SendRawEmail(&email_client.RawEmailInput{Data: emlContent})
	if sendErr != nil {
		return nil, fmt.Errorf("failed to send email: %w", sendErr)
	}

	return result, nil
}
