package main

import (
	"context"
	"log"
	"mailculator-processor/internal/app"
	"mailculator-processor/internal/config"
	"os/signal"
	"syscall"
)

const configFilePath = "../../config/app.yaml"

var runFn = run

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer cancel()
	runFn(ctx)
}

func run(ctx context.Context) {
	cfg := app.Config{}
	err := config.NewLoader(configFilePath).Load(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	runner, err := app.New(cfg)
	if err != nil {
		log.Fatal(err)
		return
	}

	runner.Run(ctx)
}
