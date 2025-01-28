package utils

import (
	"os"
	"path/filepath"
)

// MoveFile ensures the destination directory exists and moves the file
func MoveFile(originPath string, destinationPath string) error {
	// Get the parent directory of the destination path
	destinationDir := filepath.Dir(destinationPath)

	// Ensure the destination directory exists
	err := os.MkdirAll(destinationDir, os.ModePerm)
	if err != nil {
		return err
	}

	// Move (or rename) the file
	err = os.Rename(originPath, destinationPath)
	if err != nil {
		return err
	}

	return nil
}
