package main

import (
	"os"
	"log"
	"path/filepath"
	"sync"
	"time"
	"strings"
	"fmt"

	"mailculator-processor/internal/config"
	"mailculator-processor/internal/utils"
)

var container *config.Container

func init() {
	var err error

	container, err = config.NewContainer()
	if err != nil {
		panic(fmt.Sprintf("failed to create container: %v", err))
	}

	if _, err := os.Stat(container.GetString("outboxBasePath")); os.IsNotExist(err) {
		err = os.MkdirAll(container.GetString("outboxBasePath"), os.ModePerm)
		if err != nil {
			panic(fmt.Sprintf("failed to create directory: %v", err))
		}
	}
}

func main() {
	var basePath = container.GetString("basePath")
	var outboxBasePath = container.GetString("outboxBasePath")
	var sleepTime = container.GetDuration("sleepTime")
	var lastModTime = container.GetDuration("lastModTime")
	var considerEmptyAfterTime = container.GetDuration("considerEmptyAfterTime")
	var cycles int = 0

	// Main loop to process files periodically
	for {
		// Record memory and CPU usage at the start
		err := container.Metrics.CollectMemoryAndCpu()
		if err != nil {
			log.Printf("\033[31mCRITICAL: %v\033[0m", err)
		}

		startTime := time.Now()
		currentTime := time.Now()

		cycles++

		if cycles == 10 {
			considerEmptyAfterTimeThreshold := currentTime.Add(considerEmptyAfterTime)
			// Cleaning up orphans directories from outbox
			log.Printf("\033[34mINFO: Cleanup outbox %s, older than %ds\033[0m", outboxBasePath, int(considerEmptyAfterTime.Seconds()))
			err = utils.RemoveEmptyDirs(outboxBasePath, considerEmptyAfterTimeThreshold)
			if err != nil {
				log.Printf("\033[31mCRITICAL: Failed to cleanup outbox: %v\033[0m", err)
			}

			// Sleep for a defined time before processing again
			log.Printf("\033[34mINFO: Sleeping for %v before recalling the process\033[0m", sleepTime)
			time.Sleep(sleepTime)

			cycles = 0
		} else {
			// Get the current time and define the lastModTimeThreshold (15 seconds ago)
			lastModTimeThreshold := currentTime.Add(lastModTime)
			log.Printf("\033[34mINFO: Listing files in: %s, older than %ds\033[0m", outboxBasePath, int(lastModTime.Seconds()))

			// Get list of files to process
			files, err := utils.ListFiles(outboxBasePath, lastModTimeThreshold)
			if err != nil {
				log.Fatalf("Error listing files: %v", err)
			}
			log.Printf("\033[34mINFO: Found: %d message files to process\033[0m", len(files))

			// Update the in-progress files gauge
			container.Metrics.InProgressFilesGauge.WithLabelValues("outbox").Set(float64(len(files)))

			// Process each file by calling SendEMLFile in parallel
			var wg sync.WaitGroup

			for _, file := range files {
				wg.Add(1) // Increment the WaitGroup counter for each file

				go func(outboxFilePath string) {
					defer wg.Done() // Decrement the counter when the goroutine completes

					outboxRelativePath := strings.Replace(outboxFilePath, outboxBasePath, "", 1)

					log.Printf("\033[34mINFO: Processing: %s\033[0m", outboxRelativePath)

					err, result := container.FileProcessor.SendRawEmail(outboxFilePath)
					var destPath string = filepath.Join(basePath, strings.Replace(outboxRelativePath, "/outbox", "", -1))
					if err != nil {
						log.Printf("\033[31mCRITICAL: Error processing outboxFilePath %s: %v\033[0m", outboxFilePath, err)
						destPath = strings.Replace(destPath, "/queues", "/failure/queues", -1)
					} else {
						log.Printf("\033[34mINFO: Successfully processed outboxFilePath: %s, result: %v\033[0m", outboxFilePath, result)
						destPath = strings.Replace(destPath, "/queues", "/sent/queues", -1)
					}

					log.Printf("\033[34mINFO: Moving file from %s to %s\033[0m", outboxFilePath, destPath)
					err = utils.MoveFile(outboxFilePath, destPath)
					if err != nil {
						log.Printf("\033[31mCRITICAL: Failed to move file from %s to %s: %v\033[0m", outboxFilePath, destPath, err)
					}

					// Update the processed files counter with status 'success'
					container.Metrics.ProcessedFilesCounter.WithLabelValues("success").Inc()
				}(file) // Pass the file to the goroutine
			}

			wg.Wait() // Wait for all goroutines to finish
		}

		log.Printf("\u001B[36mDEBUG: Elapsed time: %.2f seconds\n\033[0m", time.Since(startTime).Seconds())

		// Record memory and CPU usage at the end
		err = container.Metrics.CollectMemoryAndCpu()
		if err != nil {
			log.Printf("\033[31mCRITICAL: %v\033[0m", err)
		}
	}
}
