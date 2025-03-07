package daemon

import (
	"context"
	"github.com/stretchr/testify/suite"
	"mailculator-processor/internal/email"
	"sync"
	"testing"
	"time"
)

type emailServiceMock struct {
	mutex       sync.Mutex
	pool        []email.Email
	locked      []email.Email
	readyCalled int
}

func (m *emailServiceMock) FindReady(_ context.Context) ([]email.Email, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.readyCalled++

	emails := m.pool
	m.pool = []email.Email{}
	return emails, nil
}

func (m *emailServiceMock) UpdateStatus(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *emailServiceMock) BatchLock(_ context.Context, emails []email.Email) ([]email.Email, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.locked = append(m.locked, emails...)
	return emails, nil
}

type emailClientMock struct {
	mutex  sync.Mutex
	sent   []email.Email
	closed int
}

func (m *emailClientMock) Send(email email.Email) (bool, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.sent = append(m.sent, email)
	return true, nil
}

func (m *emailClientMock) Close() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.closed++
}

type emailClientMockFactory struct {
	client *emailClientMock
}

func (f *emailClientMockFactory) New() (*emailClientMock, error) {
	return f.client, nil
}

type shellCommandMock struct {
	command  string
	executed bool
}

func (m *shellCommandMock) Execute() error {
	m.executed = true
	return nil
}

type shellCommandMockFactory struct {
	mutex        sync.Mutex
	shellCommand *shellCommandMock
	created      map[string][]*shellCommandMock
}

func (m *shellCommandMockFactory) New(command string) *shellCommandMock {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	created := &shellCommandMock{command: command}
	m.created[command] = append(m.created[command], created)
	return created
}

func TestDaemonTestSuite(t *testing.T) {
	suite.Run(t, &DaemonTestSuite{})
}

type DaemonTestSuite struct {
	suite.Suite
	serviceMock             *emailServiceMock
	clientMock              *emailClientMock
	clientMockFactory       *emailClientMockFactory
	shellCommandMockFactory *shellCommandMockFactory
}

func (suite *DaemonTestSuite) SetupTest() {
	suite.serviceMock = &emailServiceMock{pool: []email.Email{}}
	suite.clientMock = &emailClientMock{}
	suite.clientMockFactory = &emailClientMockFactory{client: suite.clientMock}
	suite.shellCommandMockFactory = &shellCommandMockFactory{created: map[string][]*shellCommandMock{}}
}

func (suite *DaemonTestSuite) Test_RunUntilContextDone_WhenTwoEmailsAreReady_ShouldLockSendAndCallbackTwoEmails() {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	firstCallback := "echo callback --x"
	secondCallback := "echo callback --y"

	readyPool := []email.Email{{Id: "1", SuccessCallback: firstCallback}, {Id: "2", SuccessCallback: secondCallback}}
	suite.serviceMock.pool = readyPool

	sut := NewDaemon(suite.serviceMock, suite.clientMockFactory, suite.shellCommandMockFactory)
	sut.RunUntilContextDone(ctx)

	// "ready" must be called many times
	suite.Assert().True(suite.serviceMock.readyCalled > 2)

	// email in pool should be sent and locked
	suite.Assert().ElementsMatch(readyPool, suite.clientMock.sent)
	suite.Assert().ElementsMatch(readyPool, suite.serviceMock.locked)

	// first callback must have been called exactly once
	_, firstCallbackHasBeenCreated := suite.shellCommandMockFactory.created[firstCallback]
	suite.Assert().True(firstCallbackHasBeenCreated)
	if firstCallbackHasBeenCreated && len(suite.shellCommandMockFactory.created[firstCallback]) > 0 {
		suite.Assert().Len(suite.shellCommandMockFactory.created[firstCallback], 1)
		suite.Assert().True(suite.shellCommandMockFactory.created[firstCallback][0].executed)
	}

	// second callback must have been called exactly once
	_, secondCallbackHasBeenCreated := suite.shellCommandMockFactory.created[secondCallback]
	suite.Assert().True(secondCallbackHasBeenCreated)
	if secondCallbackHasBeenCreated && len(suite.shellCommandMockFactory.created[secondCallback]) > 0 {
		suite.Assert().Len(suite.shellCommandMockFactory.created[secondCallback], 1)
		suite.Assert().True(suite.shellCommandMockFactory.created[secondCallback][0].executed)
	}

	// it should stop because of deadline
	suite.Assert().ErrorIs(ctx.Err(), context.DeadlineExceeded)
}
