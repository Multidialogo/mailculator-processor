package main

import (
	"log"
	"path/filepath"
	"sync"
	"time"
	"strings"
	"strconv"
	"fmt"

	"mailculator-processor/internal/config"
	"mailculator-processor/internal/service"
	"mailculator-processor/internal/utils"
)

var outboxBasePath string
var sentBasePath string
var failureBasePath string
var sleepTime time.Duration
var lastModTime time.Duration
var considerEmptyAfterTime time.Duration
var rawEmailClient service.RawEmailClient

func init() {
	// Retrieve paths from the configuration
	registry := config.GetRegistry()
	basePath := registry.Get("APP_DATA_PATH")
	outboxBasePath = filepath.Join(basePath, registry.Get("OUTBOX_PATH"))
	sentBasePath = filepath.Join(basePath, registry.Get("SENT_PATH"))
	failureBasePath = filepath.Join(basePath, registry.Get("FAILURE_PATH"))

	// Convert the string values to integers
	checkInterval, err := strconv.Atoi(registry.Get("CHECK_INTERVAL"))
	if err != nil {
		panic(fmt.Sprintf("Error converting CHECK_INTERVAL: %v", err))
	}

	lastModInterval, err := strconv.Atoi(registry.Get("LAST_MOD_INTERVAL"))
	if err != nil {
		panic(fmt.Sprintf("Error converting LAST_MOD_INTERVAL: %v", err))
	}

	considerEmptyAfterInterval, err := strconv.Atoi(registry.Get("EMPTY_DIR_INTERVAL"))
	if err != nil {
		panic(fmt.Sprintf("Error converting EMPTY_DIR_INTERVAL: %v", err))
	}

	// Convert to time.Duration
	sleepTime = time.Duration(checkInterval) * time.Second
	lastModTime = -time.Duration(lastModInterval) * time.Second
	considerEmptyAfterTime = -time.Duration(considerEmptyAfterInterval) * time.Second

	rawEmailClient = getEmailClient(registry.Get("ENV"))
}

func main() {
	log.Printf(
		"\033[36mDEBUG: config -> outbox: %s, sent: %s, failure: %s, sleep time: %d s, old file: %d s, old directory: %d s\033[0m",
		outboxBasePath, sentBasePath, failureBasePath, int(sleepTime.Seconds()), int(lastModTime.Seconds()), int(considerEmptyAfterTime.Seconds()),
	)

	// Main loop to process files periodically
	for {
		// Get the current time and define the lastModTimeThreshold (15 seconds ago)
		currentTime := time.Now()
		lastModTimeThreshold := currentTime.Add(lastModTime)
		considerEmptyAfterTimeThreshold := currentTime.Add(considerEmptyAfterTime)
		log.Printf("\033[34mINFO: Listing files in: %s, older than %ds\033[0m", outboxBasePath, int(lastModTime.Seconds()))

		// Get list of files to process
		files, err := utils.ListFiles(outboxBasePath, lastModTimeThreshold)
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

				log.Printf("\033[34mINFO: Processing: %s\033[0m", outboxRelativePath)

				// Call the service.SendEMLFile function for each outboxFilePath
				err, result := service.SendRawEmail(outboxFilePath, rawEmailClient)
				var destPath string
				if err != nil {
					log.Printf("\033[31mCRITICAL: Error processing outboxFilePath %s: %v\033[0m", outboxFilePath, err)
					destPath = filepath.Join(failureBasePath, outboxRelativePath)
				} else {
					log.Printf("\033[34mINFO: Successfully processed outboxFilePath: %s, result: %v\033[0m", outboxFilePath, result)
					destPath = filepath.Join(sentBasePath, outboxRelativePath)
				}

				log.Printf("\033[34mINFO: Moving file from %s to %s\033[0m", outboxFilePath, destPath)
				err = utils.MoveFile(outboxFilePath, destPath)
				if err != nil {
					log.Printf("\033[31mCRITICAL: Failed to move file from %s to %s: %v\033[0m", outboxFilePath, destPath, err)
				}
			}(file) // Pass the file to the goroutine
		}

		wg.Wait() // Wait for all goroutines to finish

		// Cleaning up orphans directories from outbox
		log.Printf("\033[34mINFO: Cleanup outbox %s, older than %ds\033[0m", outboxBasePath, int(considerEmptyAfterTime.Seconds()))
		err = utils.RemoveEmptyDirs(outboxBasePath, considerEmptyAfterTimeThreshold)
		if err != nil {
			log.Printf("\033[31mCRITICAL: Failed to cleanup outbox: %v\033[0m", err)
		}

		// Sleep for a defined time before processing again
		log.Printf("\033[34mINFO: Sleeping for %v before recalling the process\033[0m", sleepTime)
		time.Sleep(sleepTime)
	}
}

// getEmailClient returns the appropriate RawEmailClient based on the environment
func getEmailClient(env string) service.RawEmailClient {
	if env == "TEST" || env == "DEV" {
		// Return a fake client for testing or development
		return &service.FakeEmailClient{}
	}

	// Otherwise, return the real SES client for production
	sesClient, err := service.NewSESClient()
	if err != nil {
		log.Fatalf("Failed to create SES client: %v", err)
	}
	return sesClient
}
