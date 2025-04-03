//go:build unit

package main

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_Main_WhenSigtermSignal_WillGracefullyShutdown(t *testing.T) {
	runFn = func(ctx context.Context) {
		time.Sleep(200 * time.Millisecond)
	}

	var sendSignalError error
	go func() {
		time.Sleep(100 * time.Millisecond)
		p, sendSignalError := os.FindProcess(os.Getpid())
		if sendSignalError != nil {
			return
		}
		sendSignalError = p.Signal(syscall.SIGTERM)
	}()

	require.NotPanics(t, main)
	require.Nilf(t, sendSignalError, "failed to send signal: %v", sendSignalError)
}
