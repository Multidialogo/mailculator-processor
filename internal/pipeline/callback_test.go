//go:build unit

package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"mailculator-processor/internal/outbox"
	"mailculator-processor/internal/testutils/mocks"
)

type testServer struct {
	server          *httptest.Server
	statusCode      int
	calledDomain    string
	invocationCount int
}

func newTestServer(statusCode int) *testServer {
	ts := &testServer{invocationCount: 0, statusCode: statusCode}
	ts.server = httptest.NewServer(http.HandlerFunc(serverStatusOkHandleFunc(ts)))
	return ts
}

func serverStatusOkHandleFunc(ts *testServer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ts.calledDomain = r.Host
		ts.invocationCount++
		w.WriteHeader(ts.statusCode)
	}
}

func TestSuccessCallbackPipeline(t *testing.T) {
	outboxServiceMock := mocks.NewOutboxMock(mocks.Email(outbox.Email{Id: "1", Status: ""}))
	callbackConfig := CallbackConfig{RetryInterval: 2, MaxRetries: 3}

	callbacks := []*CallbackPipeline{
		NewSentCallbackPipeline(outboxServiceMock, callbackConfig),
		NewFailedCallbackPipeline(outboxServiceMock, callbackConfig),
		NewInvalidCallbackPipeline(outboxServiceMock, callbackConfig),
	}

	for _, callback := range callbacks {
		buf, logger := mocks.NewLoggerMock()
		ts := newTestServer(http.StatusOK)
		defer ts.server.Close()
		callback.cfg.Url = ts.server.URL
		callback.logger = logger
		callback.Process(context.TODO())

		assert.Contains(t, ts.server.URL, ts.calledDomain)
		assert.Equal(t, 1, ts.invocationCount)
		assert.Equal(t,
			"level=INFO msg=\"processing email 1\"\nlevel=INFO msg=\"callback successfully processed\" email=1",
			strings.TrimSpace(buf.String()),
		)
	}
}

func TestSanitizeReason_SMTPError_PreservesOriginal(t *testing.T) {
	assert.Equal(t, "552 5.3.4 Message too long", sanitizeReason("552 5.3.4 Message too long"))
	assert.Equal(t, "550 5.1.1 User unknown", sanitizeReason("550 5.1.1 User unknown"))
	assert.Equal(t, "421 Service not available", sanitizeReason("421 Service not available"))
}

func TestSanitizeReason_InternalError_ReturnsGeneric(t *testing.T) {
	assert.Equal(t, internalErrorReason, sanitizeReason("failed to read payload file /data/payloads/xyz.json: no such file or directory"))
	assert.Equal(t, internalErrorReason, sanitizeReason("dial tcp smtp-host:587: connection refused"))
	assert.Equal(t, internalErrorReason, sanitizeReason("failed to read attachment: open /data/attachments/file.pdf: permission denied"))
	assert.Equal(t, internalErrorReason, sanitizeReason("failed to unmarshal payload: invalid character"))
	assert.Equal(t, internalErrorReason, sanitizeReason("payload validation failed: Key: 'Payload.To' Error:Field validation"))
	assert.Equal(t, internalErrorReason, sanitizeReason(""))
}

