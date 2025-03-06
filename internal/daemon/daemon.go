package daemon

import (
	"context"
	"log"
	"mailculator-processor/internal/email"
	"sync"
)

type emailService interface {
	LockAndReturnReadyToProcess(ctx context.Context) ([]email.Email, error)
	SendAndUnlock(context.Context, email.Email) error
}

type Daemon struct {
	emailService emailService
}

func NewDaemon(service emailService) *Daemon {
	return &Daemon{
		emailService: service,
	}
}

func (daemon Daemon) RunUntilContextDone(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			daemon.run(ctx)
		}
	}
}

func (daemon Daemon) run(ctx context.Context) {
	emails, err := daemon.emailService.LockAndReturnReadyToProcess(ctx)
	if err != nil {
		log.Print(err.Error())
		return
	}

	var wg sync.WaitGroup

	for _, emailToProcess := range emails {
		wg.Add(1)

		go func(wg *sync.WaitGroup, email email.Email) {
			defer wg.Done()

			err := daemon.emailService.SendAndUnlock(ctx, email)
			if err != nil {
				log.Print(err.Error())
			}
		}(&wg, emailToProcess)
	}

	wg.Wait()
}
