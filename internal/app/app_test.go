//go:build unit

package app

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mailculator-processor/internal/pipeline"
	"mailculator-processor/internal/smtp"
)

type configProviderMock struct{}

func newConfigProviderMock() *configProviderMock {
	return &configProviderMock{}
}

func (cp *configProviderMock) GetAwsConfig() aws.Config {
	return aws.Config{
		Region:       "dummy-region",
		Credentials:  credentials.NewStaticCredentialsProvider("dummy-key", "dummy-secret", "dummy-session"),
		BaseEndpoint: aws.String("dummy-endpoint"),
	}
}

func (cp *configProviderMock) GetPipelineInterval() int {
	return 1
}

func (cp *configProviderMock) GetCallbackConfig() pipeline.CallbackConfig {
	return pipeline.CallbackConfig{Url: "dummy-domain.com",
		RetryInterval: 2,
		MaxRetries:    3,
	}
}

func (cp *configProviderMock) GetSmtpConfig() smtp.Config {
	return smtp.Config{
		Host:             "dummy-host",
		Port:             1234,
		User:             "dummy-user",
		Password:         "dummy-password",
		From:             "dummy-from",
		AllowInsecureTls: false,
	}
}

func TestAppInstance(t *testing.T) {
	app, errNew := New(newConfigProviderMock())
	require.NoError(t, errNew)
	require.Equal(t, 3, len(app.pipes))
	assert.NotZero(t, app.pipes[0])
	assert.NotZero(t, app.pipes[1])
	assert.NotZero(t, app.pipes[2])
}

type processorMock struct {
	sleepMilliseconds int
	calls             int
}

func newProcessorMock(sleepMilliseconds int) *processorMock {
	return &processorMock{sleepMilliseconds: sleepMilliseconds, calls: 0}
}

func (t *processorMock) Process(ctx context.Context) {
	time.Sleep(time.Duration(t.sleepMilliseconds) * time.Millisecond)
	t.calls++
}

func TestRunFunction(t *testing.T) {
	// TODO add healthcheck tests

	proc1 := newProcessorMock(200)
	proc2 := newProcessorMock(200)
	app := &App{pipes: []pipelineProcessor{proc1, proc2}}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	app.Run(ctx)

	assert.Equal(t, 1, proc1.calls)
	assert.Equal(t, 1, proc2.calls)
}
