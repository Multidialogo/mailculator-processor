package app

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"mailculator-processor/internal/pipeline"
	"mailculator-processor/internal/smtp"
	"time"
)

type AwsConfig struct {
	BaseEndpoint string `yaml:"base_endpoint"`
	Key          string `yaml:"key" validate:"required"`
	Secret       string `yaml:"secret" validate:"required"`
	Region       string `yaml:"region" validate:"required"`
}

type CallbacksConfig struct {
	MaxRetries    int `yaml:"max_retries" validate:"required"`
	RetryInterval int `yaml:"retry_interval" validate:"required"`
}

type SmtpConfig struct {
	Host             string `yaml:"host" validate:"required"`
	Port             int    `yaml:"port" validate:"required"`
	User             string `yaml:"user" validate:"required"`
	Password         string `yaml:"password" validate:"required"`
	From             string `yaml:"from" validate:"required"`
	AllowInsecureTls bool   `yaml:"allow_insecure_tls"`
}

type Config struct {
	Aws       AwsConfig       `yaml:"aws" validate:"required"`
	Callbacks CallbacksConfig `yaml:"callbacks" validate:"required"`
	Smtp      SmtpConfig      `yaml:"smtp" validate:"required"`
}

func (c *Config) getAwsCredentialsProvider() credentials.StaticCredentialsProvider {
	return credentials.NewStaticCredentialsProvider(
		c.Aws.Key,
		c.Aws.Secret,
		"",
	)
}

func (c *Config) getAwsConfig() aws.Config {
	cfg := aws.Config{
		Region:      c.Aws.Region,
		Credentials: c.getAwsCredentialsProvider(),
	}

	if c.Aws.BaseEndpoint != "" {
		cfg.BaseEndpoint = aws.String(c.Aws.BaseEndpoint)
	}

	return cfg
}

func (c *Config) getCallbackPipelineConfig() pipeline.CallbackConfig {
	return pipeline.CallbackConfig{
		MaxRetries:    c.Callbacks.MaxRetries,
		RetryInterval: time.Duration(c.Callbacks.RetryInterval),
	}
}

func (c *Config) getSmtpConfig() smtp.Config {
	return smtp.Config{
		Host:             c.Smtp.Host,
		Port:             c.Smtp.Port,
		User:             c.Smtp.User,
		Password:         c.Smtp.Password,
		From:             c.Smtp.From,
		AllowInsecureTls: c.Smtp.AllowInsecureTls,
	}
}
