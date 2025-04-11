package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"mailculator-processor/internal/outbox"
	"net/http"
	"sync"
	"time"
)

type CallbackConfig struct {
	MaxRetries    int
	RetryInterval time.Duration
	Url           string
}

type CallbackPipeline struct {
	outbox             outboxService
	cfg                CallbackConfig
	logger             *slog.Logger
	startStatus        string
	processingStatus   string
	acknowledgedStatus string
}

func (p *CallbackPipeline) Process(ctx context.Context) {
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

			if err = p.outbox.Update(ctx, e.Id, p.processingStatus, e.Reason); err != nil {
				subLogger.Warn(fmt.Sprintf("failed to acquire processing lock, error: %v", err))
				return
			}

			statusCode := "TRAVELING"
			if p.startStatus == outbox.StatusFailed {
				statusCode = "DISPATCH-ERROR"
			}
			payload := map[string]any{
				"code":          statusCode,
				"reached_at":    e.UpdatedAt,
				"message_uuids": []string{e.Id},
				"reason":        e.Reason,
			}

			jsonBody, errJson := json.Marshal(payload)
			if errJson != nil {
				subLogger.Error(fmt.Sprintf("Error during data conversion to JSON: %v", errJson))
				return
			}
			bodyReader := bytes.NewReader(jsonBody)

			req, errReq := http.NewRequest(http.MethodPost, p.cfg.Url, bodyReader)
			if errReq != nil {
				subLogger.Error(fmt.Sprintf("Error during request creation: %v", errReq))
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-MTRAX-SOURCE", "MULTIDIALOGO")

			attempt := 0
			resp := &http.Response{StatusCode: http.StatusConflict}
			for attempt < p.cfg.MaxRetries && resp.StatusCode == http.StatusConflict {
				resp, err = http.DefaultClient.Do(req)
				if err != nil {
					subLogger.Error(fmt.Sprintf("Error in the request: %v", err))
					return
				}
				if resp.StatusCode == http.StatusConflict {
					attempt++
					retryMsg := ""
					var retryInterval time.Duration = 0
					if attempt < p.cfg.MaxRetries {
						retryMsg = fmt.Sprintf(" Try to call again %s in %d seconds.", p.cfg.Url, p.cfg.RetryInterval)
						retryInterval = p.cfg.RetryInterval * time.Second
					}
					subLogger.Warn(fmt.Sprintf(
						"Response status code is %d.%s Attempt %d/%d",
						resp.StatusCode, retryMsg, attempt, p.cfg.MaxRetries,
					))
					time.Sleep(retryInterval)
				}
			}
			if attempt == p.cfg.MaxRetries {
				subLogger.Error(fmt.Sprintf("Max retries exceeded for the url %s", p.cfg.Url))
			}

			if err = p.outbox.Update(ctx, e.Id, p.acknowledgedStatus, e.Reason); err != nil {
				subLogger.Error(fmt.Sprintf("error while updating status after callback, error: %v", err))
			}
		}()
	}

	wg.Wait()
}

func NewSentCallbackPipeline(ob outboxService, cfg CallbackConfig) *CallbackPipeline {
	return &CallbackPipeline{
		outbox:             ob,
		cfg:                cfg,
		logger:             slog.With("pipe", "sent-callback"),
		startStatus:        outbox.StatusSent,
		processingStatus:   outbox.StatusCallingSentCallback,
		acknowledgedStatus: outbox.StatusSentAcknowledged,
	}
}

func NewFailedCallbackPipeline(ob outboxService, cfg CallbackConfig) *CallbackPipeline {
	return &CallbackPipeline{
		outbox:             ob,
		cfg:                cfg,
		logger:             slog.With("pipe", "failed-callback"),
		startStatus:        outbox.StatusFailed,
		processingStatus:   outbox.StatusCallingFailedCallback,
		acknowledgedStatus: outbox.StatusFailedAcknowledged,
	}
}
