//go:build unit

package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"net/textproto"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mailculator-processor/internal/email"
	"mailculator-processor/internal/outbox"
	"mailculator-processor/internal/testutils/mocks"
)

type senderMock struct {
	sendMethodError   error
	sendMethodCounter int
}

func newSenderMock(sendMethodError error) *senderMock {
	return &senderMock{sendMethodError: sendMethodError, sendMethodCounter: 0}
}

func (m *senderMock) Send(payload email.Payload, attachmentsBasePath string) error {
	if m.sendMethodError == nil {
		m.sendMethodCounter++
	}
	return m.sendMethodError
}

func createPayloadFile(t *testing.T) string {
	t.Helper()

	payload := email.Payload{
		Id:       "550e8400-e29b-41d4-a716-446655440000",
		From:     "sender@example.com",
		ReplyTo:  "reply@example.com",
		To:       "recipient@example.com",
		Subject:  "Test Subject",
		BodyText: "Test body",
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	tmpFile, err := os.CreateTemp("", "payload-*.json")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(tmpFile.Name()) })

	_, err = tmpFile.Write(data)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	return tmpFile.Name()
}

func TestSucceededSendEmails(t *testing.T) {
	payloadFile := createPayloadFile(t)
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", PayloadFilePath: payloadFile}),
	)
	senderServiceMock := newSenderMock(nil)
	buf, logger := mocks.NewLoggerMock()
	sender := NewMainSenderPipeline(outboxServiceMock, senderServiceMock, "/base/path/")
	sender.logger = logger
	sender.Process(context.TODO())
	assert.Equal(t, 1, senderServiceMock.sendMethodCounter)
	assert.Equal(t, "level=INFO msg=\"processing outbox 1\"\nlevel=INFO msg=\"successfully sent\" outbox=1", strings.TrimSpace(buf.String()))
}

func TestQueryEmailError(t *testing.T) {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(mocks.QueryMethodError(errors.New("some query error")))
	senderServiceMock := newSenderMock(nil)
	sender := MainSenderPipeline{outbox: outboxServiceMock, client: senderServiceMock, attachmentsBasePath: "/base/path/", logger: logger}

	sender.Process(context.TODO())

	assert.Equal(t, 0, senderServiceMock.sendMethodCounter)
	assert.Equal(t, "level=ERROR msg=\"error while querying emails to process: some query error\"", strings.TrimSpace(buf.String()))
}

func TestUpdateError(t *testing.T) {
	payloadFile := createPayloadFile(t)
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", PayloadFilePath: payloadFile}),
		mocks.UpdateMethodError(errors.New("some update error")),
	)
	senderServiceMock := newSenderMock(nil)
	sender := MainSenderPipeline{outbox: outboxServiceMock, client: senderServiceMock, attachmentsBasePath: "/base/path/", logger: logger}

	sender.Process(context.TODO())

	assert.Equal(t, 0, senderServiceMock.sendMethodCounter)
	assert.Equal(t,
		"level=INFO msg=\"processing outbox 1\"\nlevel=WARN msg=\"failed to acquire processing lock, error: some update error\" outbox=1",
		strings.TrimSpace(buf.String()),
	)
}

func TestSendEmailError(t *testing.T) {
	payloadFile := createPayloadFile(t)
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", PayloadFilePath: payloadFile}),
	)
	senderServiceMock := newSenderMock(errors.New("some send error"))
	sender := MainSenderPipeline{outbox: outboxServiceMock, client: senderServiceMock, attachmentsBasePath: "/base/path/", logger: logger}

	sender.Process(context.TODO())

	assert.Equal(t, 0, senderServiceMock.sendMethodCounter)
	assert.Equal(t,
		"level=INFO msg=\"processing outbox 1\"\nlevel=ERROR msg=\"failed to send, error: some send error\" outbox=1",
		strings.TrimSpace(buf.String()),
	)
}

func TestSendEmailThrottlingRequeue(t *testing.T) {
	payloadFile := createPayloadFile(t)
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", PayloadFilePath: payloadFile}),
	)
	senderServiceMock := newSenderMock(&textproto.Error{Code: 454, Msg: "Throttling failure"})
	sender := MainSenderPipeline{outbox: outboxServiceMock, client: senderServiceMock, attachmentsBasePath: "/base/path/", logger: logger}

	sender.Process(context.TODO())

	assert.Equal(t, 0, senderServiceMock.sendMethodCounter)
	assert.Equal(t, "requeue", outboxServiceMock.LastMethod())
	assert.Equal(t,
		"level=INFO msg=\"processing outbox 1\"\nlevel=WARN msg=\"smtp throttling, requeueing: 454 Throttling failure\" outbox=1",
		strings.TrimSpace(buf.String()),
	)
}

func TestHandleUpdateError(t *testing.T) {
	payloadFile := createPayloadFile(t)
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", PayloadFilePath: payloadFile}),
		mocks.UpdateMethodError(errors.New("some update error")),
		mocks.UpdateMethodFailsCall(2),
	)
	senderServiceMock := newSenderMock(nil)
	sender := MainSenderPipeline{outbox: outboxServiceMock, client: senderServiceMock, attachmentsBasePath: "/base/path/", logger: logger}

	sender.Process(context.TODO())

	assert.Equal(t, 1, senderServiceMock.sendMethodCounter)
	assert.Equal(t,
		"level=INFO msg=\"processing outbox 1\"\nlevel=INFO msg=\"successfully sent\" outbox=1\nlevel=ERROR msg=\"error updating status to SENT, error: some update error\" outbox=1",
		strings.TrimSpace(buf.String()),
	)
}
