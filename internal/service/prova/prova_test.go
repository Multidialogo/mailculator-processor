package prova

import (
	"testing"
	"time"
	"fmt"
	"os"
	"syscall"
)

func TestSuperProva(t *testing.T) {

	emailManager := NewEmailManager()
	emailManager.ProcessEmails = func() {
		fmt.Println("Testing SIGTERM signal")
	}
	go emailManager.Start()

	sleep := 100 * time.Millisecond

	time.Sleep(sleep)

	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		return
	}
	err = p.Signal(syscall.SIGTERM)

	time.Sleep(sleep)

	if emailManager.IsRunning() {
		emailManager.Stop()
		t.Errorf("Testing SIGTERM signal fail: Start method is still running")
	}
}
