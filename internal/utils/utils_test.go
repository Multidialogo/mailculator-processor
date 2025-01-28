package utils_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"mailculator-processor/internal/utils"
)

func TestMoveFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "test-movefile")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up the temporary directory

	// Define file paths
	originPath := filepath.Join(tempDir, "out", "filename.ext")
	destinationDir := filepath.Join(tempDir, "gone")
	destinationPath := filepath.Join(destinationDir, "filename.ext")

	// Create a temporary file at the origin path
	content := []byte("This is a test file.")
	if err := ioutil.WriteFile(originPath, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Call the function to test
	err = utils.MoveFile(originPath, destinationPath)
	if err != nil {
		t.Fatalf("MoveFile failed: %v", err)
	}

	// Check if the file was moved
	if _, err := os.Stat(originPath); !os.IsNotExist(err) {
		t.Errorf("File still exists at origin path: %v", originPath)
	}

	if _, err := os.Stat(destinationPath); err != nil {
		t.Errorf("File does not exist at destination path: %v", destinationPath)
	}

	// Verify the content of the moved file
	movedContent, err := ioutil.ReadFile(destinationPath)
	if err != nil {
		t.Fatalf("Failed to read moved file: %v", err)
	}

	if string(movedContent) != string(content) {
		t.Errorf("Moved file content mismatch. Got: %v, Want: %v", string(movedContent), string(content))
	}
}
