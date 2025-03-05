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

var conf = config.NewConfig()

var runner = runnerFactory()

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer cancel()
	runner(ctx)
}

func runnerFactory() func(context.Context) {
	dynamodbClient := dynamodb.NewFromConfig(conf.Aws.DynamoDb)
	emailLocker := email.NewLocker(dynamodbClient, conf.Outbox.LockTableName)
	emailFinder := email.NewFinder(emailLocker, dynamodbClient, conf.Outbox.OutboxTableName)
	emailSender := email.NewSESSender(conf.Aws.DynamoDb)

	d := daemon.NewDaemon(emailFinder, emailSender)
	return d.RunUntilContextDone
}
