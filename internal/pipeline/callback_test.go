//go:build unit

package pipeline

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/suite"
	"mailculator-processor/internal/outbox"
	"mailculator-processor/internal/testutils/mocks"
	"os/exec"
	"strings"
	"testing"
)

func TestCallbackTestSuite(t *testing.T) {
	suite.Run(t, &CallbackTestSuite{})
}

type CallbackTestSuite struct {
	suite.Suite
}

func getCallbackCommand(command string) string {
	cmd := exec.Command("which", "sh")
	shPath, _ := cmd.CombinedOutput()
	return fmt.Sprintf("%s -c %s", strings.TrimSpace(string(shPath)), command)
}

type callbackExecutorWrapper struct {
	callbackExecutor
	callback map[string]int
}

func (e *callbackExecutorWrapper) Execute(cmd *exec.Cmd) error {
	key := cmd.String()
	if _, ok := e.callback[key]; ok {
		e.callback[key]++
	} else {
		e.callback[key] = 1
	}
	return e.callbackExecutor.Execute(cmd)
}

func newCallbackExecutorWrapper() *callbackExecutorWrapper {
	return &callbackExecutorWrapper{callback: make(map[string]int)}
}

func (suite *CallbackTestSuite) TestSuccessCommandOnSentCallbackPipeline() {
	buf, logger := mocks.NewLoggerMock()
	command := "echo 'dummy text' | wc -l"
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: "", SuccessCallback: command, FailureCallback: ""}),
	)
	cExecutorMock := newCallbackExecutorWrapper()
	callback := NewSentCallbackPipeline(CallbackConfig{RetryInterval: 2, MaxRetries: 3}, outboxServiceMock)
	callback.callbackExecutor = cExecutorMock
	callback.logger = logger
	callback.Process(context.TODO())

	commandKey := getCallbackCommand(command)
	suite.Assert().Contains(cExecutorMock.callback, commandKey)
	suite.Assert().Equal(1, cExecutorMock.callback[commandKey])
	suite.Assert().Equal(
		"level=INFO msg=\"processing email 1\"",
		strings.TrimSpace(buf.String()),
	)
}

func (suite *CallbackTestSuite) TestErrorCommandOnSentCallbackPipeline() {
	buf, logger := mocks.NewLoggerMock()
	command := "dummy --command"
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: "", SuccessCallback: command, FailureCallback: ""}),
	)
	cExecutorMock := newCallbackExecutorWrapper()
	cConfig := CallbackConfig{RetryInterval: 2, MaxRetries: 3}
	callback := NewSentCallbackPipeline(cConfig, outboxServiceMock)
	callback.callbackExecutor = cExecutorMock
	callback.logger = logger
	callback.Process(context.TODO())

	commandKey := getCallbackCommand(command)
	suite.Assert().Contains(cExecutorMock.callback, commandKey)
	suite.Assert().Equal(cConfig.MaxRetries, cExecutorMock.callback[commandKey])
	suite.Assert().Equal(
		"level=INFO msg=\"processing email 1\"\nlevel=ERROR msg=\"error while executing callback, error: exec: already started\" email=1",
		strings.TrimSpace(buf.String()),
	)
}

func (suite *CallbackTestSuite) TestSuccessCommandOnFailedCallbackPipeline() {
	buf, logger := mocks.NewLoggerMock()
	command := "echo 'dummy text' | wc -l"
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: "", SuccessCallback: "", FailureCallback: command}),
	)
	cExecutorMock := newCallbackExecutorWrapper()
	callback := NewFailedCallbackPipeline(CallbackConfig{RetryInterval: 2, MaxRetries: 3}, outboxServiceMock)
	callback.callbackExecutor = cExecutorMock
	callback.logger = logger
	callback.Process(context.TODO())

	commandKey := getCallbackCommand(command)
	suite.Assert().Contains(cExecutorMock.callback, commandKey)
	suite.Assert().Equal(1, cExecutorMock.callback[commandKey])
	suite.Assert().Equal(
		"level=INFO msg=\"processing email 1\"",
		strings.TrimSpace(buf.String()),
	)
}