func TestInvalidCallbackPipeline_SanitizesInternalReason(t *testing.T) {
	outboxServiceMock := mocks.NewOutboxMock(mocks.Email(outbox.Email{
		Id:     "1",
		Status: outbox.StatusInvalid,
		Reason: "payload validation failed: Key: 'Payload.To' Error:Field validation for 'To' failed",
	}))
	callbackConfig := CallbackConfig{RetryInterval: 2, MaxRetries: 3}

	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(bodyBytes, &receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	callbackConfig.Url = ts.URL
	buf, logger := mocks.NewLoggerMock()
	callback := NewInvalidCallbackPipeline(outboxServiceMock, callbackConfig)
	callback.logger = logger

	callback.Process(context.TODO())

	assert.Equal(t, "DISPATCH-ERROR", receivedBody["code"])
	assert.Equal(t, internalErrorReason, receivedBody["reason"])
	assert.Equal(t, []any{"1"}, receivedBody["message_ids"])
	assert.Contains(t, buf.String(), "callback successfully processed")
}

func TestFailedCallbackPipeline_PreservesSMTPReason(t *testing.T) {
	outboxServiceMock := mocks.NewOutboxMock(mocks.Email(outbox.Email{
		Id:     "1",
		Status: outbox.StatusFailed,
		Reason: "552 5.3.4 Message too long",
	}))
	callbackConfig := CallbackConfig{RetryInterval: 2, MaxRetries: 3}

	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(bodyBytes, &receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	callbackConfig.Url = ts.URL
	_, logger := mocks.NewLoggerMock()
	callback := NewFailedCallbackPipeline(outboxServiceMock, callbackConfig)
	callback.logger = logger

	callback.Process(context.TODO())

	assert.Equal(t, "DISPATCH-ERROR", receivedBody["code"])
	assert.Equal(t, "552 5.3.4 Message too long", receivedBody["reason"])
}

func TestFailedCallbackPipeline_SanitizesInternalReason(t *testing.T) {
	outboxServiceMock := mocks.NewOutboxMock(mocks.Email(outbox.Email{
		Id:     "1",
		Status: outbox.StatusFailed,
		Reason: "dial tcp smtp-host:587: connection refused",
	}))
	callbackConfig := CallbackConfig{RetryInterval: 2, MaxRetries: 3}

	var receivedBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(bodyBytes, &receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	callbackConfig.Url = ts.URL
	_, logger := mocks.NewLoggerMock()
	callback := NewFailedCallbackPipeline(outboxServiceMock, callbackConfig)
	callback.logger = logger

	callback.Process(context.TODO())

	assert.Equal(t, "DISPATCH-ERROR", receivedBody["code"])
	assert.Equal(t, internalErrorReason, receivedBody["reason"])
}

func TestQueryCallbackError(t *testing.T) {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(mocks.QueryMethodError(errors.New("some query error")))
	callbackConfig := CallbackConfig{Url: "", RetryInterval: 2, MaxRetries: 3}
	callback := NewSentCallbackPipeline(outboxServiceMock, callbackConfig)
	callback.logger = logger
	callback.Process(context.TODO())

	assert.Equal(t,
		"level=ERROR msg=\"error while querying emails to process: some query error\"",
		strings.TrimSpace(buf.String()),
	)
}

func TestLockUpdateError(t *testing.T) {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(mocks.UpdateMethodError(errors.New("some update error")))
	callbackConfig := CallbackConfig{Url: "", RetryInterval: 2, MaxRetries: 3}
	callback := NewSentCallbackPipeline(outboxServiceMock, callbackConfig)
	callback.logger = logger
	callback.Process(context.TODO())

	assert.Equal(t,
		"level=INFO msg=\"processing email \"\nlevel=WARN msg=\"failed to acquire processing lock, error: some update error\" email=\"\"",
		strings.TrimSpace(buf.String()),
	)
}

func TestHttpClientDoError(t *testing.T) {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: ""}),
		mocks.UpdateMethodError(errors.New("some update error")),
		mocks.UpdateMethodFailsCall(2),
	)
	callbackConfig := CallbackConfig{Url: "pippo://pluto.it", RetryInterval: 2, MaxRetries: 3}
	callback := NewSentCallbackPipeline(outboxServiceMock, callbackConfig)
	callback.logger = logger
	callback.Process(context.TODO())

	assert.Equal(t,
		"level=INFO msg=\"processing email 1\"\nlevel=ERROR msg=\"Error in the request: Post \\\"pippo://pluto.it\\\": unsupported protocol scheme \\\"pippo\\\"\" email=1",
		strings.TrimSpace(buf.String()),
	)
}

func TestAcknowledgedUpdateError(t *testing.T) {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: ""}),
		mocks.UpdateMethodError(errors.New("some update error")),
		mocks.UpdateMethodFailsCall(2),
	)
	ts := newTestServer(http.StatusOK)
	defer ts.server.Close()
	callbackConfig := CallbackConfig{Url: ts.server.URL, RetryInterval: 2, MaxRetries: 3}
	callback := NewSentCallbackPipeline(outboxServiceMock, callbackConfig)
	callback.logger = logger
	callback.Process(context.TODO())

	assert.Equal(t,
		"level=INFO msg=\"processing email 1\"\nlevel=INFO msg=\"callback successfully processed\" email=1\nlevel=ERROR msg=\"error while updating status after callback, error: some update error\" email=1",
		strings.TrimSpace(buf.String()),
	)
}

func TestStatusConflict(t *testing.T) {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(mocks.Email(outbox.Email{Id: "1", Status: ""}))
	ts := newTestServer(http.StatusConflict)
	defer ts.server.Close()
	callbackConfig := CallbackConfig{Url: ts.server.URL, RetryInterval: 2, MaxRetries: 3}
	callback := NewSentCallbackPipeline(outboxServiceMock, callbackConfig)
	callback.logger = logger
	callback.Process(context.TODO())

	assert.Contains(t, ts.server.URL, ts.calledDomain)
	assert.Equal(t, 3, ts.invocationCount)
	expectedMsgError := `level=INFO msg="processing email 1"
level=WARN msg="Response status code is 409. Try to call again {url} in 2 seconds. Attempt 1/3" email=1
level=WARN msg="Response status code is 409. Try to call again {url} in 2 seconds. Attempt 2/3" email=1
level=WARN msg="Response status code is 409. Attempt 3/3" email=1
level=ERROR msg="Max retries exceeded for the url {url}" email=1
level=ERROR msg="error on callback, status: 409, response: " email=1`
	assert.Equal(t, strings.ReplaceAll(expectedMsgError, "{url}", ts.server.URL), strings.TrimSpace(buf.String()))
}
