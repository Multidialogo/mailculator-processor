//go:build unit

package pipeline

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"mailculator-processor/internal/outbox"
	"mailculator-processor/internal/testutils/mocks"
	"net/http"
	"strings"
	"testing"
)

type httpClientMock struct {
	calledDomain string
}

func (hc *httpClientMock) Do(req *http.Request) (*http.Response, error) {
	hc.calledDomain = req.URL.String()
	return &http.Response{}, nil
}

func TestSuccessCallbackPipeline(t *testing.T) {
	outboxServiceMock := mocks.NewOutboxMock(mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: ""}))
	dummyDomain := "dummy-domain.com"

	callbacks := []*CallbackPipeline{
		NewSentCallbackPipeline(outboxServiceMock, dummyDomain),
		NewFailedCallbackPipeline(outboxServiceMock, dummyDomain),
	}
	hcMock := &httpClientMock{}

	for _, callback := range callbacks {
		buf, logger := mocks.NewLoggerMock()
		callback.httpClient = hcMock
		callback.logger = logger
		callback.Process(context.TODO())

		assert.Equal(t, dummyDomain, hcMock.calledDomain)
		assert.Equal(t,
			"level=INFO msg=\"processing email 1\"",
			strings.TrimSpace(buf.String()),
		)
	}
}

func TestQueryCallbackError(t *testing.T) {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(mocks.QueryMethodError(errors.New("some query error")))
	callback := NewSentCallbackPipeline(outboxServiceMock, "")
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
	callback := NewSentCallbackPipeline(outboxServiceMock, "")
	callback.logger = logger
	callback.Process(context.TODO())

	assert.Equal(t,
		"level=INFO msg=\"processing email \"\nlevel=WARN msg=\"failed to acquire processing lock, error: some update error\" email=\"\"",
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
	callback := NewSentCallbackPipeline(outboxServiceMock, "")
	callback.logger = logger
	callback.Process(context.TODO())

	assert.Equal(t,
		"level=INFO msg=\"processing email 1\"\nlevel=ERROR msg=\"error while updating status after callback, error: some update error\" email=1",
		strings.TrimSpace(buf.String()),
	)
}

//func TestWannabe(t *testing.T) {
//	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		if r.Method != http.MethodPost {
//			t.Errorf("Method error: expected POST received %s", r.Method)
//		}
//
//		w.WriteHeader(http.StatusOK)
//		w.Write([]byte(`{"message": "ok"}`))
//	}))
//	defer server.Close()
//
//	outboxServiceMock := mocks.NewOutboxMock(mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: ""}))
//	callback := NewSentCallbackPipeline(outboxServiceMock, server.URL)
//	callback.Process(context.TODO())
//}
