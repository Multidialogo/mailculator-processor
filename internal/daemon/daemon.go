package daemon

import (
	"context"
	"log"
	"mailculator-processor/internal/email"
	"sync"
)

type EmailFinder interface {
	FindAndLock() ([]email.Email, error)
}

type EmailSender interface {
	SendAndUnlock(context.Context, email.Email) error
}

type Daemon struct {
	finder    EmailFinder
	processor EmailSender
}

func NewDaemon(finder EmailFinder, processor EmailSender) *Daemon {
	return &Daemon{
		finder:    finder,
		processor: processor,
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
	emails, err := daemon.finder.FindAndLock()
	if err != nil {
		log.Print(err.Error())
		return
	}

	var wg sync.WaitGroup

	for _, emailToProcess := range emails {
		wg.Add(1)

		go func(wg *sync.WaitGroup, email email.Email) {
			defer wg.Done()

			err := daemon.processor.SendAndUnlock(ctx, email)
			if err != nil {
				log.Print(err.Error())
			}
		}(&wg, emailToProcess)
	}

	wg.Wait()
}
