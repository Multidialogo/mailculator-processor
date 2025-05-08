package main

import (
	"context"
	_ "embed"
	"log"
	"os/signal"
	"syscall"

	"mailculator-processor/internal/app"
	"mailculator-processor/internal/config"
)

//go:embed config/config.yaml
var configYamlContent []byte

var runFn = run

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer cancel()
	runFn(ctx)
}

func run(ctx context.Context) {
	cfg, err := config.NewFromYamlContent(configYamlContent)
	if err != nil {
		log.Panic(err)
	}

	runner, err := app.New(cfg)
	if err != nil {
		log.Panic(err)
	}

	runner.Run(ctx)
}
