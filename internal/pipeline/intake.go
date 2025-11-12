package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"

	"mailculator-processor/internal/eml"
	"mailculator-processor/internal/outbox"
)

type EmailPayload struct {
	Id            string            `json:"id" validate:"required,uuid"`
	From          string            `json:"from" validate:"required,email"`
	ReplyTo       string            `json:"reply_to" validate:"required,email"`
	To            string            `json:"to" validate:"required,email"`
	Subject       string            `json:"subject" validate:"required"`
	BodyHTML      string            `json:"body_html" validate:"required_without=BodyText"`
	BodyText      string            `json:"body_text" validate:"required_without=BodyHTML"`
	Attachments   []string          `json:"attachments" validate:"dive,uri"`
	CustomHeaders map[string]string `json:"custom_headers"`
}

type emlStorageService interface {
	Store(emlData eml.EML) (string, error)
}

type IntakePipeline struct {
	outbox              outboxService
	emlStorage          emlStorageService
	attachmentsBasePath string
	logger              *slog.Logger
	validator           *validator.Validate
}

func NewIntakePipeline(outbox outboxService, emlStorage emlStorageService, attachmentsBasePath string) *IntakePipeline {
	return &IntakePipeline{
		outbox:              outbox,
		emlStorage:          emlStorage,
		attachmentsBasePath: attachmentsBasePath,
		logger:              slog.With("pipe", "intake"),
		validator:           validator.New(),
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

			emlPath, err := p.createAndStoreEml(e)
			if err != nil {
				subLogger.Error(fmt.Sprintf("failed to create and store EML, error: %v", err))
				p.handle(context.Background(), subLogger, e.Id, outbox.StatusInvalid, err.Error(), e.TTL)
			} else {
				if err := p.outbox.Ready(ctx, e.Id, emlPath, e.TTL); err != nil {
					subLogger.Error(fmt.Sprintf("failed to update status to READY: %v", err))
					p.handle(context.Background(), subLogger, e.Id, outbox.StatusInvalid, err.Error(), e.TTL)
				} else {
					subLogger.Info("successfully intaken")
				}
			}
		}()
	}

	wg.Wait()
}

func (p *IntakePipeline) createAndStoreEml(e outbox.Email) (string, error) {
	payloadData, err := os.ReadFile(e.PayloadFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read payload file %s: %w", e.PayloadFilePath, err)
	}

	var payload EmailPayload
	if err := json.Unmarshal(payloadData, &payload); err != nil {
		return "", fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	if err := p.validator.Struct(payload); err != nil {
		return "", fmt.Errorf("payload validation failed: %w", err)
	}

	// Prepend base path to attachments
	attachmentsWithBasePath := make([]string, len(payload.Attachments))
	for i, attachment := range payload.Attachments {
		attachmentsWithBasePath[i] = p.attachmentsBasePath + attachment
	}

	emlData := eml.EML{
		MessageId:     payload.Id,
		From:          payload.From,
		ReplyTo:       payload.ReplyTo,
		To:            payload.To,
		Subject:       payload.Subject,
		BodyHTML:      payload.BodyHTML,
		BodyText:      payload.BodyText,
		Date:          time.Now(),
		Attachments:   attachmentsWithBasePath,
		CustomHeaders: payload.CustomHeaders,
	}

	emlFilePath, err := p.emlStorage.Store(emlData)
	if err != nil {
		return "", fmt.Errorf("failed to store EML: %w", err)
	}

	return emlFilePath, nil
}

func (p *IntakePipeline) handle(ctx context.Context, logger *slog.Logger, emailId string, status string, errorReason string, ttl int64) {
	if err := p.outbox.Update(ctx, emailId, status, errorReason, ttl); err != nil {
		msg := fmt.Sprintf("error updating status to %v, error: %v", status, err)
		logger.Error(msg)
	}
}
