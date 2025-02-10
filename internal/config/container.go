package config

import (
	"fmt"
	"reflect"
	"time"
	"path/filepath"
	"strconv"

	"mailculator-processor/internal/service/email_client"
	"mailculator-processor/internal/service/file_processor"
)

type parameters struct {
	envName                string
	basePath               string
	outboxBasePath         string
	sleepTime              time.Duration
	lastModTime            time.Duration
	considerEmptyAfterTime time.Duration
}

type Container struct {
	FileProcessor *file_processor.FileProcessor
	parameters    parameters
}

func NewContainer() (*Container, error) {
	var registry = GetRegistry()
	var envName = registry.Get("ENV")
	var basePath = registry.Get("BASE_PATH")
	var emailClient email_client.EmailClient
	var err error

	if envName == "TEST" || envName == "DEV" {
		// Return a fake client for testing or development
		emailClient = &email_client.FakeEmailClient{}
	} else {
		// Otherwise, return the real SES client for production
		emailClient, err = email_client.NewSESClient()
		if err != nil {
			return nil, fmt.Errorf("Failed to create SES client: %v", err)
		}
	}

	// Convert the string values to integers
	checkInterval, err := strconv.Atoi(registry.Get("CHECK_INTERVAL"))
	if err != nil {
		return nil, fmt.Errorf("Error converting CHECK_INTERVAL: %v", err)
	}

	lastModInterval, err := strconv.Atoi(registry.Get("LAST_MOD_INTERVAL"))
	if err != nil {
		return nil, fmt.Errorf("Error converting LAST_MOD_INTERVAL: %v", err)
	}

	considerEmptyAfterInterval, err := strconv.Atoi(registry.Get("EMPTY_DIR_INTERVAL"))
	if err != nil {
		return nil, fmt.Errorf("Error converting EMPTY_DIR_INTERVAL: %v", err)
	}

	return &Container{
		FileProcessor: file_processor.NewFileProcessor(emailClient),
		parameters: parameters{
			envName:                envName,
			basePath:               basePath,
			outboxBasePath:         filepath.Join(basePath, registry.Get("OUTBOX_PATH")),
			sleepTime:              time.Duration(checkInterval) * time.Second,
			lastModTime:            -time.Duration(lastModInterval) * time.Second,
			considerEmptyAfterTime: -time.Duration(considerEmptyAfterInterval) * time.Second,
		},
	}, nil
}

func (c *Container) getParam(name string) (interface{}, error) {
	v := reflect.ValueOf(c.parameters)
	field := v.FieldByName(name)

	if !field.IsValid() {
		return nil, fmt.Errorf("Parameter not found: %s", name)
	}

	return field.Interface(), nil
}

func (c *Container) GetString(name string) string {
	value, err := c.getParam(name)
	if err != nil {
		panic(fmt.Sprintf("%s", name))
	}
	str, ok := value.(string)
	if !ok {
		panic(fmt.Sprintf("Parameter %s is not a string", name)) // Type safety check
	}
	return str
}

func (c *Container) GetDuration(name string) time.Duration {
	value, err := c.getParam(name)
	if err != nil {
		panic(fmt.Sprintf("%s", name))
	}
	dur, ok := value.(time.Duration)
	if !ok {
		panic(fmt.Sprintf("Parameter %s is not a duration", name)) // Type safety check
	}
	return dur
}
