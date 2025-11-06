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

	"mailculator-processor/internal/eml"
	"mailculator-processor/internal/outbox"
	"mailculator-processor/internal/testutils/mocks"
)

type emlStorageMock struct {
	storeMethodError   error
	storeMethodCounter int
	storedPath         string
}

func newEmlStorageMock(storeMethodError error, storedPath string) *emlStorageMock {
	return &emlStorageMock{
		storeMethodError:   storeMethodError,
		storeMethodCounter: 0,
		storedPath:         storedPath,
	}
}

func (m *emlStorageMock) Store(emlData eml.EML) (string, error) {
	if m.storeMethodError == nil {
		m.storeMethodCounter++
	}
	return m.storedPath, m.storeMethodError
}

func createTestPayloadFile(t *testing.T, payload EmailPayload) string {
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
	payload := EmailPayload{
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
			TTL:             1234567890,
		}),
	)
	
	emlStorageMock := newEmlStorageMock(nil, "/path/to/eml/file.eml")
	buf, logger := mocks.NewLoggerMock()
	
	intake := NewIntakePipeline(outboxServiceMock, emlStorageMock)
	intake.logger = logger
	
	intake.Process(context.TODO())
	
	assert.Equal(t, 1, emlStorageMock.storeMethodCounter)
	assert.Contains(t, buf.String(), "level=INFO msg=\"processing outbox 1\"")
	assert.Contains(t, buf.String(), "level=INFO msg=\"successfully intaken\" outbox=1")
}

func TestIntakeQueryError(t *testing.T) {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(mocks.QueryMethodError(errors.New("some query error")))
	emlStorageMock := newEmlStorageMock(nil, "/path/to/eml/file.eml")
	
	intake := IntakePipeline{outboxServiceMock, emlStorageMock, logger, nil}
	
	intake.Process(context.TODO())
	
	assert.Equal(t, 0, emlStorageMock.storeMethodCounter)
	assert.Equal(t, "level=ERROR msg=\"error while querying emails to process: some query error\"", strings.TrimSpace(buf.String()))
}

func TestIntakeUpdateError(t *testing.T) {
	payload := EmailPayload{
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
			TTL:             1234567890,
		}),
		mocks.UpdateMethodError(errors.New("some update error")),
	)
	
	emlStorageMock := newEmlStorageMock(nil, "/path/to/eml/file.eml")
	intake := IntakePipeline{outboxServiceMock, emlStorageMock, logger, nil}
	
	intake.Process(context.TODO())
	
	assert.Equal(t, 0, emlStorageMock.storeMethodCounter)
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
			TTL:             1234567890,
		}),
	)
	
	emlStorageMock := newEmlStorageMock(nil, "/path/to/eml/file.eml")
	intake := NewIntakePipeline(outboxServiceMock, emlStorageMock)
	intake.logger = logger
	
	intake.Process(context.TODO())
	
	assert.Equal(t, 0, emlStorageMock.storeMethodCounter)
	assert.Contains(t, buf.String(), "level=INFO msg=\"processing outbox 1\"")
	assert.Contains(t, buf.String(), "level=ERROR msg=\"failed to intake")
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
			TTL:             1234567890,
		}),
	)
	
	emlStorageMock := newEmlStorageMock(nil, "/path/to/eml/file.eml")
	intake := NewIntakePipeline(outboxServiceMock, emlStorageMock)
	intake.logger = logger
	
	intake.Process(context.TODO())
	
	assert.Equal(t, 0, emlStorageMock.storeMethodCounter)
	assert.Contains(t, buf.String(), "level=ERROR msg=\"failed to intake")
	assert.Contains(t, buf.String(), "failed to unmarshal payload")
}

func TestIntakeValidationError(t *testing.T) {
	// Invalid payload - missing required fields
	payload := EmailPayload{
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
			TTL:             1234567890,
		}),
	)
	
	emlStorageMock := newEmlStorageMock(nil, "/path/to/eml/file.eml")
	intake := NewIntakePipeline(outboxServiceMock, emlStorageMock)
	intake.logger = logger
	
	intake.Process(context.TODO())
	
	assert.Equal(t, 0, emlStorageMock.storeMethodCounter)
	assert.Contains(t, buf.String(), "level=ERROR msg=\"failed to intake")
	assert.Contains(t, buf.String(), "payload validation failed")
}

func TestIntakeStorageError(t *testing.T) {
	payload := EmailPayload{
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
			TTL:             1234567890,
		}),
	)
	
	emlStorageMock := newEmlStorageMock(errors.New("storage error"), "")
	intake := NewIntakePipeline(outboxServiceMock, emlStorageMock)
	intake.logger = logger
	
	intake.Process(context.TODO())
	
	assert.Equal(t, 0, emlStorageMock.storeMethodCounter)
	assert.Contains(t, buf.String(), "level=ERROR msg=\"failed to intake")
	assert.Contains(t, buf.String(), "failed to store EML")
}

