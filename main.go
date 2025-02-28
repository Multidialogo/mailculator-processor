package main

import (
	"os"
	"path/filepath"
	"sync"
	"time"
	"strings"
	"fmt"

	"mailculator-processor/internal/config"
	"mailculator-processor/internal/service/logger"
)

var container *config.Container

func init() {
	var err error

	container, err = config.NewContainer()
	if err != nil {
		panic(fmt.Sprintf("failed to create container: %v", err))
	}

	if _, err := os.Stat(container.Config.OutboxBasePath); os.IsNotExist(err) {
		err = os.MkdirAll(container.Config.OutboxBasePath, os.ModePerm)
		if err != nil {
			panic(fmt.Sprintf("failed to create directory: %v", err))
		}
	}
}

func main() {
	var log = logger.NewLogger()
	var basePath = container.Config.BasePath
	var outboxBasePath = container.Config.OutboxBasePath
	var sleepTime = container.Config.SleepTime
	var lastModTime = container.Config.LastModTime
	var considerEmptyAfterTime = container.Config.ConsiderEmptyAfterTime
	var cycles int = 0

	// Main loop to process files periodically
	for {
		// Record memory and CPU usage at the start
		err := container.Metrics.CollectMemoryAndCpu()
		if err != nil {
			log.Print("CRITICAL", fmt.Sprintf("%v", err))
		}

		startTime := time.Now()
		currentTime := time.Now()

		cycles++

		if cycles == 10 {
			considerEmptyAfterTimeThreshold := currentTime.Add(considerEmptyAfterTime)
			// Cleaning up orphans directories from outbox
			log.Print("INFO", fmt.Sprintf("Cleanup outbox %s, older than %ds", outboxBasePath, int(considerEmptyAfterTime.Seconds())))
			err = container.FsUtils.RemoveEmptyDirs(outboxBasePath, considerEmptyAfterTimeThreshold)
			if err != nil {
				log.Print("CRITICAL", fmt.Sprintf("Error trying to remove empty directories: %v", err))
			}

			// Sleep for a defined time before processing again
			log.Print("INFO", fmt.Sprintf("Sleeping for %v", sleepTime))
			time.Sleep(sleepTime)

			cycles = 0
		} else {
			// Get the current time and define the lastModTimeThreshold (15 seconds ago)
			lastModTimeThreshold := currentTime.Add(lastModTime)
			log.Print("INFO", fmt.Sprintf("Listing files in: %s, older than %ds", outboxBasePath, int(lastModTime.Seconds())))

			// Get list of files to process
			files, err := container.FsUtils.ListFiles(outboxBasePath, lastModTimeThreshold)
			if err != nil {
				log.Fatal("EMERGENCY", fmt.Sprintf(" Error listing files: %v", err))
			}

			log.Print("INFO", fmt.Sprintf("Found: %d message files to process", len(files)))

			// Update the in-progress files gauge
			container.Metrics.InProgressFilesGauge.WithLabelValues("outbox").Set(float64(len(files)))

			// Process each file by calling SendEMLFile in parallel
			var wg sync.WaitGroup

			for _, file := range files {
				wg.Add(1) // Increment the WaitGroup counter for each file

				go func(outboxFilePath string) {
					defer wg.Done() // Decrement the counter when the goroutine completes

					outboxRelativePath := strings.Replace(outboxFilePath, outboxBasePath, "", 1)

					log.Print("INFO", fmt.Sprintf("Processing: %s", outboxRelativePath))

					result, err := container.FileProcessor.SendRawEmail(outboxFilePath)
					var destPath string = filepath.Join(basePath, strings.Replace(outboxRelativePath, "/outbox", "", -1))
					if err != nil {
						log.Print("CRITICAL", fmt.Sprintf("Error processing outboxFilePath %s: %v", outboxFilePath, err))
						destPath = strings.Replace(destPath, "/queues", "/failure/queues", -1)
					} else {
						log.Print("INFO", fmt.Sprintf("Successfully processed outboxFilePath: %s, result: %v", outboxFilePath, result))
						destPath = strings.Replace(destPath, "/queues", "/sent/queues", -1)
					}

					log.Print("INFO", fmt.Sprintf("Moving file from %s to %s", outboxFilePath, destPath))
					err = container.FsUtils.MoveFile(outboxFilePath, destPath)
					if err != nil {
						log.Print("CRITICAL", fmt.Sprintf("Failed to move file from %s to %s: %v\033[0m", outboxFilePath, destPath, err))
					}

					// Update the processed files counter with status 'success'
					container.Metrics.ProcessedFilesCounter.WithLabelValues("success").Inc()
				}(file) // Pass the file to the goroutine
			}

			wg.Wait() // Wait for all goroutines to finish
		}

		log.Print("DEBUG", fmt.Sprintf("Elapsed time: %.2f seconds", time.Since(startTime).Seconds()))

		// Record memory and CPU usage at the end
		err = container.Metrics.CollectMemoryAndCpu()
		if err != nil {
			log.Print("CRITICAL", fmt.Sprintf("%v", err))
		}
	}
}
