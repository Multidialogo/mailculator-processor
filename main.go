package main

import (
	"fmt"
	"os"
	"log"
	"path/filepath"
	"sync"
	"sort"
	"time"

	"mailculator-processor/internal/config"
	"mailculator-processor/internal/service"
	"mailculator-processor/internal/utils"
)

func main() {
	// Retrieve paths from the configuration
	registry := config.GetRegistry()
	basePath := registry.Get("APP_DATA_PATH")
	outboxPath := filepath.Join(basePath, registry.Get("OUTBOX_PATH"))
	sentPath := filepath.Join(basePath, registry.Get("SENT_PATH"))
	failurePath := filepath.Join(basePath, registry.Get("FAILURE_PATH"))

	// Get the current time and define the threshold (1 minute ago)
	currentTime := time.Now()
	threshold := currentTime.Add(-1 * time.Minute)
	sleepTime := time.Duration(6)

	// Get list of files to process
	files, err := listFiles(outboxPath, threshold)
	if err != nil {
		panic(fmt.Sprintf("Error: %v", err))
	}

	// Process each file by calling SendEMLFile in parallel
	var wg sync.WaitGroup

	for _, file := range files {
		wg.Add(1) // Increment the WaitGroup counter for each file

		go func(originPath string) {
			defer wg.Done() // Decrement the counter when the goroutine completes

			log.Printf("INFO: Processing originPath: %s\n", originPath)

			// Call the service.SendEMLFile function for each originPath
			err := service.SendEMLFile(originPath)
			var destPath string
			if err != nil {
				log.Printf("CRITICAL: Error processing originPath %s: %v\n", originPath, err)
				destPath = failurePath
			} else {
				log.Printf("INFO: Successfully processed originPath: %s\n", originPath)
				destPath = sentPath
			}

			err = utils.MoveFile(originPath, destPath)
			if err != nil {
				log.Printf("CRITICAL: Failed to move originPath %s to %s: %v\n", originPath, destPath, err)
			} else {
				log.Printf("INFO: File moved to processed originPath: %s\n", destPath)
			}
		}(file) // Pass the file to the goroutine
	}

	wg.Wait() // Wait for all goroutines to finish

	// Sleep for 6 seconds before recalling the main function
	log.Printf("INFO: Sleeping for %d seconds before recalling the main function.\n", sleepTime)
	time.Sleep(sleepTime * time.Second)

	main()
}

// listFiles returns a sorted slice of file paths that were last modified more than 1 minute ago.
func listFiles(dir string, threshold time.Time) ([]string, error) {
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

		// Check if the last modified time is more than 1 minute ago
		if info.ModTime().Before(threshold) {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort files by modification time (oldest first)
	sort.Slice(files, func(i, j int) bool {
		fileInfoI, _ := os.Stat(files[i])
		fileInfoJ, _ := os.Stat(files[j])
		return fileInfoI.ModTime().Before(fileInfoJ.ModTime())
	})

	return files, nil
}
