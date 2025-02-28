package config

import (
	"os"
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"mailculator-processor/internal/service/email_client"
	"mailculator-processor/internal/service/file_locker"
	"mailculator-processor/internal/service/file_processor"
	"mailculator-processor/internal/service/fs_utils"
	"mailculator-processor/internal/service/metrics"

	"github.com/redis/go-redis/v9"
	"github.com/aws/aws-sdk-go-v2/config"
)

type Config struct {
	envName                string
	BasePath               string
	OutboxBasePath         string
	SleepTime              time.Duration
	LastModTime            time.Duration
	ConsiderEmptyAfterTime time.Duration
}

type Container struct {
	Metrics       *metrics.Metrics
	FileProcessor *file_processor.FileProcessor
	FsUtils       *fs_utils.FsUtils
	Config        Config
}

func NewContainer() (*Container, error) {
	var envName = os.Getenv("ENV")
	var basePath = os.Getenv("BASE_PATH")
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
		var redisHost = os.Getenv("REDIS_HOST")
		var redisPort = os.Getenv("REDIS_PORT")
		var redisClient = redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
		})

		// Load AWS configuration
		var awsCfg, err = config.LoadDefaultConfig(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("unable to load AWS config, %v", err)
		}
		// Otherwise, return the real SES client for production
		emailClient, err = email_client.NewSESClient(awsCfg)
		if err != nil {
			return nil, fmt.Errorf("Failed to create SES client: %v", err)
		}
		// Convert the string values to integers
		prometheusPort, err := strconv.Atoi(os.Getenv("PROMETHEUS_PORT"))
		if err != nil {
			return nil, fmt.Errorf("Error converting PROMETHEUS_PORT: %v", err)
		}
		fileLockerFactory = file_locker.NewFactory("REDIS", redisClient)
		metricsService = metrics.NewMetrics(true, prometheusPort)
	}

	// Convert the string values to integers
	checkInterval, err := strconv.Atoi(os.Getenv("CHECK_INTERVAL"))
	if err != nil {
		return nil, fmt.Errorf("Error converting CHECK_INTERVAL: %v", err)
	}

	lastModInterval, err := strconv.Atoi(os.Getenv("LAST_MOD_INTERVAL"))
	if err != nil {
		return nil, fmt.Errorf("Error converting LAST_MOD_INTERVAL: %v", err)
	}

	considerEmptyAfterInterval, err := strconv.Atoi(os.Getenv("EMPTY_DIR_INTERVAL"))
	if err != nil {
		return nil, fmt.Errorf("Error converting EMPTY_DIR_INTERVAL: %v", err)
	}

	return &Container{
		Metrics:       metricsService,
		FileProcessor: file_processor.NewFileProcessor(emailClient, fileLockerFactory),
		FsUtils:       fs_utils.NewFsUtils(fileLockerFactory),
		Config: Config{
			envName:                envName,
			BasePath:               basePath,
			OutboxBasePath:         filepath.Join(basePath, os.Getenv("OUTBOX_PATH")),
			SleepTime:              time.Duration(checkInterval) * time.Second,
			LastModTime:            -time.Duration(lastModInterval) * time.Second,
			ConsiderEmptyAfterTime: -time.Duration(considerEmptyAfterInterval) * time.Second,
		},
	}, nil
}
