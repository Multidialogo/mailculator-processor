//go:build unit

package pipeline

import (
	"context"
	"errors"
	"github.com/stretchr/testify/suite"
	"mailculator-processor/internal/outbox"
	"mailculator-processor/internal/testutils/mocks"
	"strings"
	"testing"
)

func TestSenderTestSuite(t *testing.T) {
	suite.Run(t, &SenderTestSuite{})
}

type SenderTestSuite struct {
	suite.Suite
}

type senderMock struct {
	sendMethodError   error
	sendMethodCounter int
}

func newSenderMock(sendMethodError error) *senderMock {
	return &senderMock{sendMethodError: sendMethodError, sendMethodCounter: 0}
}

func (m *senderMock) Send(emlFilePath string) error {
	if m.sendMethodError == nil {
		m.sendMethodCounter++
	}
	return m.sendMethodError
}

func (suite *SenderTestSuite) TestSucceededSendEmails() {
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: ""}),
	)
	senderServiceMock := newSenderMock(nil)
	buf, logger := mocks.NewLoggerMock()
	sender := NewMainSenderPipeline(outboxServiceMock, senderServiceMock)
	sender.logger = logger
	sender.Process(context.TODO())
	suite.Assert().Equal(1, senderServiceMock.sendMethodCounter)
	suite.Assert().Equal("level=INFO msg=\"processing outbox 1\"\nlevel=INFO msg=\"successfully sent\" outbox=1", strings.TrimSpace(buf.String()))
}

func (suite *SenderTestSuite) TestQueryError() {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(mocks.QueryMethodError(errors.New("some query error")))
	senderServiceMock := newSenderMock(nil)
	sender := MainSenderPipeline{outboxServiceMock, senderServiceMock, logger}

	sender.Process(context.TODO())

	suite.Assert().Equal(0, senderServiceMock.sendMethodCounter)
	suite.Assert().Equal("level=ERROR msg=\"error while querying emails to process: some query error\"", strings.TrimSpace(buf.String()))
}

func (suite *SenderTestSuite) TestUpdateError() {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: ""}),
		mocks.UpdateMethodError(errors.New("some update error")),
	)
	senderServiceMock := newSenderMock(nil)
	sender := MainSenderPipeline{outboxServiceMock, senderServiceMock, logger}

	sender.Process(context.TODO())

	suite.Assert().Equal(0, senderServiceMock.sendMethodCounter)
	suite.Assert().Equal(
		"level=INFO msg=\"processing outbox 1\"\nlevel=WARN msg=\"failed to acquire processing lock, error: some update error\" outbox=1",
		strings.TrimSpace(buf.String()),
	)
}

func (suite *SenderTestSuite) TestSendEmailError() {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: ""}),
	)
	senderServiceMock := newSenderMock(errors.New("some send error"))
	sender := MainSenderPipeline{outboxServiceMock, senderServiceMock, logger}

	sender.Process(context.TODO())

	suite.Assert().Equal(0, senderServiceMock.sendMethodCounter)
	suite.Assert().Equal(
		"level=INFO msg=\"processing outbox 1\"\nlevel=ERROR msg=\"failed to send, error: some send error\" outbox=1",
		strings.TrimSpace(buf.String()),
	)
}

func (suite *SenderTestSuite) TestHandleUpdateError() {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: ""}),
		mocks.UpdateMethodError(errors.New("some update error")),
		mocks.UpdateMethodFailsCall(2),
	)
	senderServiceMock := newSenderMock(nil)
	sender := MainSenderPipeline{outboxServiceMock, senderServiceMock, logger}

	sender.Process(context.TODO())

	suite.Assert().Equal(1, senderServiceMock.sendMethodCounter)
	suite.Assert().Equal(
		"level=INFO msg=\"processing outbox 1\"\nlevel=INFO msg=\"successfully sent\" outbox=1\nlevel=ERROR msg=\"error updating status to SENT, error: some update error\" outbox=1",
		strings.TrimSpace(buf.String()),
	)
}
