//go:build unit

package email

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttachmentList_UnmarshalJSON_ArrayOfStrings(t *testing.T) {
	jsonData := []byte(`["file:///path/to/file1.pdf", "file:///path/to/file2.docx"]`)

	var attachments AttachmentList
	err := json.Unmarshal(jsonData, &attachments)

	require.NoError(t, err)
	assert.Len(t, attachments, 2)
	assert.Equal(t, "file:///path/to/file1.pdf", attachments[0].Path)
	assert.Equal(t, "file1.pdf", attachments[0].Name)
	assert.Equal(t, "file:///path/to/file2.docx", attachments[1].Path)
	assert.Equal(t, "file2.docx", attachments[1].Name)
}

func TestAttachmentList_UnmarshalJSON_ArrayOfObjects(t *testing.T) {
	jsonData := []byte(`[
		{"path": "file:///path/to/file1.pdf", "name": "Report Finale.pdf"},
		{"path": "file:///path/to/file2.docx", "name": "Contratto.docx"}
	]`)

	var attachments AttachmentList
	err := json.Unmarshal(jsonData, &attachments)

	require.NoError(t, err)
	assert.Len(t, attachments, 2)
	assert.Equal(t, "file:///path/to/file1.pdf", attachments[0].Path)
	assert.Equal(t, "Report Finale.pdf", attachments[0].Name)
	assert.Equal(t, "file:///path/to/file2.docx", attachments[1].Path)
	assert.Equal(t, "Contratto.docx", attachments[1].Name)
}

func TestAttachmentList_UnmarshalJSON_EmptyArray(t *testing.T) {
	jsonData := []byte(`[]`)

	var attachments AttachmentList
	err := json.Unmarshal(jsonData, &attachments)

	require.NoError(t, err)
	assert.Len(t, attachments, 0)
}

func TestAttachmentList_UnmarshalJSON_InvalidFormat(t *testing.T) {
	jsonData := []byte(`"not an array"`)

	var attachments AttachmentList
	err := json.Unmarshal(jsonData, &attachments)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "attachments must be either array of strings or array of objects")
}

func TestLoadPayload_WithAttachmentsAsStrings(t *testing.T) {
	jsonContent := `{
		"id": "550e8400-e29b-41d4-a716-446655440000",
		"from": "sender@example.com",
		"reply_to": "reply@example.com",
		"to": "recipient@example.com",
		"subject": "Test Subject",
		"body_text": "Test body",
		"attachments": ["file:///path/to/file1.pdf", "file:///path/to/file2.docx"]
	}`

	tmpFile, err := os.CreateTemp("", "payload-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(jsonContent)
	require.NoError(t, err)
	tmpFile.Close()

	payload, err := LoadPayload(tmpFile.Name())

	require.NoError(t, err)
	assert.Len(t, payload.Attachments, 2)
	assert.Equal(t, "file:///path/to/file1.pdf", payload.Attachments[0].Path)
	assert.Equal(t, "file1.pdf", payload.Attachments[0].Name)
	assert.Equal(t, "file:///path/to/file2.docx", payload.Attachments[1].Path)
	assert.Equal(t, "file2.docx", payload.Attachments[1].Name)
}

func TestLoadPayload_WithAttachmentsAsObjects(t *testing.T) {
	jsonContent := `{
		"id": "550e8400-e29b-41d4-a716-446655440000",
		"from": "sender@example.com",
		"reply_to": "reply@example.com",
		"to": "recipient@example.com",
		"subject": "Test Subject",
		"body_text": "Test body",
		"attachments": [
			{"path": "file:///path/to/file1.pdf", "name": "Report.pdf"},
			{"path": "file:///path/to/file2.docx", "name": "Contract.docx"}
		]
	}`

	tmpFile, err := os.CreateTemp("", "payload-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(jsonContent)
	require.NoError(t, err)
	tmpFile.Close()

	payload, err := LoadPayload(tmpFile.Name())

	require.NoError(t, err)
	assert.Len(t, payload.Attachments, 2)
	assert.Equal(t, "file:///path/to/file1.pdf", payload.Attachments[0].Path)
	assert.Equal(t, "Report.pdf", payload.Attachments[0].Name)
	assert.Equal(t, "file:///path/to/file2.docx", payload.Attachments[1].Path)
	assert.Equal(t, "Contract.docx", payload.Attachments[1].Name)
}

func TestLoadPayload_WithoutAttachments(t *testing.T) {
	jsonContent := `{
		"id": "550e8400-e29b-41d4-a716-446655440000",
		"from": "sender@example.com",
		"reply_to": "reply@example.com",
		"to": "recipient@example.com",
		"subject": "Test Subject",
		"body_text": "Test body"
	}`

	tmpFile, err := os.CreateTemp("", "payload-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(jsonContent)
	require.NoError(t, err)
	tmpFile.Close()

	payload, err := LoadPayload(tmpFile.Name())

	require.NoError(t, err)
	assert.Len(t, payload.Attachments, 0)
}

func TestAttachmentList_UnmarshalJSON_WithWindowsPath(t *testing.T) {
	jsonData := []byte(`["file:///C:/Users/test/document.pdf"]`)

	var attachments AttachmentList
	err := json.Unmarshal(jsonData, &attachments)

	require.NoError(t, err)
	assert.Len(t, attachments, 1)
	assert.Equal(t, "file:///C:/Users/test/document.pdf", attachments[0].Path)
	assert.Equal(t, "document.pdf", attachments[0].Name)
}

func TestAttachmentList_UnmarshalJSON_MixedFormatsNotAllowed(t *testing.T) {
	jsonData := []byte(`["file:///path/to/file1.pdf", {"path": "file:///path/to/file2.pdf", "name": "Custom.pdf"}]`)

	var attachments AttachmentList
	err := json.Unmarshal(jsonData, &attachments)

	require.Error(t, err)
}
