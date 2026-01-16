//go:build integration

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mailculator-processor/internal/outbox"
	"mailculator-processor/internal/testutils/facades"
)

func TestMainComplete(t *testing.T) {
	payloadDir := t.TempDir()
	emlDir := filepath.Join(t.TempDir(), "eml")
	require.NoError(t, os.MkdirAll(emlDir, 0o755))

	t.Setenv("ATTACHMENTS_BASE_PATH", payloadDir+"/")
	t.Setenv("EML_STORAGE_PATH", emlDir)
	t.Setenv("PIPELINE_CALLBACK_URL", "http://127.0.0.1:8081/status-updates")
	t.Setenv("PIPELINE_INTERVAL", "1")
	t.Setenv("SMTP_HOST", "127.0.0.1")
	t.Setenv("SMTP_USER", "user")
	t.Setenv("SMTP_PASS", "pass")
	t.Setenv("SMTP_PORT", "1025")
	t.Setenv("SMTP_FROM", "mailer@example.com")
	t.Setenv("SMTP_ALLOW_INSECURE_TLS", "true")
	t.Setenv("MYSQL_HOST", "127.0.0.1")
	t.Setenv("MYSQL_PORT", "3306")
	t.Setenv("MYSQL_USER", "root")
	t.Setenv("MYSQL_PASSWORD", "test")
	t.Setenv("MYSQL_DATABASE", "mailculator_test")

	oFacade, err := facades.NewMySQLOutboxFacade()
	require.NoError(t, err)
	defer oFacade.Close()

	fixtures := make([]string, 0)

	for i := 0; i < 5; i++ {
		payloadPath, payloadErr := createPayloadFile(payloadDir)
		require.NoError(t, payloadErr)

		emailId, err := oFacade.AddEmailWithPayload(context.TODO(), outbox.StatusAccepted, payloadPath)
		require.NoError(t, err)
		fixtures = append(fixtures, emailId)
	}

	srv := &http.Server{
		Addr: ":8081",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("Method error: expected POST received %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}),
	}
	go srv.ListenAndServe()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	run(ctx)
	srv.Shutdown(ctx)

	for _, value := range fixtures {
		status, err := oFacade.GetEmailStatus(context.TODO(), value)
		require.NoError(t, err)
		assert.Equal(t, outbox.StatusSentAcknowledged, status)
	}

	// Delete fixtures.
	for _, value := range fixtures {
		errFix := oFacade.DeleteEmail(context.Background(), value)
		if errFix != nil {
			t.Errorf("error while deleting fixture %s, error: %v", value, errFix)
		}
	}
}

func createPayloadFile(dir string) (string, error) {
	payload := map[string]any{
		"id":          uuid.NewString(),
		"from":        "sender@example.com",
		"reply_to":    "reply@example.com",
		"to":          "recipient@example.com",
		"subject":     "Integration test email",
		"body_text":   "Hello from the integration test",
		"attachments": []string{},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, uuid.NewString()+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}

	return path, nil
}
