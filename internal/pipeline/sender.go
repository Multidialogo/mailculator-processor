package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"mailculator-processor/internal/outbox"
	"sync"
)

type mainOutboxService interface {
	Query(ctx context.Context, status string, limit int) ([]outbox.Email, error)
	Update(ctx context.Context, id string, status string) error
}

type clientService interface {
	Send(emlFilePath string) error
}

type MainSenderPipeline struct {
	outbox mainOutboxService
	client clientService
	logger *slog.Logger
}

func NewMainSenderPipeline(outbox mainOutboxService, client clientService) *MainSenderPipeline {
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
			p.process(ctx, e)
		}()
	}

	wg.Wait()
}

func (p *MainSenderPipeline) process(ctx context.Context, item outbox.Email) {
	logger := p.logger.With("outbox", item.Id)

	if err := p.outbox.Update(ctx, item.Id, outbox.StatusProcessing); err != nil {
		logger.Warn(fmt.Sprintf("failed to acquire processing lock, error: %v", err))
		return
	}

	if err := p.client.Send(item.EmlFilePath); err != nil {
		logger.Error(fmt.Sprintf("failed to send, error: %v", err))
		p.handle(ctx, logger, item, outbox.StatusFailed)
	} else {
		logger.Info("successfully sent")
		p.handle(ctx, logger, item, outbox.StatusSent)
	}
}

func (p *MainSenderPipeline) handle(ctx context.Context, logger *slog.Logger, item outbox.Email, status string) {
	if err := p.outbox.Update(ctx, item.Id, status); err != nil {
		msg := fmt.Sprintf("error updating status to %v, error: %v", status, err)
		logger.Error(msg)
	}
}
