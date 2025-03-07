package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"mailculator-processor/internal/config"
	"mailculator-processor/internal/daemon"
	"mailculator-processor/internal/email"
	"mailculator-processor/internal/shellexec"
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
	shellCommandFactory := shellexec.NewCommandFactory()

	d := daemon.NewDaemon(emailService, smtpClientFactory, shellCommandFactory)
	return d.RunUntilContextDone
}
