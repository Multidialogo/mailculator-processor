package prova

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

type EmailManager struct {
	stopCh        chan struct{}
	running       bool
	ProcessEmails func()
}

func NewEmailManager() *EmailManager {
	return &EmailManager{
		stopCh:  make(chan struct{}),
		running: false,
		ProcessEmails: func() {
			fmt.Println("Process emails")
		},
	}
}

func (em *EmailManager) Start() error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	em.running = true

	for {
		select {
		case <-sigCh:
			em.running = false
			return nil

		case <-em.stopCh:
			em.running = false
			return nil

		default:
			em.ProcessEmails()
		}
	}
}

func (em *EmailManager) Stop() {
	close(em.stopCh)
}

func (em *EmailManager) IsRunning() bool {
	return em.running
}
