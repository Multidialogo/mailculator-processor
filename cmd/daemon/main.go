package main

import (
	"context"
	"fmt"
	"mailculator-processor/internal/config"
	"mailculator-processor/internal/daemon"
	"os/signal"
	"syscall"
)

var daemonRunFunc = daemon.Run

func main() {
	container, err := config.NewContainer()
	if err != nil {
		panic(fmt.Sprintf("failed to create container: %v", err))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer cancel()

	daemonRunFunc(ctx, container)
}
