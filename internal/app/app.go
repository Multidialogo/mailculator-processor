package app

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"log/slog"
	"mailculator-processor/internal/outbox"
	"mailculator-processor/internal/pipeline"
	"mailculator-processor/internal/smtp"
	"os"
	"sync"
)

type pipelineProcessor interface {
	Process(ctx context.Context)
}

type App struct {
	pipes []pipelineProcessor
}

func New(cfg Config) (*App, error) {
	loggerWriter, err := os.OpenFile(cfg.Log.FilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	loggerHandler := slog.NewTextHandler(loggerWriter, nil)
	slog.SetDefault(slog.New(loggerHandler))

	db := dynamodb.NewFromConfig(cfg.getAwsConfig())
	client := smtp.New(cfg.getSmtpConfig())
	outboxService := outbox.NewOutbox(db)

	mainSenderPipe := pipeline.NewMainSenderPipeline(outboxService, client)
	sentCallbackPipe := pipeline.NewSentCallbackPipeline(cfg.getPipelineConfig(), outboxService)
	failedCallbackPipe := pipeline.NewFailedCallbackPipeline(cfg.getPipelineConfig(), outboxService)

	pipes := []pipelineProcessor{mainSenderPipe, sentCallbackPipe, failedCallbackPipe}
	return &App{pipes: pipes}, nil
}

func (a *App) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for _, proc := range a.pipes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.runPipelineUntilContextIsDone(ctx, proc)
		}()
	}

	wg.Wait()
}

func (a *App) runPipelineUntilContextIsDone(ctx context.Context, proc pipelineProcessor) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			proc.Process(ctx)
		}
	}
}
