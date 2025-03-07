package daemon

import (
	"context"
	"log"
	"mailculator-processor/internal/email"
	"sync"
)

type emailService interface {
	FindReady(context.Context) ([]email.Email, error)
	UpdateStatus(context.Context, string, string) error
	BatchLock(context.Context, []email.Email) ([]email.Email, error)
}

type callbackExecutor interface {
	Execute(context.Context, string) error
}

type emailClient interface {
	Send(email.Email) (bool, error)
	Close()
}

type emailClientFactory[T emailClient] interface {
	New() (T, error)
}

type Daemon[T emailClient] struct {
	service          emailService
	callbackExecutor callbackExecutor
	clientFactory    emailClientFactory[T]
}

func NewDaemon[T emailClient](service emailService, callbackExecutor callbackExecutor, clientFactory emailClientFactory[T]) *Daemon[T] {
	return &Daemon[T]{
		service:          service,
		callbackExecutor: callbackExecutor,
		clientFactory:    clientFactory,
	}
}

func (d *Daemon[T]) RunUntilContextDone(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			d.runSingleIteration(ctx)
		}
	}
}

func (d *Daemon[T]) runSingleIteration(ctx context.Context) {
	foundEmails, err := d.service.FindReady(ctx)
	if err != nil {
		log.Print(err.Error())
		return
	}

	if len(foundEmails) == 0 {
		return
	}

	lockedEmails, err := d.service.BatchLock(ctx, foundEmails)
	if err != nil || len(lockedEmails) == 0 {
		log.Print("no locks acquired")
		return
	}

	client, err := d.clientFactory.New()
	if err != nil {
		log.Print(err.Error())
		return
	}

	defer client.Close()
	var wg sync.WaitGroup

	for _, currentEmail := range lockedEmails {
		wg.Add(1)

		go func(wg *sync.WaitGroup, email email.Email) {
			defer wg.Done()

			_, clientErr := client.Send(email)
			if clientErr != nil {
				d.handleFailure(ctx, email)
				return
			}

			d.handleSuccess(ctx, email)
		}(&wg, currentEmail)
	}

	wg.Wait()
}

func (d *Daemon[T]) handleSuccess(ctx context.Context, email email.Email) {
	updErr := d.service.UpdateStatus(ctx, email.Id, "SENT")
	if updErr != nil {
		log.Print(updErr.Error())
		return
	}

	callErr := d.callbackExecutor.Execute(ctx, email.SuccessCallback)
	if callErr != nil {
		log.Print(callErr.Error())
		return
	}

	updErr = d.service.UpdateStatus(ctx, email.Id, "SENT-ACK")
	if updErr != nil {
		log.Print(updErr.Error())
	}
}

func (d *Daemon[T]) handleFailure(ctx context.Context, email email.Email) {
	updErr := d.service.UpdateStatus(ctx, email.Id, "FAILED")
	if updErr != nil {
		log.Print(updErr.Error())
		return
	}

	callErr := d.callbackExecutor.Execute(ctx, email.FailureCallback)
	if callErr != nil {
		log.Print(callErr.Error())
		return
	}

	updErr = d.service.UpdateStatus(ctx, email.Id, "FAILED-ACK")
	if updErr != nil {
		log.Print(updErr.Error())
	}
}
