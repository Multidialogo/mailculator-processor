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

type emailClient interface {
	Send(email.Email) (bool, error)
	Close()
}

type emailClientFactory[T emailClient] interface {
	New() (T, error)
}

type shellCommand interface {
	Execute() error
}

type shellCommandFactory[T shellCommand] interface {
	New(string) T
}

type Daemon[E emailClient, S shellCommand] struct {
	service             emailService
	clientFactory       emailClientFactory[E]
	shellCommandFactory shellCommandFactory[S]
}

func NewDaemon[E emailClient, S shellCommand](es emailService, ecf emailClientFactory[E], scf shellCommandFactory[S]) *Daemon[E, S] {
	return &Daemon[E, S]{
		service:             es,
		clientFactory:       ecf,
		shellCommandFactory: scf,
	}
}

func (d *Daemon[E, S]) RunUntilContextDone(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			d.runSingleIteration(ctx)
		}
	}
}

func (d *Daemon[E, S]) runSingleIteration(ctx context.Context) {
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

func (d *Daemon[E, S]) handleSuccess(ctx context.Context, email email.Email) {
	if updErr := d.service.UpdateStatus(ctx, email.Id, "SENT"); updErr != nil {
		log.Print(updErr.Error())
		return
	}

	cmd := d.shellCommandFactory.New(email.SuccessCallback)
	if cmdErr := cmd.Execute(); cmdErr != nil {
		log.Print(cmdErr.Error())
		return
	}

	if updErr := d.service.UpdateStatus(ctx, email.Id, "SENT-ACK"); updErr != nil {
		log.Print(updErr.Error())
	}
}

func (d *Daemon[E, S]) handleFailure(ctx context.Context, email email.Email) {
	if updErr := d.service.UpdateStatus(ctx, email.Id, "FAILED"); updErr != nil {
		log.Print(updErr.Error())
		return
	}

	cmd := d.shellCommandFactory.New(email.FailureCallback)
	if cmdErr := cmd.Execute(); cmdErr != nil {
		log.Print(cmdErr.Error())
		return
	}

	if updErr := d.service.UpdateStatus(ctx, email.Id, "FAILED-ACK"); updErr != nil {
		log.Print(updErr.Error())
	}
}
