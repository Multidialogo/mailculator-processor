package main

import (
	"context"
	"fmt"
	"mailculator-processor/internal/config"
	"mailculator-processor/internal/daemon"
)

var daemonRunFunc = daemon.Run

func main() {
	container, err := config.NewContainer()
	if err != nil {
		panic(fmt.Sprintf("failed to create container: %v", err))
	}

	ctx := context.Background()
	daemonRunFunc(ctx, container)
}
