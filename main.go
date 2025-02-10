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
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
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
		printStats()
	}
}

// printStats logs the memory and CPU stats and updates the Prometheus metrics
func printStats() {
	// Memory stats using gopsutil/mem
	v, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("\033[31mCRITICAL: Error fetching memory usage: %v\033[0m", err)
	}

	// Update the Prometheus memory usage gauge
	container.Metrics.MemoryUsageGauge.WithLabelValues("total").Set(float64(v.Total)) // Total memory in bytes
	container.Metrics.MemoryUsageGauge.WithLabelValues("used").Set(float64(v.Used))   // Used memory in bytes
	container.Metrics.MemoryUsageGauge.WithLabelValues("free").Set(float64(v.Free))   // Free memory in bytes
	container.Metrics.MemoryUsageGauge.WithLabelValues("percent").Set(v.UsedPercent)  // Percent used

	// CPU stats
	cpus, err := cpu.Percent(0, false)
	if err != nil {
		log.Printf("\033[31mCRITICAL: Error fetching CPU usage: %v\033[0m", err)
	}

	// Update the Prometheus CPU usage gauge
	if len(cpus) > 0 {
		container.Metrics.CpuUsageGauge.WithLabelValues("cpu0").Set(cpus[0]) // Assuming a single CPU for simplicity, can be extended for multiple CPUs
	}

	log.Printf(
		"\u001B[33mDEBUG: MEMORY: Total = %v MiB, Used = %v MiB, Free = %v MiB, Percent = %v%%\033[0m",
		v.Total/1024/1024, v.Used/1024/1024, v.Free/1024/1024, v.UsedPercent,
	)
	log.Printf("\u001B[33mDEBUG: CPU: Peak Usage = %v%%\033[0m", cpus[0])
}
