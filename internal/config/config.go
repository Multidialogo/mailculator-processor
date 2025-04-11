package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"mailculator-processor/internal/pipeline"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/go-playground/validator/v10"
	"mailculator-processor/internal/smtp"
)

type AwsConfig struct {
	BaseEndpoint string `yaml:"base_endpoint"`
	Key          string `yaml:"key" validate:"required"`
	Secret       string `yaml:"secret" validate:"required"`
	Region       string `yaml:"region" validate:"required"`
}

type CallbacksConfig struct {
	MaxRetries    int    `yaml:"max_retries" validate:"required"`
	RetryInterval int    `yaml:"retry_interval" validate:"required"`
	Url           string `yaml:"url" validate:"required"`
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
	Aws      AwsConfig       `yaml:"aws,flow" validate:"required"`
	Callback CallbacksConfig `yaml:"callback" validate:"required"`
	Pipeline PipelineConfig  `yaml:"pipeline" validate:"required"`
	Smtp     SmtpConfig      `yaml:"smtp,flow" validate:"required"`
}

func NewFromYaml(filePath string) (*Config, error) {
	config := &Config{}

	yamlData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	yamlString := os.ExpandEnv(string(yamlData))

	reader := strings.NewReader(yamlString)

	if err = config.load(reader); err != nil {
		return nil, err
	}

	return config, nil
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
	return err
}

func (c *Config) getAwsCredentialsProvider() credentials.StaticCredentialsProvider {
	return credentials.NewStaticCredentialsProvider(
		c.Aws.Key,
		c.Aws.Secret,
		"",
	)
}

func (c *Config) GetAwsConfig() aws.Config {
	cfg := aws.Config{
		Region:      c.Aws.Region,
		Credentials: c.getAwsCredentialsProvider(),
	}

	if c.Aws.BaseEndpoint != "" {
		cfg.BaseEndpoint = aws.String(c.Aws.BaseEndpoint)
	}

	return cfg
}

func (c *Config) GetCallbackConfig() pipeline.CallbackConfig {
	return pipeline.CallbackConfig{
		MaxRetries:    c.Callback.MaxRetries,
		RetryInterval: time.Duration(c.Callback.RetryInterval),
		Url:           c.Callback.Url,
	}
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
