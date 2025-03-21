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

type callbackExecutor struct {
	cmd           *exec.Cmd
	maxRetries    int
	retryInterval time.Duration
}

func (e *callbackExecutor) Execute() error {
	var err error
	for i := 0; i < e.maxRetries; i++ {
		if err = e.cmd.Run(); err == nil {
			return nil
		}
		time.Sleep(e.retryInterval)
	}
	return err
}

type callbackExecutorFactory struct {
	maxRetries    int
	retryInterval time.Duration
}

func newCallbackExecutorFactory(maxRetries int, retryInterval time.Duration) *callbackExecutorFactory {
	return &callbackExecutorFactory{
		maxRetries:    maxRetries,
		retryInterval: retryInterval,
	}
}

func (f *callbackExecutorFactory) New(cmd string) *callbackExecutor {
	return &callbackExecutor{
		cmd:           exec.Command("sh", "-c", cmd),
		maxRetries:    f.maxRetries,
		retryInterval: f.retryInterval,
	}
}

type callbackOutboxService interface {
	Query(ctx context.Context, status string, limit int) ([]outbox.Email, error)
	Update(ctx context.Context, id string, status string) error
}

type callbackPipeline struct {
	outbox                  callbackOutboxService
	callbackExecutorFactory *callbackExecutorFactory
	logger                  *slog.Logger
	startStatus             string
	processingStatus        string
	acknowledgedStatus      string
}

func (p *callbackPipeline) process(ctx context.Context, logger *slog.Logger, id string, callback string) {
	if err := p.outbox.Update(ctx, id, p.processingStatus); err != nil {
		logger.Warn(fmt.Sprintf("failed to acquire processing lock, error: %v", err))
		return
	}

	cmd := p.callbackExecutorFactory.New(callback)
	if err := cmd.Execute(); err != nil {
		logger.Error(fmt.Sprintf("error while executing callback, error: %v", err))
		if rollErr := p.outbox.Update(ctx, id, p.startStatus); rollErr != nil {
			logger.Error(fmt.Sprintf("error while rolling back status after callback error, error: %v", err))
		}
		return
	}

	if err := p.outbox.Update(ctx, id, p.acknowledgedStatus); err != nil {
		logger.Error(fmt.Sprintf("error while updating status after callback, error: %v", err))
	}
}

type SentCallbackPipeline struct {
	callbackPipeline
}

func NewSentCallbackPipeline(cfg CallbackConfig, ob callbackOutboxService) *SentCallbackPipeline {
	return &SentCallbackPipeline{
		callbackPipeline{
			outbox:                  ob,
			callbackExecutorFactory: newCallbackExecutorFactory(cfg.MaxRetries, cfg.RetryInterval),
			logger:                  slog.With("pipe", "sent-callback"),
			startStatus:             outbox.StatusSent,
			processingStatus:        outbox.StatusCallingSentCallback,
			acknowledgedStatus:      outbox.StatusSentAcknowledged,
		},
	}
}

func (p *SentCallbackPipeline) Process(ctx context.Context) {
	sentList, err := p.outbox.Query(ctx, outbox.StatusSent, 25)
	if err != nil {
		p.logger.Error(fmt.Sprintf("error while querying emails to process: %v", err))
		return
	}

	var wg sync.WaitGroup

	for _, e := range sentList {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.logger.Info(fmt.Sprintf("processing email %v", e.Id))
			subLogger := p.logger.With("email", e.Id)
			p.process(ctx, subLogger, e.Id, e.SuccessCallback)
		}()
	}

	wg.Wait()
}

type FailedCallbackPipeline struct {
	callbackPipeline
}

func NewFailedCallbackPipeline(cfg CallbackConfig, ob callbackOutboxService) *FailedCallbackPipeline {
	return &FailedCallbackPipeline{
		callbackPipeline{
			outbox:                  ob,
			logger:                  slog.With("pipe", "failed-callback"),
			callbackExecutorFactory: newCallbackExecutorFactory(cfg.MaxRetries, cfg.RetryInterval),
			startStatus:             outbox.StatusFailed,
			processingStatus:        outbox.StatusCallingFailedCallback,
			acknowledgedStatus:      outbox.StatusFailedAcknowledged,
		},
	}
}

func (p *FailedCallbackPipeline) Process(ctx context.Context) {
	failedList, err := p.outbox.Query(ctx, outbox.StatusFailed, 25)
	if err != nil {
		p.logger.Error(fmt.Sprintf("error while querying emails to process: %v", err))
		return
	}

	var wg sync.WaitGroup

	for _, e := range failedList {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.logger.Info(fmt.Sprintf("processing email %v", e.Id))
			subLogger := p.logger.With("email", e.Id)
			p.process(ctx, subLogger, e.Id, e.FailureCallback)
		}()
	}

	wg.Wait()
}
