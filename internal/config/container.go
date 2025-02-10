package config

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	"mailculator-processor/internal/service/email_client"
	"mailculator-processor/internal/service/file_locker"
	"mailculator-processor/internal/service/file_processor"
	"mailculator-processor/internal/service/metrics"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/redis/go-redis/v9"
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
	Metrics       *metrics.Metrics
	FileProcessor *file_processor.FileProcessor
	parameters    parameters
}

func NewContainer() (*Container, error) {
	var registry = GetRegistry()
	var envName = registry.Get("ENV")
	var basePath = registry.Get("BASE_PATH")
	var metricsService *metrics.Metrics
	var fileLockerFactory *file_locker.Factory
	var emailClient email_client.EmailClient
	var err error

	if envName == "TEST" || envName == "DEV" {
		// Return a fake client for testing or development
		emailClient = &email_client.FakeEmailClient{}
		fileLockerFactory = file_locker.NewFactory("FS", nil)
		metricsService = metrics.NewMetrics(false, 0)
	} else {
		var redisHost = registry.Get("REDIS_HOST")
		var redisPort = registry.Get("REDIS_PORT")
		// Load AWS configuration
		var awsCfg, err = config.LoadDefaultConfig(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("unable to load AWS config, %v", err)
		}

		var redisClient = redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
		})

		// Otherwise, return the real SES client for production
		emailClient, err = email_client.NewSESClient(awsCfg)
		if err != nil {
			return nil, fmt.Errorf("Failed to create SES client: %v", err)
		}
		// Convert the string values to integers
		prometheusPort, err := strconv.Atoi(registry.Get("PROMETHEUS_PORT"))
		if err != nil {
			return nil, fmt.Errorf("Error converting PROMETHEUS_PORT: %v", err)
		}
		fileLockerFactory = file_locker.NewFactory("REDIS", redisClient)
		metricsService = metrics.NewMetrics(true, prometheusPort)
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
		Metrics:       metricsService,
		FileProcessor: file_processor.NewFileProcessor(emailClient, fileLockerFactory),
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
