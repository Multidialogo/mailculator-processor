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
	mutex      sync.Mutex
	pool       []email.Email
	processed  []email.Email
	error      error
	findCalled int
	sendCalled int
}

func (m *emailServiceMock) LockAndReturnReadyToProcess(_ context.Context) ([]email.Email, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.findCalled++

	if m.error != nil {
		return nil, m.error
	}

	emails := m.pool
	m.pool = []email.Email{}
	return emails, nil
}

func (m *emailServiceMock) SendAndUnlock(_ context.Context, email email.Email) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.sendCalled++

	if m.error != nil {
		return m.error
	}

	m.processed = append(m.processed, email)
	return nil
}

func Test_RunUntilContextDone(t *testing.T) {
	pool := []email.Email{{Id: "1"}, {Id: "2"}}
	serviceMock := &emailServiceMock{pool: pool}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	sut := NewDaemon(serviceMock)
	sut.RunUntilContextDone(ctx)

	assert.Equal(t, 2, serviceMock.sendCalled)
	assert.ElementsMatch(t, pool, serviceMock.processed)
	assert.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
}
