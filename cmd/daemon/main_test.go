package main

import (
	"context"
	"github.com/stretchr/testify/assert"
	"mailculator-processor/internal/config"
	"mailculator-processor/internal/daemon"
	"os"
	"syscall"
	"testing"
	"time"
)

func newDaemonRunMockFuncWithTimeout(wrapperCtx context.Context) func(context.Context, *config.Container) {
	return func(ctx context.Context, container *config.Container) {
		ctx, cancel := context.WithTimeout(wrapperCtx, 5*time.Second)
		defer cancel()
		daemon.Run(ctx, container)
	}
}

func TestMainWillGracefullyShutdownWhenSigtermSignal(t *testing.T) {
	// a timeout is needed to ensure function will stop even if sigterm does not work
	contextWithTimeout, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	daemonRunFunc = newDaemonRunMockFuncWithTimeout(contextWithTimeout)
	go sendSigtermSignalInOneSecond(t)

	assert.NotPanics(t, main)
	assert.Nil(t, contextWithTimeout.Err())
}

func sendSigtermSignalInOneSecond(t *testing.T) {
	time.Sleep(time.Second)

	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal("failed to find process to send signal")
	}

	err = p.Signal(syscall.SIGTERM)
	if err != nil {
		t.Fatal("failed to send SIGTERM")
	}
}
