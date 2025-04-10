//go:build integration

package smtp

import (
	"github.com/stretchr/testify/require"
	"os"
	"strconv"
	"sync"
	"testing"
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

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := client.Send("testdata/smol.EML")
			require.NoError(t, err)
		}()
	}

	wg.Wait()
}
