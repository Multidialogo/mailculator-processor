package main

import (
	"context"
	"github.com/stretchr/testify/require"
	"os"
	"syscall"
	"testing"
	"time"
)

type runnerMock struct {
	duration time.Duration
}

func (d *runnerMock) runWithTimeout(_ context.Context) {
	time.Sleep(d.duration)
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
	runnerMock := &runnerMock{duration: 200 * time.Millisecond}
	runFn = runnerMock.runWithTimeout

	var sendSignalError error
	go sleepAndSendSigtermSignal(100*time.Millisecond, sendSignalError)

	require.NotPanics(t, main)
	require.Nilf(t, sendSignalError, "failed to send signal: %v", sendSignalError)
}
