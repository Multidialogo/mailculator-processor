package mocks

import (
	"context"
	"mailculator-processor/internal/outbox"
)

type OutboxMock struct {
	queryMethodError      error
	updateMethodError     error
	updateMethodCall      int
	updateMethodFailsCall int
	email                 outbox.Email
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
		email:                 outbox.Email{},
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func (m *OutboxMock) Query(ctx context.Context, status string, limit int) ([]outbox.Email, error) {
	return []outbox.Email{m.email}, m.queryMethodError
}

func (m *OutboxMock) Update(ctx context.Context, id string, status string, errorReason string) error {
	m.updateMethodCall++
	if m.updateMethodCall == m.updateMethodFailsCall {
		return m.updateMethodError
	}
	return nil
}
