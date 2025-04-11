package app

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"mailculator-processor/internal/outbox"
	"mailculator-processor/internal/pipeline"
	"mailculator-processor/internal/smtp"
)

type pipelineProcessor interface {
	Process(ctx context.Context)
}

type App struct {
	pipes    []pipelineProcessor
	interval int
}

type configProvider interface {
	GetAwsConfig() aws.Config
	GetPipelineInterval() int
	GetCallbackConfig() pipeline.CallbackConfig
	GetSmtpConfig() smtp.Config
}

func New(cp configProvider) (*App, error) {
	db := dynamodb.NewFromConfig(cp.GetAwsConfig())
	client := smtp.New(cp.GetSmtpConfig())
	outboxService := outbox.NewOutbox(db)

	mainSenderPipe := pipeline.NewMainSenderPipeline(outboxService, client)
	callbackConfig := cp.GetCallbackConfig()
	sentCallbackPipe := pipeline.NewSentCallbackPipeline(outboxService, callbackConfig)
	failedCallbackPipe := pipeline.NewFailedCallbackPipeline(outboxService, callbackConfig)

	pipes := []pipelineProcessor{mainSenderPipe, sentCallbackPipe, failedCallbackPipe}
	return &App{pipes: pipes, interval: cp.GetPipelineInterval()}, nil
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

	wg.Wait()
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
