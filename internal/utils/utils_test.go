package utils

import (
	"os"
	"testing"
	"path/filepath"
)

// TestMoveFile tests the MoveFile function
func TestMoveFile(t *testing.T) {
	// Create a temporary directory to store the test files
	tmpDir := t.TempDir()

	// Test data
	sourceFile := filepath.Join(tmpDir, "source.txt")
	destinationDir := filepath.Join(tmpDir, "destination")

	// Create a file in the source directory
	err := os.WriteFile(sourceFile, []byte("Test file content"), 0644)
	if err != nil {
		t.Fatalf("Error creating source file: %v", err)
	}

	// Move the file to the destination
	err = MoveFile(sourceFile, destinationDir)
	if err != nil {
		t.Fatalf("Failed to move file: %v", err)
	}

	// Check that the file was moved to the destination
	destPath := filepath.Join(destinationDir, "source.txt")
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Errorf("Destination file does not exist: %v", destPath)
	}

	// Test moving a non-existing file
	err = MoveFile("nonexistent.txt", destinationDir)
	if err == nil {
		t.Errorf("Expected error for non-existent file, but got none")
	}
}

// TestMoveFileCreateDir tests the case where the destination directory doesn't exist yet
func TestMoveFileCreateDir(t *testing.T) {
	// Create a temporary directory to store the test files
	tmpDir := t.TempDir()

	// Test data
	sourceFile := filepath.Join(tmpDir, "source.txt")
	destinationDir := filepath.Join(tmpDir, "newdir", "destination")

	// Create a file in the source directory
	err := os.WriteFile(sourceFile, []byte("Test file content"), 0644)
	if err != nil {
		t.Fatalf("Error creating source file: %v", err)
	}

	// Move the file to the destination
	err = MoveFile(sourceFile, destinationDir)
	if err != nil {
		t.Fatalf("Failed to move file: %v", err)
	}

	// Check that the file was moved to the destination
	destPath := filepath.Join(destinationDir, "source.txt")
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Errorf("Destination file does not exist: %v", destPath)
	}
}
