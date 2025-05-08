package config

import (
	"context"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/go-playground/validator/v10"

	"mailculator-processor/internal/pipeline"
	"mailculator-processor/internal/smtp"
)

type AwsConfig struct {
	BaseEndpoint string `yaml:"base_endpoint"`
	sdkConfig    aws.Config
}

type CallbacksConfig struct {
	MaxRetries    int    `yaml:"max_retries" validate:"required"`
	RetryInterval int    `yaml:"retry_interval" validate:"required"`
	Url           string `yaml:"url" validate:"required"`
}

type HealthCheckServerConfig struct {
	Port int `yaml:"port" validate:"required"`
}

type HealthCheckConfig struct {
	Server HealthCheckServerConfig `yaml:"server" validate:"required"`
}

type PipelineConfig struct {
	Interval int `yaml:"interval" validate:"required"`
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
	Aws         AwsConfig         `yaml:"aws,flow"`
	Callback    CallbacksConfig   `yaml:"callback" validate:"required"`
	HealthCheck HealthCheckConfig `yaml:"health-check,flow" validate:"required"`
	Pipeline    PipelineConfig    `yaml:"pipeline" validate:"required"`
	Smtp        SmtpConfig        `yaml:"smtp,flow" validate:"required"`
}

func NewFromYamlContent(yamlContent []byte) (*Config, error) {
	cfg := &Config{}
	yamlString := os.ExpandEnv(string(yamlContent))
	reader := strings.NewReader(yamlString)

	if err := cfg.load(reader); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) load(r io.Reader) error {
	decoder := yaml.NewDecoder(r)
	decoder.KnownFields(true)

	decodeErr := decoder.Decode(c)
	validate := validator.New(validator.WithRequiredStructEnabled())
	err := validate.Struct(c)

	if decodeErr != nil && err != nil {
		return fmt.Errorf("%w\n%w", err, decodeErr)
	}
	if decodeErr != nil {
		return decodeErr
	}
	if err != nil {
		return err
	}

	awsConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	if c.Aws.BaseEndpoint != "" {
		awsConfig.BaseEndpoint = aws.String(c.Aws.BaseEndpoint)
	}

	c.Aws.sdkConfig = awsConfig
	return nil
}

func (c *Config) GetAwsConfig() aws.Config {
	return c.Aws.sdkConfig
}

func (c *Config) GetCallbackConfig() pipeline.CallbackConfig {
	return pipeline.CallbackConfig{
		MaxRetries:    c.Callback.MaxRetries,
		RetryInterval: time.Duration(c.Callback.RetryInterval),
		Url:           c.Callback.Url,
	}
}

func (c *Config) GetHealthCheckServerPort() int {
	return c.HealthCheck.Server.Port
}

func (c *Config) GetPipelineInterval() int {
	return c.Pipeline.Interval
}

func (c *Config) GetSmtpConfig() smtp.Config {
	return smtp.Config{
		Host:             c.Smtp.Host,
		Port:             c.Smtp.Port,
		User:             c.Smtp.User,
		Password:         c.Smtp.Password,
		From:             c.Smtp.From,
		AllowInsecureTls: c.Smtp.AllowInsecureTls,
	}
}
