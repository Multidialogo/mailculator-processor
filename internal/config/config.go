package config

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"os"
)

type Config struct {
	Aws AwsConfig
}

func NewConfig() *Config {
	return &Config{
		Aws: *newAwsConfig(),
	}
}

type AwsConfig struct {
	DynamoDb aws.Config
	Ses      aws.Config
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
		Ses: aws.Config{
			Region:       os.Getenv("AWS_REGION"),
			Credentials:  awsCredentials,
			BaseEndpoint: aws.String(os.Getenv("AWS_SES_ENDPOINT")),
		},
	}
}
