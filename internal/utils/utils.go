package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func MoveFile(filePath string, destination string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("Missing origin file: %s", filePath)
	}

	// Create destination directory if it doesn't exist
	err := os.MkdirAll(destination, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Cannot create destination directory: %v", err)
	}

	// Create the destination path
	destPath := filepath.Join(destination, filepath.Base(filePath))

	// Move the file to the destination path
	err = os.Rename(filePath, destPath)
	if err != nil {
		return fmt.Errorf("Cannot move the file: %v", err)
	}

	// Ensure the file exists at the new location
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		return fmt.Errorf("Failed to verify destination file: %s", destPath)
	}

	return nil
}
