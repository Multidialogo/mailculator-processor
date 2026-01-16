//go:build unit

package app

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mailculator-processor/internal/healthcheck"
	"mailculator-processor/internal/pipeline"
	"mailculator-processor/internal/smtp"
)

type configProviderMock struct{}

func newConfigProviderMock() *configProviderMock {
	return &configProviderMock{}
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

func (cp *configProviderMock) GetHealthCheckServerPort() int {
	return 8080
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

func (cp *configProviderMock) GetAttachmentsBasePath() string {
	return "/base/attachments/path/"
}

func (cp *configProviderMock) GetMySQLDSN() string {
	return "sqlmock"
}

func TestAppInstance(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})
	mock.ExpectPing()

	opener := func(_ string, _ string) (*sql.DB, error) {
		return db, nil
	}

	app, errNew := NewWithMySQLOpener(newConfigProviderMock(), opener)
	require.NoError(t, errNew)
	require.Equal(t, 4, len(app.pipes))
	assert.NotZero(t, app.pipes[0])
	assert.NotZero(t, app.pipes[1])
	assert.NotZero(t, app.pipes[2])
	assert.NotZero(t, app.pipes[3])
	assert.NoError(t, mock.ExpectationsWereMet())
}

type processorMock struct {
	sleepMilliseconds int
	calls             int
}

func newProcessorMock(sleepMilliseconds int) *processorMock {
	return &processorMock{sleepMilliseconds: sleepMilliseconds, calls: 0}
}

func (t *processorMock) Process(_ context.Context) {
	time.Sleep(time.Duration(t.sleepMilliseconds) * time.Millisecond)
	t.calls++
}

func TestRunFunction(t *testing.T) {
	// TODO add healthcheck tests

	proc1 := newProcessorMock(200)
	proc2 := newProcessorMock(200)
	healthCheckServer := healthcheck.NewServer(8080)
	app := &App{pipes: []pipelineProcessor{proc1, proc2}, healthCheckServer: healthCheckServer}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	app.Run(ctx)

	assert.Equal(t, 1, proc1.calls)
	assert.Equal(t, 1, proc2.calls)
}
