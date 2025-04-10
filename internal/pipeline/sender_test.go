//go:build unit

package pipeline

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"mailculator-processor/internal/outbox"
	"mailculator-processor/internal/testutils/mocks"
	"strings"
	"testing"
)

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

func TestSucceededSendEmails(t *testing.T) {
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: ""}),
	)
	senderServiceMock := newSenderMock(nil)
	buf, logger := mocks.NewLoggerMock()
	sender := NewMainSenderPipeline(outboxServiceMock, senderServiceMock)
	sender.logger = logger
	sender.Process(context.TODO())
	assert.Equal(t, 1, senderServiceMock.sendMethodCounter)
	assert.Equal(t, "level=INFO msg=\"processing outbox 1\"\nlevel=INFO msg=\"successfully sent\" outbox=1", strings.TrimSpace(buf.String()))
}

func TestQueryEmailError(t *testing.T) {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(mocks.QueryMethodError(errors.New("some query error")))
	senderServiceMock := newSenderMock(nil)
	sender := MainSenderPipeline{outboxServiceMock, senderServiceMock, logger}

	sender.Process(context.TODO())

	assert.Equal(t, 0, senderServiceMock.sendMethodCounter)
	assert.Equal(t, "level=ERROR msg=\"error while querying emails to process: some query error\"", strings.TrimSpace(buf.String()))
}

func TestUpdateError(t *testing.T) {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: ""}),
		mocks.UpdateMethodError(errors.New("some update error")),
	)
	senderServiceMock := newSenderMock(nil)
	sender := MainSenderPipeline{outboxServiceMock, senderServiceMock, logger}

	sender.Process(context.TODO())

	assert.Equal(t, 0, senderServiceMock.sendMethodCounter)
	assert.Equal(t,
		"level=INFO msg=\"processing outbox 1\"\nlevel=WARN msg=\"failed to acquire processing lock, error: some update error\" outbox=1",
		strings.TrimSpace(buf.String()),
	)
}

func TestSendEmailError(t *testing.T) {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: ""}),
	)
	senderServiceMock := newSenderMock(errors.New("some send error"))
	sender := MainSenderPipeline{outboxServiceMock, senderServiceMock, logger}

	sender.Process(context.TODO())

	assert.Equal(t, 0, senderServiceMock.sendMethodCounter)
	assert.Equal(t,
		"level=INFO msg=\"processing outbox 1\"\nlevel=ERROR msg=\"failed to send, error: some send error\" outbox=1",
		strings.TrimSpace(buf.String()),
	)
}

func TestHandleUpdateError(t *testing.T) {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: ""}),
		mocks.UpdateMethodError(errors.New("some update error")),
		mocks.UpdateMethodFailsCall(2),
	)
	senderServiceMock := newSenderMock(nil)
	sender := MainSenderPipeline{outboxServiceMock, senderServiceMock, logger}

	sender.Process(context.TODO())

	assert.Equal(t, 1, senderServiceMock.sendMethodCounter)
	assert.Equal(t,
		"level=INFO msg=\"processing outbox 1\"\nlevel=INFO msg=\"successfully sent\" outbox=1\nlevel=ERROR msg=\"error updating status to SENT, error: some update error\" outbox=1",
		strings.TrimSpace(buf.String()),
	)
}
