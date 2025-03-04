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

func newDaemonRunMockFuncWithTimeout() (func(context.Context, *config.Container), context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	return func(_ context.Context, container *config.Container) {
		daemon.Run(ctx, container)
	}, cancel
}

func TestMainWillGracefullyShutdownWhenSigtermSignal(t *testing.T) {
	// a timeout is needed to ensure function will stop even if sigterm does not work
	daemonRunMockFunc, cancel := newDaemonRunMockFuncWithTimeout()
	defer cancel()

	daemonRunFunc = daemonRunMockFunc
	go sendSigtermSignalInOneSecond(t)

	assert.NotPanics(t, main)
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