func (suite *CallbackTestSuite) TestErrorCommandOnFailedCallbackPipeline() {
	buf, logger := mocks.NewLoggerMock()
	command := "dummy --command"
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: "", SuccessCallback: "", FailureCallback: command}),
	)
	cExecutorMock := newCallbackExecutorWrapper()
	cConfig := CallbackConfig{RetryInterval: 2, MaxRetries: 3}
	callback := NewFailedCallbackPipeline(cConfig, outboxServiceMock)
	callback.callbackExecutor = cExecutorMock
	callback.logger = logger
	callback.Process(context.TODO())

	commandKey := getCallbackCommand(command)
	suite.Assert().Contains(cExecutorMock.callback, commandKey)
	suite.Assert().Equal(cConfig.MaxRetries, cExecutorMock.callback[commandKey])
	suite.Assert().Equal(
		"level=INFO msg=\"processing email 1\"\nlevel=ERROR msg=\"error while executing callback, error: exec: already started\" email=1",
		strings.TrimSpace(buf.String()),
	)
}

func (suite *CallbackTestSuite) TestQueryError() {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(mocks.QueryMethodError(errors.New("some query error")))
	callback := NewSentCallbackPipeline(CallbackConfig{RetryInterval: 2, MaxRetries: 3}, outboxServiceMock)
	callback.logger = logger
	callback.Process(context.TODO())

	suite.Assert().Equal(
		"level=ERROR msg=\"error while querying emails to process: some query error\"",
		strings.TrimSpace(buf.String()),
	)
}

func (suite *CallbackTestSuite) TestLockUpdateError() {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(mocks.UpdateMethodError(errors.New("some update error")))
	callback := NewSentCallbackPipeline(CallbackConfig{RetryInterval: 2, MaxRetries: 3}, outboxServiceMock)
	callback.logger = logger
	callback.Process(context.TODO())

	suite.Assert().Equal(
		"level=INFO msg=\"processing email \"\nlevel=WARN msg=\"failed to acquire processing lock, error: some update error\" email=\"\"",
		strings.TrimSpace(buf.String()),
	)
}

func (suite *CallbackTestSuite) TestRollbackUpdateError() {
	buf, logger := mocks.NewLoggerMock()
	command := "dummy --command"
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: "", SuccessCallback: command, FailureCallback: ""}),
		mocks.UpdateMethodError(errors.New("some update error")),
		mocks.UpdateMethodFailsCall(2),
	)
	callback := NewSentCallbackPipeline(CallbackConfig{RetryInterval: 2, MaxRetries: 3}, outboxServiceMock)
	callback.logger = logger
	callback.Process(context.TODO())

	suite.Assert().Equal(
		"level=INFO msg=\"processing email 1\"\nlevel=ERROR msg=\"error while executing callback, error: exec: already started\" email=1\nlevel=ERROR msg=\"error while rolling back status after callback error, error: exec: already started\" email=1",
		strings.TrimSpace(buf.String()),
	)
}

func (suite *CallbackTestSuite) TestAcknowledgedUpdateError() {
	buf, logger := mocks.NewLoggerMock()
	command := "echo 'dummy text' | wc -l"
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: "", SuccessCallback: command, FailureCallback: ""}),
		mocks.UpdateMethodError(errors.New("some update error")),
		mocks.UpdateMethodFailsCall(2),
	)
	callback := NewSentCallbackPipeline(CallbackConfig{RetryInterval: 2, MaxRetries: 3}, outboxServiceMock)
	callback.logger = logger
	callback.Process(context.TODO())

	suite.Assert().Equal(
		"level=INFO msg=\"processing email 1\"\nlevel=ERROR msg=\"error while updating status after callback, error: some update error\" email=1",
		strings.TrimSpace(buf.String()),
	)
}
