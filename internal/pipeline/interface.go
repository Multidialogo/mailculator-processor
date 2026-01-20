package pipeline

import (
	"context"
	"time"

	"mailculator-processor/internal/outbox"
)

type outboxService interface {
	Query(ctx context.Context, status string, limit int) ([]outbox.Email, error)
	QueryStale(ctx context.Context, status string, olderThan time.Duration, limit int) ([]outbox.Email, error)
	Update(ctx context.Context, id string, status string, errorReason string) error
	UpdateFrom(ctx context.Context, id string, fromStatus string, toStatus string, errorReason string) error
	Ready(ctx context.Context, id string) error
}
