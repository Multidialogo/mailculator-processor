//go:build unit

package pipeline

import (
	"context"
	"errors"
	"github.com/stretchr/testify/suite"
	"mailculator-processor/internal/outbox"
	"mailculator-processor/internal/testutils/mocks"
	"net/http"
	"strings"
	"testing"
)

func TestCallbackTestSuite(t *testing.T) {
	suite.Run(t, &CallbackTestSuite{})
}

type CallbackTestSuite struct {
	suite.Suite
}

type httpClientMock struct {
	calledDomain string
}

func (hc *httpClientMock) Do(req *http.Request) (*http.Response, error) {
	hc.calledDomain = req.URL.String()
	return &http.Response{}, nil
}

func (suite *CallbackTestSuite) TestSuccess_SentCallbackPipeline() {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: ""}))
	dummyDomain := "dummy-domain.com"
	hcMock := &httpClientMock{}
	callback := NewSentCallbackPipeline(outboxServiceMock, dummyDomain)
	callback.httpClient = hcMock
	callback.logger = logger
	callback.Process(context.TODO())

	suite.Assert().Equal(dummyDomain, hcMock.calledDomain)
	suite.Assert().Equal(
		"level=INFO msg=\"processing email 1\"",
		strings.TrimSpace(buf.String()),
	)
}

func (suite *CallbackTestSuite) TestSuccess_FailedCallbackPipeline() {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: ""}))
	dummyDomain := "dummy-domain.com"
	hcMock := &httpClientMock{}
	callback := NewFailedCallbackPipeline(outboxServiceMock, dummyDomain)
	callback.httpClient = hcMock
	callback.logger = logger
	callback.Process(context.TODO())

	suite.Assert().Equal(dummyDomain, hcMock.calledDomain)
	suite.Assert().Equal(
		"level=INFO msg=\"processing email 1\"",
		strings.TrimSpace(buf.String()),
	)
}

func (suite *CallbackTestSuite) TestQueryError() {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(mocks.QueryMethodError(errors.New("some query error")))
	callback := NewSentCallbackPipeline(outboxServiceMock, "")
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
	callback := NewSentCallbackPipeline(outboxServiceMock, "")
	callback.logger = logger
	callback.Process(context.TODO())

	suite.Assert().Equal(
		"level=INFO msg=\"processing email \"\nlevel=WARN msg=\"failed to acquire processing lock, error: some update error\" email=\"\"",
		strings.TrimSpace(buf.String()),
	)
}

func (suite *CallbackTestSuite) TestAcknowledgedUpdateError() {
	buf, logger := mocks.NewLoggerMock()
	outboxServiceMock := mocks.NewOutboxMock(
		mocks.Email(outbox.Email{Id: "1", Status: "", EmlFilePath: ""}),
		mocks.UpdateMethodError(errors.New("some update error")),
		mocks.UpdateMethodFailsCall(2),
	)
	callback := NewSentCallbackPipeline(outboxServiceMock, "")
	callback.logger = logger
	callback.Process(context.TODO())

	suite.Assert().Equal(
		"level=INFO msg=\"processing email 1\"\nlevel=ERROR msg=\"error while updating status after callback, error: some update error\" email=1",
		strings.TrimSpace(buf.String()),
	)
}
