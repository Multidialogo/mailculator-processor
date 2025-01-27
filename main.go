package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"path/filepath"
	"sync"
	"log"

	"mailculator-processor/internal/config"
	"mailculator-processor/internal/service"
	"mailculator-processor/internal/utils"
)

var mutex sync.Mutex

func main() {
	registry := config.GetRegistry()
	basePath := registry.Get("APP_DATA_PATH")
	draftPath := filepath.Join(basePath, registry.Get("DRAFT_OUTPUT_PATH"))
	sentPath := filepath.Join(basePath, registry.Get("SENT_OUTPUT_PATH"))
	failurePath := filepath.Join(basePath, registry.Get("FAILURE_OUTPUT_PATH"))

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(fmt.Sprintf("EMERGENCY: %v", err))
	}
	defer watcher.Close()

	err = watcher.Add(draftPath)
	if err != nil {
		panic(fmt.Sprintf("EMERGENCY: %v", err))
	}

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Create == fsnotify.Create {
				// Lock the file processing to prevent concurrency issues
				mutex.Lock()

				// Proceed with file processing
				var destinationPath string
				err := service.SendEMLFile(event.Name)
				if err != nil {
					log.Printf("CRITICAL: %v\n", err)
					destinationPath = failurePath
				} else {
					log.Printf("INFO: %s sent\n", event.Name)
					destinationPath = sentPath
				}

				// Move the file and check for errors
				err = utils.MoveFile(event.Name, destinationPath)
				if err != nil {
					// In case of error moving, release the lock and continue
					mutex.Unlock()
					panic(fmt.Sprintf("CRITICAL: %v", err))
				}

				// Unlock the file processing
				mutex.Unlock()
			}
		case err := <-watcher.Errors:
			panic(fmt.Sprintf("CRITICAL: %v", err))
		}
	}
}
