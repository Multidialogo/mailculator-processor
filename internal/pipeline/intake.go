package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"mailculator-processor/internal/email"
	"mailculator-processor/internal/outbox"
)

type IntakePipeline struct {
	outbox outboxService
	logger *slog.Logger
}

func NewIntakePipeline(outbox outboxService) *IntakePipeline {
	return &IntakePipeline{
		outbox: outbox,
		logger: slog.With("pipe", "intake"),
	}
}

func (p *IntakePipeline) Process(ctx context.Context) {
	acceptedList, err := p.outbox.Query(ctx, outbox.StatusAccepted, 25)
	if err != nil {
		p.logger.Error(fmt.Sprintf("error while querying emails to process: %v", err))
		return
	}

	var wg sync.WaitGroup

	for _, e := range acceptedList {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.logger.Info(fmt.Sprintf("processing outbox %v", e.Id))
			subLogger := p.logger.With("outbox", e.Id)

			if err = p.outbox.Update(ctx, e.Id, outbox.StatusIntaking, "", e.TTL); err != nil {
				subLogger.Warn(fmt.Sprintf("failed to acquire processing lock, error: %v", err))
				return
			}

			if err := p.validatePayload(e); err != nil {
				subLogger.Error(fmt.Sprintf("failed to validate payload, error: %v", err))
				p.handle(context.Background(), subLogger, e.Id, outbox.StatusInvalid, err.Error(), e.TTL)
				return
			}

			if err := p.outbox.Ready(context.Background(), e.Id, "", e.TTL); err != nil {
				subLogger.Error(fmt.Sprintf("failed to update status to READY: %v", err))
				p.handle(context.Background(), subLogger, e.Id, outbox.StatusInvalid, err.Error(), e.TTL)
			} else {
				subLogger.Info("successfully intaken")
			}
		}()
	}

	wg.Wait()
}

func (p *IntakePipeline) validatePayload(e outbox.Email) error {
	_, err := email.LoadPayload(e.PayloadFilePath)
	if err != nil {
		return err
	}
	return nil
}

func (p *IntakePipeline) handle(ctx context.Context, logger *slog.Logger, emailId string, status string, errorReason string, ttl *int64) {
	if err := p.outbox.Update(ctx, emailId, status, errorReason, ttl); err != nil {
		msg := fmt.Sprintf("error updating status to %v, error: %v", status, err)
		logger.Error(msg)
	}
}
