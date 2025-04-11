//go:build unit

package pipeline

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"mailculator-processor/internal/outbox"
	"mailculator-processor/internal/testutils/mocks"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
	outboxServiceMock := mocks.NewOutboxMock(mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: ""}))
	callbackConfig := CallbackConfig{RetryInterval: 2, MaxRetries: 3}

	callbacks := []*CallbackPipeline{
		NewSentCallbackPipeline(outboxServiceMock, callbackConfig),
		NewFailedCallbackPipeline(outboxServiceMock, callbackConfig),
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
			"level=INFO msg=\"processing email 1\"",
			strings.TrimSpace(buf.String()),
		)
	}
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
		mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: ""}),
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
		mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: ""}),
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
		"level=INFO msg=\"processing email 1\"\nlevel=ERROR msg=\"error while updating status after callback, error: some update error\" email=1",
		strings.TrimSpace(buf.String()),
	)
}

func TestStatusConflict(t *testing.T) {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: ""}))
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
level=ERROR msg="Max retries exceeded for the url {url}" email=1`
	assert.Equal(t, strings.ReplaceAll(expectedMsgError, "{url}", ts.server.URL), strings.TrimSpace(buf.String()))
}
