package daemon

import (
	"context"
	"github.com/stretchr/testify/assert"
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

type callbackExecutorMock struct{}

func (c *callbackExecutorMock) Execute(_ context.Context, callback string) error {
	return nil
}

func Test_RunUntilContextDone(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	pool := []email.Email{{Id: "1"}, {Id: "2"}}
	service := &emailServiceMock{pool: pool}
	client := &emailClientMock{}
	clientFactory := &emailClientMockFactory{client: client}
	executor := &callbackExecutorMock{}

	sut := NewDaemon(service, executor, clientFactory)
	sut.RunUntilContextDone(ctx)

	assert.Less(t, 2, service.readyCalled)
	assert.ElementsMatch(t, pool, client.sent)
	assert.ElementsMatch(t, pool, service.locked)
	assert.Equal(t, 1, client.closed)
	assert.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
}
