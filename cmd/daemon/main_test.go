package main

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"syscall"
	"testing"
	"time"
)

type runnerMockWithTimeout struct {
	timeout      time.Duration
	duration     time.Duration
	contextError error
}

func (d *runnerMockWithTimeout) runWithTimeout(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	time.Sleep(d.duration)
	d.contextError = ctx.Err()
}

func sleepAndSendSigtermSignal(sleep time.Duration, err error) {
	time.Sleep(sleep)
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		return
	}
	err = p.Signal(syscall.SIGTERM)
}

func Test_Main_WhenSigtermSignal_WillGracefullyShutdown(t *testing.T) {
	runnerMock := &runnerMockWithTimeout{timeout: time.Second, duration: 200 * time.Millisecond}
	runner = runnerMock.runWithTimeout

	var sendSignalError error
	go sleepAndSendSigtermSignal(100*time.Millisecond, sendSignalError)

	require.NotPanics(t, main)
	require.Nilf(t, sendSignalError, "failed to send signal: %v", sendSignalError)
	assert.NotErrorIs(t, runnerMock.contextError, context.DeadlineExceeded)
}
