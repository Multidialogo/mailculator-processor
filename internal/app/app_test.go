package app

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/stretchr/testify/suite"
	"mailculator-processor/internal/pipeline"
	"mailculator-processor/internal/smtp"
	"testing"
	"time"
)

func TestAppTestSuite(t *testing.T) {
	suite.Run(t, &AppTestSuite{})
}

type AppTestSuite struct {
	suite.Suite
}

type mockConfigProvider struct{}

func newMockConfigProvider() *mockConfigProvider {
	return &mockConfigProvider{}
}

func (cp *mockConfigProvider) GetAwsConfig() aws.Config {
	return aws.Config{
		Region:       "dummy-region",
		Credentials:  credentials.NewStaticCredentialsProvider("dummy-key", "dummy-secret", "dummy-session"),
		BaseEndpoint: aws.String("dummy-endpoint"),
	}
}

func (cp *mockConfigProvider) GetCallbackPipelineConfig() pipeline.CallbackConfig {
	return pipeline.CallbackConfig{MaxRetries: 2, RetryInterval: 3}
}

func (cp *mockConfigProvider) GetSmtpConfig() smtp.Config {
	return smtp.Config{
		Host:             "dummy-host",
		Port:             1234,
		User:             "dummy-user",
		Password:         "dummy-password",
		From:             "dummy-from",
		AllowInsecureTls: false,
	}
}

func (suite *AppTestSuite) TestAppInstance() {
	app, errNew := New(newMockConfigProvider())
	suite.Require().NoError(errNew)
	suite.Require().Equal(3, len(app.pipes))
	suite.Assert().NotZero(app.pipes[0])
	suite.Assert().NotZero(app.pipes[1])
	suite.Assert().NotZero(app.pipes[2])
}

type mockProcessor struct {
	sleepMilliseconds int
	calls             int
}

func newMockProcessor(sleepMilliseconds int) *mockProcessor {
	return &mockProcessor{sleepMilliseconds: sleepMilliseconds, calls: 0}
}

func (t *mockProcessor) Process(ctx context.Context) {
	time.Sleep(time.Duration(t.sleepMilliseconds) * time.Millisecond)
	t.calls++
}

func (suite *AppTestSuite) TestRunFunction() {
	proc1 := newMockProcessor(200)
	proc2 := newMockProcessor(200)
	app := &App{pipes: []pipelineProcessor{proc1, proc2}}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	app.Run(ctx)

	suite.Assert().Equal(1, proc1.calls)
	suite.Assert().Equal(1, proc2.calls)
}
