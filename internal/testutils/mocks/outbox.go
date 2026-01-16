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
	requeueMethodError     error
	requeueMethodCall      int
	requeueMethodFailsCall int
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

func RequeueMethodError(requeueMethodError error) OutboxMockOptions {
	return func(o *OutboxMock) {
		o.requeueMethodError = requeueMethodError
	}
}

func RequeueMethodFailsCall(requeueMethodFailsCall int) OutboxMockOptions {
	return func(o *OutboxMock) {
		o.requeueMethodFailsCall = requeueMethodFailsCall
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
		requeueMethodError:     nil,
		requeueMethodCall:      0,
		requeueMethodFailsCall: 1,
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

func (m *OutboxMock) Update(ctx context.Context, id string, status string, errorReason string) error {
	m.lastMethod = "update"
	m.updateMethodCall++
	if m.updateMethodCall == m.updateMethodFailsCall {
		return m.updateMethodError
	}
	return nil
}

func (m *OutboxMock) Ready(ctx context.Context, id string, emlFilePath string) error {
	m.lastMethod = "ready"
	m.updateMethodCall++
	if m.updateMethodCall == m.updateMethodFailsCall {
		return m.updateMethodError
	}
	return nil
}

func (m *OutboxMock) Requeue(ctx context.Context, id string) error {
	m.lastMethod = "requeue"
	m.requeueMethodCall++
	if m.requeueMethodCall == m.requeueMethodFailsCall {
		return m.requeueMethodError
	}
	return nil
}

func (m *OutboxMock) LastMethod() string {
	return m.lastMethod
}
