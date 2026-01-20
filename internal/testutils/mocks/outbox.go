package mocks

import (
	"context"
	"time"

	"mailculator-processor/internal/outbox"
)

type OutboxMock struct {
	queryMethodError      error
	updateMethodError     error
	updateMethodCall      int
	updateMethodFailsCall int
	queryStaleMethodError error
	updateFromMethodError error
	updateFromMethodCall  int
	updateFromFailsCall   int
	email                 outbox.Email
	lastMethod            string
}

type OutboxMockOptions func(*OutboxMock)

func QueryMethodError(queryMethodError error) OutboxMockOptions {
	return func(o *OutboxMock) {
		o.queryMethodError = queryMethodError
	}
}

func UpdateMethodError(updateMethodError error) OutboxMockOptions {
	return func(o *OutboxMock) {
		o.updateMethodError = updateMethodError
	}
}

func UpdateMethodFailsCall(updateMethodFailsCall int) OutboxMockOptions {
	return func(o *OutboxMock) {
		o.updateMethodFailsCall = updateMethodFailsCall
	}
}

func QueryStaleMethodError(queryStaleMethodError error) OutboxMockOptions {
	return func(o *OutboxMock) {
		o.queryStaleMethodError = queryStaleMethodError
	}
}

func UpdateFromMethodError(updateFromMethodError error) OutboxMockOptions {
	return func(o *OutboxMock) {
		o.updateFromMethodError = updateFromMethodError
	}
}

func UpdateFromMethodFailsCall(updateFromFailsCall int) OutboxMockOptions {
	return func(o *OutboxMock) {
		o.updateFromFailsCall = updateFromFailsCall
	}
}

func Email(email outbox.Email) OutboxMockOptions {
	return func(o *OutboxMock) {
		o.email = email
	}
}

func NewOutboxMock(opts ...OutboxMockOptions) *OutboxMock {
	o := &OutboxMock{
		queryMethodError:      nil,
		updateMethodError:     nil,
		updateMethodCall:      0,
		updateMethodFailsCall: 1,
		queryStaleMethodError: nil,
		updateFromMethodError: nil,
		updateFromMethodCall:  0,
		updateFromFailsCall:   1,
		email:                 outbox.Email{},
		lastMethod:            "",
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func (m *OutboxMock) Query(ctx context.Context, status string, limit int) ([]outbox.Email, error) {
	m.lastMethod = "query"
	return []outbox.Email{m.email}, m.queryMethodError
}

func (m *OutboxMock) QueryStale(ctx context.Context, status string, olderThan time.Duration, limit int) ([]outbox.Email, error) {
	m.lastMethod = "queryStale"
	return []outbox.Email{m.email}, m.queryStaleMethodError
}

func (m *OutboxMock) Update(ctx context.Context, id string, status string, errorReason string) error {
	m.lastMethod = "update"
	m.updateMethodCall++
	if m.updateMethodCall == m.updateMethodFailsCall {
		return m.updateMethodError
	}
	return nil
}

func (m *OutboxMock) Ready(ctx context.Context, id string) error {
	m.lastMethod = "ready"
	m.updateMethodCall++
	if m.updateMethodCall == m.updateMethodFailsCall {
		return m.updateMethodError
	}
	return nil
}

func (m *OutboxMock) UpdateFrom(ctx context.Context, id string, fromStatus string, toStatus string, errorReason string) error {
	m.lastMethod = "updateFrom"
	m.updateFromMethodCall++
	if m.updateFromMethodCall == m.updateFromFailsCall {
		return m.updateFromMethodError
	}
	return nil
}

func (m *OutboxMock) LastMethod() string {
	return m.lastMethod
}
