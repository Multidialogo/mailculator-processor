package main

import (
	"os"
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
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

var basePath string
var outboxBasePath string
var sleepTime time.Duration
var lastModTime time.Duration
var considerEmptyAfterTime time.Duration
var rawEmailClient service.RawEmailClient

func init() {
	// Retrieve paths from the configuration
	registry := config.GetRegistry()
	basePath = registry.Get("APP_DATA_PATH")
	outboxBasePath = filepath.Join(basePath, registry.Get("OUTBOX_PATH"))

	if _, err := os.Stat(outboxBasePath); os.IsNotExist(err) {
		err = os.MkdirAll(outboxBasePath, os.ModePerm)
		if err != nil {
			panic(fmt.Sprintf("failed to create directory: %v", err))
		}
	}

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
	var cycles int = 0

	log.Printf(
		"\033[36mDEBUG: config -> outbox: %s, sleep time: %d s, old file: %d s, old directory: %d s\033[0m",
		outboxBasePath, int(sleepTime.Seconds()), int(lastModTime.Seconds()), int(considerEmptyAfterTime.Seconds()),
	)

	// Main loop to process files periodically
	for {
		// Record memory and CPU usage at the start
		printStats()
		startTime := time.Now()
		currentTime := time.Now()

		cycles++

		if cycles == 10 {
			considerEmptyAfterTimeThreshold := currentTime.Add(considerEmptyAfterTime)
			// Cleaning up orphans directories from outbox
			log.Printf("\033[34mINFO: Cleanup outbox %s, older than %ds\033[0m", outboxBasePath, int(considerEmptyAfterTime.Seconds()))
			err := utils.RemoveEmptyDirs(outboxBasePath, considerEmptyAfterTimeThreshold)
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
				}(file) // Pass the file to the goroutine
			}

			wg.Wait() // Wait for all goroutines to finish
		}

		log.Printf("\u001B[36mDEBUG: Elapsed time: %.2f seconds\n\033[0m", time.Since(startTime).Seconds())

		// Record memory and CPU usage at the end
		printStats()
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

// printStats logs the memory and CPU stats
func printStats() {
	// Memory stats using gopsutil/mem
	v, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("\033[31mCRITICAL: Error fetching memory usage: %v\033[0m", err)
	}

	// CPU stats
	cpus, err := cpu.Percent(0, false)
	if err != nil {
		log.Printf("\033[31mCRITICAL: Error fetching CPU usage: %v\033[0m", err)
	}

	log.Printf(
		"\u001B[33mDEBUG: MEMORY: Total = %v MiB, Used = %v MiB, Free = %v MiB, Percent = %v%%\033[0m",
		v.Total/1024/1024, v.Used/1024/1024, v.Free/1024/1024, v.UsedPercent,
	)
	log.Printf("\u001B[33mDEBUG: CPU: Peak Usage = %v%%\033[0m", cpus[0])
}
