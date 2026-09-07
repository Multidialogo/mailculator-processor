//go:build unit

package smtp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mailculator-processor/internal/email"
)

func createTempAttachment(t *testing.T, dir string, name string, size int) string {
	t.Helper()
	path := filepath.Join(dir, name)
	data := make([]byte, size)
	require.NoError(t, os.WriteFile(path, data, 0644))
	return name
}

func TestCheckAttachmentsSize_UnderLimit(t *testing.T) {
	dir := t.TempDir()
	fileName := createTempAttachment(t, dir, "small.pdf", 1024)

	client := New(Config{}, 30408704)
	attachments := email.AttachmentList{{Path: fileName, Name: "small.pdf"}}

	err := client.checkAttachmentsSize(attachments, dir+"/")
	assert.NoError(t, err)
}

func TestCheckAttachmentsSize_OverLimit(t *testing.T) {
	dir := t.TempDir()
	createTempAttachment(t, dir, "big1.bin", 20_000_000)
	createTempAttachment(t, dir, "big2.bin", 15_000_000)

	client := New(Config{}, 30408704)
	attachments := email.AttachmentList{
		{Path: "big1.bin", Name: "big1.bin"},
		{Path: "big2.bin", Name: "big2.bin"},
	}

	err := client.checkAttachmentsSize(attachments, dir+"/")
	require.Error(t, err)

	var sizeErr *AttachmentsSizeExceededError
	require.ErrorAs(t, err, &sizeErr)
	assert.Equal(t, int64(35_000_000), sizeErr.TotalSize)
	assert.Equal(t, 30408704, sizeErr.MaxSize)
	assert.Contains(t, sizeErr.Error(), "35000000 bytes")
	assert.Contains(t, sizeErr.Error(), "30408704 bytes")
	assert.Equal(t, "La dimensione totale degli allegati supera il limite massimo consentito", sizeErr.Reason())
}

func TestCheckAttachmentsSize_ExactlyAtLimit(t *testing.T) {
	dir := t.TempDir()
	createTempAttachment(t, dir, "exact.bin", 30408704)

	client := New(Config{}, 30408704)
	attachments := email.AttachmentList{{Path: "exact.bin", Name: "exact.bin"}}

	err := client.checkAttachmentsSize(attachments, dir+"/")
	assert.NoError(t, err)
}

func TestCheckAttachmentsSize_SkippedWhenLimitIsZero(t *testing.T) {
	client := New(Config{}, 0)
	attachments := email.AttachmentList{{Path: "nonexistent.pdf", Name: "test.pdf"}}

	err := client.checkAttachmentsSize(attachments, "/fake/")
	assert.NoError(t, err)
}

func TestCheckAttachmentsSize_SkippedWhenNoAttachments(t *testing.T) {
	client := New(Config{}, 30408704)

	err := client.checkAttachmentsSize(email.AttachmentList{}, "/fake/")
	assert.NoError(t, err)
}

func TestCheckAttachmentsSize_FileStatError(t *testing.T) {
	client := New(Config{}, 30408704)
	attachments := email.AttachmentList{{Path: "nonexistent.pdf", Name: "test.pdf"}}

	err := client.checkAttachmentsSize(attachments, "/fake/path/")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "impossibile verificare la dimensione dell'allegato")
}
