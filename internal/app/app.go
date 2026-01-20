package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"mailculator-processor/internal/healthcheck"
	"mailculator-processor/internal/outbox"
	"mailculator-processor/internal/pipeline"
	"mailculator-processor/internal/smtp"
)

type pipelineProcessor interface {
	Process(ctx context.Context)
}

type pipelineEntry struct {
	proc     pipelineProcessor
	interval int
}

type App struct {
	pipes             []pipelineEntry
	healthCheckServer *healthcheck.Server
	mysqlDB           *sql.DB // Keep reference for cleanup
}

type configProvider interface {
	GetHealthCheckServerPort() int
	GetPipelineInterval() int
	GetRestorePipelineInterval() int
	GetRestorePipelineMaxAge() time.Duration
	GetCallbackConfig() pipeline.CallbackConfig
	GetSmtpConfig() smtp.Config
	GetAttachmentsBasePath() string
	GetMySQLDSN() string
}

type mysqlOpener func(driverName, dsn string) (*sql.DB, error)

func New(cp configProvider) (*App, error) {
	return NewWithMySQLOpener(cp, sql.Open)
}

func NewWithMySQLOpener(cp configProvider, opener mysqlOpener) (*App, error) {
	client := smtp.New(cp.GetSmtpConfig())
	callbackConfig := cp.GetCallbackConfig()
	healthCheckServer := healthcheck.NewServer(cp.GetHealthCheckServerPort())

	var pipes []pipelineEntry
	var mysqlDB *sql.DB

	dsn := cp.GetMySQLDSN()
	if dsn == "" {
		return nil, fmt.Errorf("MySQL DSN is empty")
	}

	slog.Info("MySQL pipelines initializing...")
	var err error
	mysqlDB, err = opener("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	// Configure connection pool
	mysqlDB.SetMaxOpenConns(25)
	mysqlDB.SetMaxIdleConns(5)
	mysqlDB.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := mysqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	mysqlOutbox := outbox.NewOutbox(mysqlDB)

	mainInterval := cp.GetPipelineInterval()
	restoreInterval := cp.GetRestorePipelineInterval()
	restoreMaxAge := cp.GetRestorePipelineMaxAge()

	pipes = append(pipes,
		pipelineEntry{proc: pipeline.NewIntakePipeline(mysqlOutbox), interval: mainInterval},
		pipelineEntry{proc: pipeline.NewMainSenderPipeline(mysqlOutbox, client, cp.GetAttachmentsBasePath()), interval: mainInterval},
		pipelineEntry{proc: pipeline.NewSentCallbackPipeline(mysqlOutbox, callbackConfig), interval: mainInterval},
		pipelineEntry{proc: pipeline.NewFailedCallbackPipeline(mysqlOutbox, callbackConfig), interval: mainInterval},
		pipelineEntry{proc: pipeline.NewRestoreIntakingPipeline(mysqlOutbox, restoreMaxAge), interval: restoreInterval},
		pipelineEntry{proc: pipeline.NewRestoreProcessingPipeline(mysqlOutbox, restoreMaxAge), interval: restoreInterval},
		pipelineEntry{proc: pipeline.NewRestoreCallingSentPipeline(mysqlOutbox, restoreMaxAge), interval: restoreInterval},
		pipelineEntry{proc: pipeline.NewRestoreCallingFailedPipeline(mysqlOutbox, restoreMaxAge), interval: restoreInterval},
	)
	slog.Info("MySQL pipelines initialized", "count", len(pipes))

	slog.Info("App initialized", "total_pipelines", len(pipes))

	return &App{
		pipes:             pipes,
		healthCheckServer: healthCheckServer,
		mysqlDB:           mysqlDB,
	}, nil
}

func (a *App) runPipelineUntilContextIsDone(ctx context.Context, proc pipelineProcessor, interval int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			proc.Process(ctx)
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func (a *App) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for _, entry := range a.pipes {
		proc := entry.proc
		interval := entry.interval
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.runPipelineUntilContextIsDone(ctx, proc, interval)
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info(fmt.Sprintf("%v", a.healthCheckServer.ListenAndServe(ctx)))
	}()

	wg.Wait()

	// Cleanup MySQL connection if it was opened
	if a.mysqlDB != nil {
		if err := a.mysqlDB.Close(); err != nil {
			slog.Error("failed to close MySQL connection", "error", err)
		}
	}
}
