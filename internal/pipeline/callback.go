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
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type CallbackPipeline struct {
	outbox             outboxService
	callbackUrl        string
	httpClient         HttpClient
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
				"data": map[string]any{
					"attributes": map[string]any{
						"code":         statusCode,
						"reachedAt":    e.UpdatedAt,
						"messageUuids": []string{e.Id},
						"reason":       e.Reason,
					},
				},
			}

			jsonBody, errJson := json.Marshal(payload)
			if errJson != nil {
				subLogger.Error(fmt.Sprintf("Error during data conversion to JSON: %v", errJson))
				return
			}
			bodyReader := bytes.NewReader(jsonBody)

			req, errReq := http.NewRequest(http.MethodPost, p.callbackUrl, bodyReader)
			if errReq != nil {
				subLogger.Error(fmt.Sprintf("Error during request creation: %v", errReq))
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-MTRAX-SOURCE", "MULTIDIALOGO")
			_, _ = p.httpClient.Do(req)

			if err = p.outbox.Update(ctx, e.Id, p.acknowledgedStatus, e.Reason); err != nil {
				subLogger.Error(fmt.Sprintf("error while updating status after callback, error: %v", err))
			}
		}()
	}

	wg.Wait()
}

func NewSentCallbackPipeline(ob outboxService, callbackUrl string) *CallbackPipeline {
	return &CallbackPipeline{
		outbox:             ob,
		callbackUrl:        callbackUrl,
		httpClient:         &http.Client{},
		logger:             slog.With("pipe", "sent-callback"),
		startStatus:        outbox.StatusSent,
		processingStatus:   outbox.StatusCallingSentCallback,
		acknowledgedStatus: outbox.StatusSentAcknowledged,
	}
}

func NewFailedCallbackPipeline(ob outboxService, callbackUrl string) *CallbackPipeline {
	return &CallbackPipeline{
		outbox:             ob,
		callbackUrl:        callbackUrl,
		httpClient:         &http.Client{},
		logger:             slog.With("pipe", "failed-callback"),
		startStatus:        outbox.StatusFailed,
		processingStatus:   outbox.StatusCallingFailedCallback,
		acknowledgedStatus: outbox.StatusFailedAcknowledged,
	}
}
