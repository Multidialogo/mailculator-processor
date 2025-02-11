package fs_utils

import (
	"os"
	"path/filepath"
	"testing"
	"time"
	"io/ioutil"

	"mailculator-processor/internal/service/file_locker"
)

func TestMoveFile(t *testing.T) {
	fileLockerFactory := &file_locker.Factory{}
	fsUtils := NewFsUtils(fileLockerFactory)

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "test-movefile")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up the temporary directory

	// Create necessary subdirectories
	outDir := filepath.Join(tempDir, "out")
	err = os.MkdirAll(outDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create 'out' directory: %v", err)
	}

	// Define file paths
	originPath := filepath.Join(outDir, "filename.ext")
	destinationDir := filepath.Join(tempDir, "gone")
	destinationPath := filepath.Join(destinationDir, "filename.ext")

	// Create a temporary file at the origin path
	content := []byte("This is a test file.")
	err = ioutil.WriteFile(originPath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Call the function to test
	err = fsUtils.MoveFile(originPath, destinationPath)
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

func TestListFiles(t *testing.T) {
	fileLockerFactory := &file_locker.Factory{}
	fsUtils := NewFsUtils(fileLockerFactory)

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "test-listfiles")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up the temporary directory

	// Create .EML files with different modification times
	content := []byte("Test content.")
	file1Path := filepath.Join(tempDir, "file1.EML")
	file2Path := filepath.Join(tempDir, "file2.EML")
	file3Path := filepath.Join(tempDir, "file3.EML")

	err = ioutil.WriteFile(file1Path, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create file1.EML: %v", err)
	}
	err = ioutil.WriteFile(file2Path, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create file2.EML: %v", err)
	}
	err = ioutil.WriteFile(file3Path, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create file3.EML: %v", err)
	}

	// Change the modification times
	err = os.Chtimes(file1Path, time.Now().Add(-2*time.Hour), time.Now().Add(-2*time.Hour))
	if err != nil {
		t.Fatalf("Failed to set file1 modification time: %v", err)
	}

	err = os.Chtimes(file2Path, time.Now().Add(-1*time.Hour), time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("Failed to set file2 modification time: %v", err)
	}

	err = os.Chtimes(file3Path, time.Now().Add(-3*time.Hour), time.Now().Add(-3*time.Hour))
	if err != nil {
		t.Fatalf("Failed to set file3 modification time: %v", err)
	}

	// Define the threshold time
	lastModThreshold := time.Now().Add(-2 * time.Hour)

	// Call the ListFiles function
	files, err := fsUtils.ListFiles(tempDir, lastModThreshold)
	if err != nil {
		t.Fatalf("Error listing files: %v", err)
	}

	// Check that the correct files are returned
	expectedFiles := []string{file1Path, file3Path}
	for _, file := range expectedFiles {
		found := false
		for _, f := range files {
			if f == file {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected file %s to be in the result", file)
		}
	}
}

func TestRemoveEmptyDirs(t *testing.T) {
	fileLockerFactory := &file_locker.Factory{}
	fsUtils := NewFsUtils(fileLockerFactory)

	threshold := time.Now().Add(-1 * time.Second)

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "test-removeemptydirs")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up the temporary directory

	// Create directories with .EML files and without
	dir1 := filepath.Join(tempDir, "dir1")
	dir2 := filepath.Join(tempDir, "dir2")
	err = os.MkdirAll(dir1, 0755)
	if err != nil {
		t.Fatalf("Failed to create dir1: %v", err)
	}
	err = os.MkdirAll(dir2, 0755)
	if err != nil {
		t.Fatalf("Failed to create dir2: %v", err)
	}

	// Create .EML file in dir1
	emlFile := filepath.Join(dir1, "file1.EML")
	err = ioutil.WriteFile(emlFile, []byte("Test content."), 0644)
	if err != nil {
		t.Fatalf("Failed to create .EML file: %v", err)
	}

	// Call RemoveEmptyDirs
	err = fsUtils.RemoveEmptyDirs(tempDir, threshold)
	if err != nil {
		t.Fatalf("Error removing empty dirs: %v", err)
	}
}
