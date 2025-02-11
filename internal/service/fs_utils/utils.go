package fs_utils

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"mailculator-processor/internal/service/file_locker"
)

type FsUtils struct {
	fileLockerFactory *file_locker.Factory
}

func NewFsUtils(fileLockerFactory *file_locker.Factory) *FsUtils {
	return &FsUtils{
		fileLockerFactory: fileLockerFactory,
	}
}

// MoveFile ensures the destination directory exists and moves the file
func (fu *FsUtils) MoveFile(originPath string, destinationPath string) error {
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
func (fu *FsUtils) ListFiles(dir string, lastModificationThreshold time.Time) ([]string, error) {
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

		// Check if the file is locked at the filesystem level
		if fu.isFileLocked(path) != false {
			// If the file is locked, skip it

			return nil
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

func (fu *FsUtils) isFileLocked(filePath string) bool {
	// Create a flock instance for the given file
	fileLock := fu.fileLockerFactory.GetInstance(filePath)

	// Try to acquire an exclusive (blocking) lock on the file

	locked, err := fileLock.TryLock()
	if err != nil {
		// If there was an error trying to lock the file, to stay safe we consider it as locked
		fmt.Println("Error trying to lock file:", err)
		return true
	}

	// If locked is true, that means the file was locked successfully
	if locked {
		// Release the lock immediately (because we just wanted to check)
		fileLock.Unlock()
		return false
	}

	// If the lock could not be obtained, the file is locked
	return true
}

// RemoveEmptyDirs deletes directories that do not contain .EML files and have not been modified since the threshold.
func (fu *FsUtils) RemoveEmptyDirs(dir string, threshold time.Time) error {
	emptyDirs, err := findEmptyDirs(dir, threshold)
	if err != nil {
		return err
	}

	var removalErrors []error
	for _, path := range emptyDirs {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			removalErrors = append(removalErrors, fmt.Errorf("failed to remove directory %s: %v", path, err))
		}
	}

	if len(removalErrors) > 0 {
		return fmt.Errorf("Errors during removal: %v", removalErrors)
	}
	return nil
}

func findEmptyDirs(dir string, threshold time.Time) ([]string, error) {
	var emptyDirs []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Saltiamo il root directory
		if path == dir {
			return nil
		}

		if info.IsDir() {
			// Legge il contenuto della directory
			subEntries, err := os.ReadDir(path)
			if err != nil {
				return nil
			}

			// Se la directory è stata modificata di recente, ignoriamola
			if info.ModTime().After(threshold) {
				return nil
			}

			// Verifica se contiene file .EML
			isEmpty := true
			for _, subEntry := range subEntries {
				if !subEntry.IsDir() && filepath.Ext(subEntry.Name()) == ".EML" {
					isEmpty = false
					break
				}
			}

			// Se è vuota e vecchia, aggiungiamola alla lista
			if isEmpty {
				emptyDirs = append(emptyDirs, path)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Ordinare per lunghezza decrescente (le foglie prima dei genitori)
	sort.Slice(emptyDirs, func(i, j int) bool {
		return len(emptyDirs[i]) > len(emptyDirs[j])
	})

	return emptyDirs, nil
}
