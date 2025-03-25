package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"mailculator-processor/internal/app"
	"mailculator-processor/internal/config"
)

const configFilePath = "../../config/app.yaml"

var runFn = run

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer cancel()
	runFn(ctx)
}

func run(ctx context.Context) {
	cfg, err := config.NewFromYaml(configFilePath)
	if err != nil {
		log.Panic(err)
	}

	runner, err := app.New(cfg)
	if err != nil {
		log.Panic(err)
	}

	runner.Run(ctx)
}
