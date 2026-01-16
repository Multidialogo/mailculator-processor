package pipeline

import (
	"context"
	"mailculator-processor/internal/outbox"
)

type outboxService interface {
	Query(ctx context.Context, status string, limit int) ([]outbox.Email, error)
	Update(ctx context.Context, id string, status string, errorReason string) error
	Ready(ctx context.Context, id string, emlFilePath string) error
	Requeue(ctx context.Context, id string) error
}
