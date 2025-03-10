package pipeline

import "time"

type Config struct {
	MaxRetries    int
	RetryInterval time.Duration
}
