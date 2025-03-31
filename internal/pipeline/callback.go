package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"mailculator-processor/internal/outbox"
	"os/exec"
	"sync"
	"time"
)

type CallbackConfig struct {
	MaxRetries    int
	RetryInterval time.Duration
}

type callbackExecutorInterface interface {
	Execute(cmd *exec.Cmd) error
}

type callbackExecutor struct{}

func (e *callbackExecutor) Execute(cmd *exec.Cmd) error {
	return cmd.Run()
}

type callbackPipeline struct {
	outbox             outboxService
	cfg                CallbackConfig
	callbackExecutor   callbackExecutorInterface
	logger             *slog.Logger
	startStatus        string
	processingStatus   string
	acknowledgedStatus string
}

func (p *callbackPipeline) process(ctx context.Context, getCallback func(e outbox.Email) string) {
	callbackList, err := p.outbox.Query(ctx, p.startStatus, 25)
	if err != nil {
		p.logger.Error(fmt.Sprintf("error while querying emails to process: %v", err))
		return
	}

	var wg sync.WaitGroup

	for _, e := range callbackList {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.logger.Info(fmt.Sprintf("processing email %v", e.Id))
			subLogger := p.logger.With("email", e.Id)

			if err := p.outbox.Update(ctx, e.Id, p.processingStatus); err != nil {
				subLogger.Warn(fmt.Sprintf("failed to acquire processing lock, error: %v", err))
				return
			}

			cmd := exec.Command("sh", "-c", getCallback(e))
			for i := 0; i < p.cfg.MaxRetries; i++ {
				if err = p.callbackExecutor.Execute(cmd); err == nil {
					break
				}
				time.Sleep(p.cfg.RetryInterval)
			}

			if err != nil {
				subLogger.Error(fmt.Sprintf("error while executing callback, error: %v", err))
				if rollErr := p.outbox.Update(ctx, e.Id, p.startStatus); rollErr != nil {
					subLogger.Error(fmt.Sprintf("error while rolling back status after callback error, error: %v", err))
				}
				return
			}

			if err := p.outbox.Update(ctx, e.Id, p.acknowledgedStatus); err != nil {
				subLogger.Error(fmt.Sprintf("error while updating status after callback, error: %v", err))
			}
		}()
	}

	wg.Wait()
}

type SentCallbackPipeline struct {
	callbackPipeline
}

func NewSentCallbackPipeline(cfg CallbackConfig, ob outboxService) *SentCallbackPipeline {
	return &SentCallbackPipeline{
		callbackPipeline{
			outbox:             ob,
			cfg:                cfg,
			callbackExecutor:   &callbackExecutor{},
			logger:             slog.With("pipe", "sent-callback"),
			startStatus:        outbox.StatusSent,
			processingStatus:   outbox.StatusCallingSentCallback,
			acknowledgedStatus: outbox.StatusSentAcknowledged,
		},
	}
}

func (p *SentCallbackPipeline) Process(ctx context.Context) {
	p.process(ctx, func(e outbox.Email) string { return e.SuccessCallback })
}

type FailedCallbackPipeline struct {
	callbackPipeline
}

func NewFailedCallbackPipeline(cfg CallbackConfig, ob outboxService) *FailedCallbackPipeline {
	return &FailedCallbackPipeline{
		callbackPipeline{
			outbox:             ob,
			cfg:                cfg,
			callbackExecutor:   &callbackExecutor{},
			logger:             slog.With("pipe", "failed-callback"),
			startStatus:        outbox.StatusFailed,
			processingStatus:   outbox.StatusCallingFailedCallback,
			acknowledgedStatus: outbox.StatusFailedAcknowledged,
		},
	}
}

func (p *FailedCallbackPipeline) Process(ctx context.Context) {
	p.process(ctx, func(e outbox.Email) string { return e.FailureCallback })
}
