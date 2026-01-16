//go:build integration

package smtp

import (
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"mailculator-processor/internal/email"
)

var client *Client

func init() {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	cfg := Config{
		User:             os.Getenv("SMTP_USER"),
		Password:         os.Getenv("SMTP_PASS"),
		Host:             os.Getenv("SMTP_HOST"),
		Port:             port,
		From:             os.Getenv("SMTP_FROM"),
		AllowInsecureTls: true,
	}

	client = New(cfg)
}

func TestClientSendIntegration(t *testing.T) {
	var wg sync.WaitGroup
	payload := email.Payload{
		Id:       "550e8400-e29b-41d4-a716-446655440000",
		From:     "sender@example.com",
		ReplyTo:  "reply@example.com",
		To:       "recipient@example.com",
		Subject:  "Integration test email",
		BodyText: "Hello from the integration test",
	}

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := client.Send(payload, "")
			require.NoError(t, err)
		}()
	}

	wg.Wait()
}
