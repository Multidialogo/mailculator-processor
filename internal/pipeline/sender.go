package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"mailculator-processor/internal/email"
	"mailculator-processor/internal/outbox"
)

type clientService interface {
	Send(payload email.Payload, attachmentsBasePath string) error
}

type MainSenderPipeline struct {
	outbox              outboxService
	client              clientService
	attachmentsBasePath string
	logger              *slog.Logger
}

func NewMainSenderPipeline(outbox outboxService, client clientService, attachmentsBasePath string) *MainSenderPipeline {
	return &MainSenderPipeline{
		outbox:              outbox,
		client:              client,
		attachmentsBasePath: attachmentsBasePath,
		logger:              slog.With("pipe", "main"),
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
		go func(outboxEmail outbox.Email) {
			defer wg.Done()
			p.logger.Info(fmt.Sprintf("processing outbox %v", outboxEmail.Id))
			logger := p.logger.With("outbox", outboxEmail.Id)

			if err = p.outbox.Update(ctx, outboxEmail.Id, outbox.StatusProcessing, "", outboxEmail.TTL); err != nil {
				logger.Warn(fmt.Sprintf("failed to acquire processing lock, error: %v", err))
				return
			}

			payload, payloadErr := email.LoadPayload(outboxEmail.PayloadFilePath)
			if payloadErr != nil {
				logger.Error(fmt.Sprintf("failed to load payload, error: %v", payloadErr))
				p.handle(context.Background(), logger, outboxEmail.Id, outbox.StatusFailed, payloadErr.Error(), outboxEmail.TTL)
				return
			}

			if err = p.client.Send(payload, p.attachmentsBasePath); err != nil {
				logger.Error(fmt.Sprintf("failed to send, error: %v", err))
				p.handle(context.Background(), logger, outboxEmail.Id, outbox.StatusFailed, err.Error(), outboxEmail.TTL)
			} else {
				logger.Info("successfully sent")
				p.handle(context.Background(), logger, outboxEmail.Id, outbox.StatusSent, "", outboxEmail.TTL)
			}
		}(e)
	}

	wg.Wait()
}

func (p *MainSenderPipeline) handle(ctx context.Context, logger *slog.Logger, emailId string, status string, errorReason string, ttl *int64) {
	if err := p.outbox.Update(ctx, emailId, status, errorReason, ttl); err != nil {
		msg := fmt.Sprintf("error updating status to %v, error: %v", status, err)
		logger.Error(msg)
	}
}
