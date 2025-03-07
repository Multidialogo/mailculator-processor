package config

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"os"
	"strconv"
)

type Config struct {
	Aws  AwsConfig
	Smtp SmtpConfig
}

func NewConfig() *Config {
	// TODO handle errors
	return &Config{
		Aws:  *newAwsConfig(),
		Smtp: *newSmtpConfig(),
	}
}

type AwsConfig struct {
	DynamoDb aws.Config
}

func newAwsConfig() *AwsConfig {
	awsCredentials := credentials.NewStaticCredentialsProvider(
		os.Getenv("AWS_ACCESS_KEY_ID"),
		os.Getenv("AWS_SECRET_ACCESS_KEY"),
		"",
	)

	return &AwsConfig{
		DynamoDb: aws.Config{
			Region:       os.Getenv("AWS_REGION"),
			Credentials:  awsCredentials,
			BaseEndpoint: aws.String(os.Getenv("AWS_DYNAMODB_ENDPOINT")),
		},
	}
}

type SmtpConfig struct {
	Username         string
	Password         string
	Host             string
	Port             int
	From             string
	AllowInsecureTls bool
}

func newSmtpConfig() *SmtpConfig {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	allowInsecureTls, _ := strconv.ParseBool(os.Getenv("SMTP_ALLOW_INSECURE_TLS"))

	return &SmtpConfig{
		Username:         os.Getenv("SMTP_USER"),
		Password:         os.Getenv("SMTP_PASS"),
		Host:             os.Getenv("SMTP_HOST"),
		Port:             port,
		From:             os.Getenv("SMTP_FROM"),
		AllowInsecureTls: allowInsecureTls,
	}
}
