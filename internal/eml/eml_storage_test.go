package eml

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEMLStorage_Store(t *testing.T) {
	t.Parallel()

	emlData := EML{
		MessageId:     "ff0fb587-e29b-4278-bbab-a525196b8917",
		From:          "sender@test.multidialogo.it",
		ReplyTo:       "no-reply@test.multidialogo.it",
		To:            "recipient2@test.multidialogo.it",
		Subject:       "Test Email with Reply-To",
		BodyHTML:      "<p>This is another test email in HTML format.</p>",
		BodyText:      "This is another test email in plain text format.",
		Date:          time.Unix(0, 0),
		Attachments:   []string{},
		CustomHeaders: map[string]string{"X-Custom-Header": "AnotherHeaderValue"},
	}

	baseStoragePath := "testdata/eml_storage_test/.out"
	sut := &EMLStorage{basePath: baseStoragePath}

	savedFilePath, err := sut.Store(emlData)
	require.NoError(t, err)

	year, month, _ := time.Now().Date()
	expectedFilePath := fmt.Sprintf("%v/%v/%v/%v.EML", baseStoragePath, year, month, emlData.MessageId)
	require.Equal(t, expectedFilePath, savedFilePath)
	assert.FileExists(t, expectedFilePath)
}
