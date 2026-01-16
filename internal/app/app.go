package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"mailculator-processor/internal/eml"
	"mailculator-processor/internal/healthcheck"
	"mailculator-processor/internal/mysql_outbox"
	"mailculator-processor/internal/pipeline"
	"mailculator-processor/internal/smtp"
)

type pipelineProcessor interface {
	Process(ctx context.Context)
}

type App struct {
	pipes             []pipelineProcessor
	interval          int
	healthCheckServer *healthcheck.Server
	mysqlDB           *sql.DB // Keep reference for cleanup
}

type configProvider interface {
	GetHealthCheckServerPort() int
	GetPipelineInterval() int
	GetCallbackConfig() pipeline.CallbackConfig
	GetSmtpConfig() smtp.Config
	GetEmlStoragePath() string
	GetAttachmentsBasePath() string
	GetMySQLDSN() string
	MySQLPipelinesEnabled() bool
}

type mysqlOpener func(driverName, dsn string) (*sql.DB, error)

func New(cp configProvider) (*App, error) {
	return NewWithMySQLOpener(cp, sql.Open)
}

func NewWithMySQLOpener(cp configProvider, opener mysqlOpener) (*App, error) {
	client := smtp.New(cp.GetSmtpConfig())
	emlStorage := eml.NewEMLStorage(cp.GetEmlStoragePath())
	callbackConfig := cp.GetCallbackConfig()
	healthCheckServer := healthcheck.NewServer(cp.GetHealthCheckServerPort())

	var pipes []pipelineProcessor
	var mysqlDB *sql.DB

	// Create MySQL pipelines if enabled
	if cp.MySQLPipelinesEnabled() {
		dsn := cp.GetMySQLDSN()
		if dsn == "" {
			return nil, fmt.Errorf("MySQL pipelines enabled but MySQL DSN is empty")
		}

		slog.Info("MySQL pipelines enabled, initializing...")
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

		mysqlOutbox := mysql_outbox.NewOutbox(mysqlDB)

		pipes = append(pipes,
			pipeline.NewIntakePipeline(mysqlOutbox, emlStorage, cp.GetAttachmentsBasePath()),
			pipeline.NewMainSenderPipeline(mysqlOutbox, client),
			pipeline.NewSentCallbackPipeline(mysqlOutbox, callbackConfig),
			pipeline.NewFailedCallbackPipeline(mysqlOutbox, callbackConfig),
		)
		slog.Info("MySQL pipelines initialized", "count", 4)
	}

	if len(pipes) == 0 {
		return nil, fmt.Errorf("no pipelines enabled, MySQL pipelines must be enabled")
	}

	slog.Info("App initialized", "total_pipelines", len(pipes))

	return &App{
		pipes:             pipes,
		interval:          cp.GetPipelineInterval(),
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

	for _, proc := range a.pipes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.runPipelineUntilContextIsDone(ctx, proc, a.interval)
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
