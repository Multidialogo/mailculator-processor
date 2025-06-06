package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"mailculator-processor/internal/outbox"
	"sync"
)

type clientService interface {
	Send(emlFilePath string) error
}

type MainSenderPipeline struct {
	outbox outboxService
	client clientService
	logger *slog.Logger
}

func NewMainSenderPipeline(outbox outboxService, client clientService) *MainSenderPipeline {
	return &MainSenderPipeline{
		outbox: outbox,
		client: client,
		logger: slog.With("pipe", "main"),
	}
}

func (p *MainSenderPipeline) Process(ctx context.Context) {
	readyList, err := p.outbox.Query(ctx, outbox.StatusReady, 25)
	if err != nil {
		p.logger.Error(fmt.Sprintf("error while querying emails to process: %v", err))
		return
	}

	var wg sync.WaitGroup

	for _, e := range readyList {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.logger.Info(fmt.Sprintf("processing outbox %v", e.Id))
			logger := p.logger.With("outbox", e.Id)

			if err = p.outbox.Update(ctx, e.Id, outbox.StatusProcessing, ""); err != nil {
				logger.Warn(fmt.Sprintf("failed to acquire processing lock, error: %v", err))
				return
			}

			if err = p.client.Send(e.EmlFilePath); err != nil {
				logger.Error(fmt.Sprintf("failed to send, error: %v", err))
				p.handle(ctx, logger, e.Id, outbox.StatusFailed, err.Error())
			} else {
				logger.Info("successfully sent")
				p.handle(ctx, logger, e.Id, outbox.StatusSent, "")
			}
		}()
	}

	wg.Wait()
}

func (p *MainSenderPipeline) handle(ctx context.Context, logger *slog.Logger, emailId string, status string, errorReason string) {
	if err := p.outbox.Update(ctx, emailId, status, errorReason); err != nil {
		msg := fmt.Sprintf("error updating status to %v, error: %v", status, err)
		logger.Error(msg)
	}
}
