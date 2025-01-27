package utils

import (
	"fmt"
	"os"
)

func MoveFile(originPath string, destPath string) error {
	if _, err := os.Stat(originPath); os.IsNotExist(err) {
		return fmt.Errorf("Missing origin file: %s", originPath)
	}

	// Create destination directory if it doesn't exist
	err := os.MkdirAll(destPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Cannot create destination directory: %v", err)
	}

	// Move the file to the destination path
	err = os.Rename(originPath, destPath)
	if err != nil {
		return fmt.Errorf("Cannot move the file: %v", err)
	}

	return nil
}
