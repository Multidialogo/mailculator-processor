package main

import (
	"os"
	"log"
	"path/filepath"
	"sync"
	"sort"
	"time"
	"strings"

	"mailculator-processor/internal/config"
	"mailculator-processor/internal/service"
	"mailculator-processor/internal/utils"
)

func main() {
	// Retrieve paths from the configuration
	registry := config.GetRegistry()
	basePath := registry.Get("APP_DATA_PATH")
	outboxBasePath := filepath.Join(basePath, registry.Get("OUTBOX_PATH"))
	sentBasePath := filepath.Join(basePath, registry.Get("SENT_PATH"))
	failureBasePath := filepath.Join(basePath, registry.Get("FAILURE_PATH"))
	sleepTime := 6 * time.Second

	log.Printf("DEBUG: config -> outbox: %s, sent: %s, failure: %s, sleep time: %ds", outboxBasePath, sentBasePath, failureBasePath, (sleepTime / time.Second))

	// Main loop to process files periodically
	for {
		// Get the current time and define the lastModTimeThreshold (15 seconds ago)
		currentTime := time.Now()
		lastModTimeThreshold := currentTime.Add(-15 * time.Second)
		log.Printf("INFO: Listing files in: %s\n", outboxBasePath)

		// Get list of files to process
		files, err := listFiles(outboxBasePath, lastModTimeThreshold)
		if err != nil {
			log.Fatalf("Error listing files: %v", err)
		}

		// Process each file by calling SendEMLFile in parallel
		var wg sync.WaitGroup

		for _, file := range files {
			wg.Add(1) // Increment the WaitGroup counter for each file

			go func(outboxFilePath string) {
				defer wg.Done() // Decrement the counter when the goroutine completes

				outboxRelativePath := strings.Replace(outboxFilePath, outboxBasePath, "", 1)

				log.Printf("INFO: Processing: %s\n", outboxRelativePath)

				// Call the service.SendEMLFile function for each outboxFilePath
				err := service.SendEMLFile(outboxFilePath)
				var destPath string
				if err != nil {
					log.Printf("CRITICAL: Error processing outboxFilePath %s: %v\n", outboxFilePath, err)
					destPath = filepath.Join(failureBasePath, outboxRelativePath)
				} else {
					log.Printf("INFO: Successfully processed outboxFilePath: %s\n", outboxFilePath)
					destPath = filepath.Join(sentBasePath, outboxRelativePath)
				}

				log.Printf("INFO: Moving file from %s to %s\n", outboxFilePath, destPath)
				err = utils.MoveFile(outboxFilePath, destPath)
				if err != nil {
					log.Printf("CRITICAL: Failed to move file from %s to %s: %v\n", outboxFilePath, destPath, err)
				}
			}(file) // Pass the file to the goroutine
		}

		wg.Wait() // Wait for all goroutines to finish

		// Sleep for a defined time before processing again
		log.Printf("INFO: Sleeping for %v before recalling the process.\n", sleepTime)
		time.Sleep(sleepTime)
	}
}

// listFiles returns a sorted slice of file paths that were last modified more than the threshold.
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

		// Check if the last modified time is more than the threshold
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
		// Use the file's modification time directly
		fileInfoI, errI := os.Stat(files[i])
		fileInfoJ, errJ := os.Stat(files[j])

		if errI != nil || errJ != nil {
			// Log the error and continue sorting
			log.Printf("Error getting file info for sorting: %v %v", errI, errJ)
			return false
		}
		return fileInfoI.ModTime().Before(fileInfoJ.ModTime())
	})

	return files, nil
}
