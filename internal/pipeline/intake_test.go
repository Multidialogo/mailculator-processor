//go:build unit

package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mailculator-processor/internal/email"
	"mailculator-processor/internal/outbox"
	"mailculator-processor/internal/testutils/mocks"
)

func createTestPayloadFile(t *testing.T, payload email.Payload) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "payload-*.json")
	require.NoError(t, err)

	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	_, err = tmpFile.Write(data)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}

func TestSuccessfulIntake(t *testing.T) {
	payload := email.Payload{
		Id:       "550e8400-e29b-41d4-a716-446655440000",
		From:     "sender@example.com",
		ReplyTo:  "reply@example.com",
		To:       "recipient@example.com",
		Subject:  "Test Subject",
		BodyHTML: "<html><body>Test</body></html>",
		BodyText: "Test",
	}

	payloadFile := createTestPayloadFile(t, payload)

	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{
			Id:              "1",
			Status:          outbox.StatusAccepted,
			PayloadFilePath: payloadFile,
		}),
	)

	buf, logger := mocks.NewLoggerMock()

	intake := NewIntakePipeline(outboxServiceMock)
	intake.logger = logger

	intake.Process(context.TODO())

	assert.Contains(t, buf.String(), "level=INFO msg=\"processing outbox 1\"")
	assert.Contains(t, buf.String(), "level=INFO msg=\"successfully intaken\" outbox=1")
}

func TestIntakeQueryError(t *testing.T) {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(mocks.QueryMethodError(errors.New("some query error")))

	intake := IntakePipeline{outbox: outboxServiceMock, logger: logger}

	intake.Process(context.TODO())

	assert.Equal(t, "level=ERROR msg=\"error while querying emails to process: some query error\"", strings.TrimSpace(buf.String()))
}

func TestIntakeUpdateError(t *testing.T) {
	payload := email.Payload{
		Id:       "550e8400-e29b-41d4-a716-446655440000",
		From:     "sender@example.com",
		ReplyTo:  "reply@example.com",
		To:       "recipient@example.com",
		Subject:  "Test Subject",
		BodyHTML: "<html><body>Test</body></html>",
	}

	payloadFile := createTestPayloadFile(t, payload)

	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{
			Id:              "1",
			Status:          outbox.StatusAccepted,
			PayloadFilePath: payloadFile,
		}),
		mocks.UpdateMethodError(errors.New("some update error")),
	)

	intake := IntakePipeline{outbox: outboxServiceMock, logger: logger}

	intake.Process(context.TODO())

	assert.Equal(t,
		"level=INFO msg=\"processing outbox 1\"\nlevel=WARN msg=\"failed to acquire processing lock, error: some update error\" outbox=1",
		strings.TrimSpace(buf.String()),
	)
}

func TestIntakeInvalidPayloadFile(t *testing.T) {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{
			Id:              "1",
			Status:          outbox.StatusAccepted,
			PayloadFilePath: "/nonexistent/file.json",
		}),
	)

	intake := NewIntakePipeline(outboxServiceMock)
	intake.logger = logger

	intake.Process(context.TODO())

	assert.Contains(t, buf.String(), "level=INFO msg=\"processing outbox 1\"")
	assert.Contains(t, buf.String(), "level=ERROR msg=\"failed to validate payload")
	assert.Contains(t, buf.String(), "failed to read payload file")
}

func TestIntakeInvalidJSON(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "payload-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("invalid json content")
	require.NoError(t, err)
	tmpFile.Close()

	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{
			Id:              "1",
			Status:          outbox.StatusAccepted,
			PayloadFilePath: tmpFile.Name(),
		}),
	)

	intake := NewIntakePipeline(outboxServiceMock)
	intake.logger = logger

	intake.Process(context.TODO())

	assert.Contains(t, buf.String(), "level=ERROR msg=\"failed to validate payload")
	assert.Contains(t, buf.String(), "failed to unmarshal payload")
}

func TestIntakeValidationError(t *testing.T) {
	// Invalid payload - missing required fields
	payload := email.Payload{
		Id:      "invalid-uuid",
		From:    "not-an-email",
		Subject: "Test",
	}

	payloadFile := createTestPayloadFile(t, payload)

	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{
			Id:              "1",
			Status:          outbox.StatusAccepted,
			PayloadFilePath: payloadFile,
		}),
	)

	intake := NewIntakePipeline(outboxServiceMock)
	intake.logger = logger

	intake.Process(context.TODO())

	assert.Contains(t, buf.String(), "level=ERROR msg=\"failed to validate payload")
	assert.Contains(t, buf.String(), "payload validation failed")
}
