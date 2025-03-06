package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"mailculator-processor/internal/awsutils"
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
	cfg := config.NewConfig()

	dynamodbClient := dynamodb.NewFromConfig(cfg.Aws.DynamoDb)
	sesClient := ses.NewFromConfig(cfg.Aws.Ses)

	emailDataMapper := email.NewDataMapper(dynamodbClient)
	lockDataMapper := email.NewLockDataMapper(dynamodbClient)
	emailClient := awsutils.NewSesEmailClient(sesClient)

	emailService := email.NewService(emailDataMapper, lockDataMapper, emailClient)

	d := daemon.NewDaemon(emailService)
	return d.RunUntilContextDone
}
