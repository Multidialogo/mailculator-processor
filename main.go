package main

import (
	"os"
	"path/filepath"
	"sync"
	"log"
	"time"

	"mailculator-processor/internal/config"
	"mailculator-processor/internal/service"
	"mailculator-processor/internal/utils"
)

var mutex sync.Mutex

func main() {
	// Retrieve paths from the configuration
	registry := config.GetRegistry()
	basePath := registry.Get("APP_DATA_PATH")
	outboxPath := filepath.Join(basePath, registry.Get("OUTBOX_PATH"))
	sentPath := filepath.Join(basePath, registry.Get("SENT_PATH"))
	failurePath := filepath.Join(basePath, registry.Get("FAILURE_PATH"))

}
