package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"mailculator-processor/internal/outbox"
)

type RestorePipeline struct {
	outbox        outboxService
	logger        *slog.Logger
	startStatus   string
	restoreStatus string
	maxAge        time.Duration
}

func newRestorePipeline(outbox outboxService, name string, startStatus string, restoreStatus string, maxAge time.Duration) *RestorePipeline {
	return &RestorePipeline{
		outbox:        outbox,
		logger:        slog.With("pipe", name),
		startStatus:   startStatus,
		restoreStatus: restoreStatus,
		maxAge:        maxAge,
	}
}

func (p *RestorePipeline) Process(ctx context.Context) {
	staleList, err := p.outbox.QueryStale(ctx, p.startStatus, p.maxAge, 0)
	if err != nil {
		p.logger.Error(fmt.Sprintf("error while querying stale emails to restore: %v", err))
		return
	}

	var wg sync.WaitGroup

	for _, e := range staleList {
		wg.Add(1)
		go func(email outbox.Email) {
			defer wg.Done()
			p.logger.Info(fmt.Sprintf("restoring email %v", email.Id))
			subLogger := p.logger.With("email", email.Id)

			if err = p.outbox.UpdateFrom(ctx, email.Id, p.startStatus, p.restoreStatus, ""); err != nil {
				subLogger.Warn(fmt.Sprintf("failed to restore email status, error: %v", err))
				return
			}

			subLogger.Info("successfully restored email status")
		}(e)
	}

	wg.Wait()
}

func NewRestoreIntakingPipeline(ob outboxService, maxAge time.Duration) *RestorePipeline {
	return newRestorePipeline(ob, "restore-intaking", outbox.StatusIntaking, outbox.StatusAccepted, maxAge)
}

func NewRestoreProcessingPipeline(ob outboxService, maxAge time.Duration) *RestorePipeline {
	return newRestorePipeline(ob, "restore-processing", outbox.StatusProcessing, outbox.StatusReady, maxAge)
}

func NewRestoreCallingSentPipeline(ob outboxService, maxAge time.Duration) *RestorePipeline {
	return newRestorePipeline(ob, "restore-calling-sent", outbox.StatusCallingSentCallback, outbox.StatusSent, maxAge)
}

func NewRestoreCallingFailedPipeline(ob outboxService, maxAge time.Duration) *RestorePipeline {
	return newRestorePipeline(ob, "restore-calling-failed", outbox.StatusCallingFailedCallback, outbox.StatusFailed, maxAge)
}
