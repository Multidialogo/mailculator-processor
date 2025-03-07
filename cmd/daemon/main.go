package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"mailculator-processor/internal/config"
	"mailculator-processor/internal/daemon"
	"mailculator-processor/internal/email"
	"os/signal"
	"syscall"
)

var runner = runnerFactory()

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer cancel()
	runner(ctx)
}

func runnerFactory() func(context.Context) {
	// TODO handle errors
	cfg := config.NewConfig()
	dynamodbClient := dynamodb.NewFromConfig(cfg.Aws.DynamoDb)
	emailService := email.NewService(dynamodbClient)
	smtpClientFactory := email.NewClientFactory(cfg.Smtp)
	callbackExecutor := email.NewCallbackExecutor()

	d := daemon.NewDaemon(emailService, callbackExecutor, smtpClientFactory)
	return d.RunUntilContextDone
}
