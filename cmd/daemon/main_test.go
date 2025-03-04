package main

import (
	"context"
	"github.com/stretchr/testify/assert"
	"mailculator-processor/internal/config"
	"mailculator-processor/internal/daemon"
	"testing"
	"time"
)

func newDaemonRunMockFunc(ctx context.Context) func(context.Context, *config.Container) {
	return func(_ context.Context, container *config.Container) {
		daemon.Run(ctx, container)
	}
}

func TestMainFunc(t *testing.T) {
	ctx, cancelTestCtx := context.WithTimeout(context.Background(), time.Second)
	defer cancelTestCtx()

	daemonRunFunc = newDaemonRunMockFunc(ctx)
	assert.NotPanics(t, main)
}
