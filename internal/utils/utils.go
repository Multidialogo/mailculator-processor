package utils

import (
	"os"
	"io"
	"path/filepath"
	"time"
	"sort"
	"fmt"
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

// ListFiles returns a sorted slice of file paths that were last modified more than the threshold.
func ListFiles(dir string, lastModificationThreshold time.Time) ([]string, error) {
	var files []string

	// Walk through the directory
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and process only .EML files
		if info.IsDir() || filepath.Ext(path) != ".EML" {
			return nil
		}

		// Check if the last modified time is more than the lastModificationThreshold
		if info.ModTime().Before(lastModificationThreshold) {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort files by modification time (oldest first)
	sort.Slice(files, func(i, j int) bool {
		// Use the file's modification time directly
		fileInfoI, errI := os.Stat(files[i])
		fileInfoJ, errJ := os.Stat(files[j])

		if errI != nil || errJ != nil {
			// Continue sorting
			return false
		}
		return fileInfoI.ModTime().Before(fileInfoJ.ModTime())
	})

	return files, nil
}

// RemoveEmptyDirs deletes directories that do not contain .EML files and have not been modified since the threshold.
func RemoveEmptyDirs(dir string, threshold time.Time) error {
	var emptyDirs, err = findEmptyDirs(dir, threshold)
	if err != nil {
		return err
	}

	// Remove the collected empty directories
	var removalErrors []error
	// Remove the collected empty directories
	for _, path := range emptyDirs {
		err := os.Remove(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Already deleted, ignore
			}
			removalErrors = append(removalErrors, fmt.Errorf("failed to remove directory %s: %v", path, err))
		}
	}

	if len(removalErrors) > 0 {
		return fmt.Errorf("Errors during removal operations: %v", removalErrors)
	}

	return nil
}

func findEmptyDirs(dir string, threshold time.Time) ([]string, error) {
	var emptyDirs []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			path := filepath.Join(dir, entry.Name())
			info, err := os.Stat(path)
			if err != nil {
				return nil, err
			}

			if info.ModTime().Before(threshold) {
				containsEML, err := directoryContainsEML(path)
				if err != nil {
					return nil, err
				}

				if !containsEML {
					empty, err := isDirectoryEmpty(path)
					if err != nil {
						return nil, err
					}
					if empty {
						emptyDirs = append(emptyDirs, path)
					}
				}
			}
		}
	}

	return emptyDirs, nil
}

// isDirectoryEmpty checks if a directory is empty
func isDirectoryEmpty(path string) (bool, error) {
	d, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer d.Close()

	_, err = d.Readdirnames(1) // Read at most one file
	if err == io.EOF {
		return true, nil // Directory is empty
	}
	return false, err // Either not empty or another error
}

// Helper function to check if the directory contains any .EML files
func directoryContainsEML(dir string) (bool, error) {
	var containsEML bool
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".EML" {
			containsEML = true
			return filepath.SkipDir // Stop walking if we find an EML file
		}
		return nil
	})
	return containsEML, err
}
