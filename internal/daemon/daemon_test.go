package daemon

import (
	"context"
	"github.com/stretchr/testify/assert"
	"mailculator-processor/internal/email"
	"sync"
	"testing"
	"time"
)

type emailFinderMock struct {
	mutex  sync.Mutex
	emails []email.Email
	error  error
	called int
}

func (m *emailFinderMock) FindAndLock() ([]email.Email, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.called++

	if m.error != nil {
		return nil, m.error
	}

	emails := m.emails
	m.emails = []email.Email{}
	return emails, nil
}

type emailSenderMock struct {
	mutex     sync.Mutex
	processed []email.Email
	error     error
	called    int
}

func (m *emailSenderMock) SendAndUnlock(ctx context.Context, email email.Email) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.called++

	if m.error != nil {
		return m.error
	}

	m.processed = append(m.processed, email)
	return nil
}

func Test_RunUntilContextDone(t *testing.T) {
	pool := []email.Email{{Id: "1"}, {Id: "2"}}
	finderMock := &emailFinderMock{emails: pool}
	senderMock := &emailSenderMock{}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	sut := NewDaemon(finderMock, senderMock)
	sut.RunUntilContextDone(ctx)

	assert.Less(t, 2, finderMock.called)
	assert.EqualValues(t, pool, senderMock.processed)
	assert.Equal(t, 2, senderMock.called)
	assert.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
}
