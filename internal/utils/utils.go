package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func MoveFile(originPath string, destDir string) error {
	if _, err := os.Stat(originPath); os.IsNotExist(err) {
		return fmt.Errorf("Missing origin file: %s", originPath)
	}

	// Clean the destination directory path
	destDir = filepath.Clean(destDir)

	// Create destination directory if it doesn't exist
	err := os.MkdirAll(destDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Cannot create destination directory: %v", err)
	}

	// Extract the file name from the origin path
	fileName := filepath.Base(originPath)
	destPath := filepath.Join(destDir, fileName)

	// Move the file to the destination path
	err = os.Rename(originPath, destPath)
	if err != nil {
		return fmt.Errorf("Cannot move the file: %v", err)
	}

	return nil
}
