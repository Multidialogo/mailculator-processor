package utils

import (
	"os"
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
	// Walk through the directory recursively
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		// Skip if there's an error accessing the path
		if err != nil {
			return err
		}

		// Skip the root directory from removal
		if path == dir {
			return nil
		}

		// Process directories only
		if info.IsDir() {
			// First, check if the directory has been modified since the threshold
			if info.ModTime().Before(threshold) {
				// Check if the directory contains any .EML files
				containsEML, err := directoryContainsEML(path)
				if err != nil {
					return err
				}

				// If the directory doesn't contain .EML files, it's empty, remove it
				if !containsEML {
					// First, attempt to remove the contents of the directory
					err := removeDirectoryContents(path)
					if err != nil {
						return err
					}

					// Check if the directory still exists before trying to remove it
					_, err = os.Stat(path)
					if err == nil {
						// The directory exists, attempt to remove it
						err = os.Remove(path)
						if err != nil {
							return fmt.Errorf("failed to remove directory %s: %v", path, err)
						}
					} else if os.IsNotExist(err) {
						// Directory no longer exists, log and skip
					} else {
						return err // other errors (permissions, etc.)
					}
				}
			}
		}
		return nil
	})
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

// Helper function to remove contents of a directory
func removeDirectoryContents(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory contents: %v", err)
	}

	for _, entry := range entries {
		entryPath := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			// Recursively remove contents of subdirectories
			err := removeDirectoryContents(entryPath)
			if err != nil {
				return err
			}
			// Remove the subdirectory after cleaning its contents
			err = os.Remove(entryPath)
			if err != nil {
				return fmt.Errorf("failed to remove subdirectory %s: %v", entryPath, err)
			}
		} else {
			// Remove individual files
			err := os.Remove(entryPath)
			if err != nil {
				return fmt.Errorf("failed to remove file %s: %v", entryPath, err)
			}
		}
	}
	return nil
}
