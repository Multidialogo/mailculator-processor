package smtp

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"mailculator-processor/internal/email"
)

func TestBuildConvertsDataImagesToCIDInlineParts(t *testing.T) {
	// 1x1 red PNG
	pngBytes, err := base64.StdEncoding.DecodeString(
		"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==",
	)
	require.NoError(t, err)
	pngB64 := base64.StdEncoding.EncodeToString(pngBytes)

	emailID := "550e8400-e29b-41d4-a716-446655440000"
	html := `<html><body><p>Hello</p><img src="data:image/png;base64,` + pngB64 + `"></body></html>`

	builder := &MessageBuilder{}
	msg, err := builder.Build(email.Payload{
		Id:       emailID,
		From:     "sender@example.com",
		ReplyTo:  "sender@example.com",
		To:       "recipient@example.com",
		Subject:  "CID test",
		BodyHTML: html,
		BodyText: "Hello",
	}, "")
	require.NoError(t, err)

	raw := string(msg)

	require.Contains(t, raw, "Content-Type: multipart/mixed;")
	require.Contains(t, raw, "Content-Type: multipart/related;")
	require.True(t, strings.HasPrefix(raw, "MIME-Version: 1.0\r\n"))
	require.Contains(t, raw, "boundary=\"mixed-"+emailID+"\"")
	require.Contains(t, raw, "boundary=\"related-"+emailID+"\"")

	expectedCID := "inline-1-" + emailID + "@mailculator.local"
	require.Contains(t, raw, "Content-ID: <"+expectedCID+">")
	require.Contains(t, raw, "cid:inline-1-")
	require.Contains(t, raw, "@mailculator.local")
	require.NotContains(t, raw, "data:image/png;base64,")
	require.Contains(t, raw, "Content-Disposition: inline; filename=\"inline-1.png\"")
	require.Contains(t, raw, "Content-Type: image/png")
	require.Contains(t, raw, "Content-Transfer-Encoding: base64")
}

func TestBuildLeavesHTMLWithoutDataImagesUnchanged(t *testing.T) {
	html := `<html><body><p>No images</p><img src="https://example.com/logo.png"></body></html>`

	builder := &MessageBuilder{}
	msg, err := builder.Build(email.Payload{
		Id:       "550e8400-e29b-41d4-a716-446655440000",
		From:     "sender@example.com",
		ReplyTo:  "sender@example.com",
		To:       "recipient@example.com",
		Subject:  "No CID",
		BodyHTML: html,
	}, "")
	require.NoError(t, err)

	raw := string(msg)
	require.NotContains(t, raw, "multipart/related")
	require.NotContains(t, raw, "Content-ID:")
	require.Contains(t, raw, "https://example.com/logo.png")
	require.Contains(t, raw, "Content-Type: text/html;")
}

func TestBuildKeepsFileAttachmentsAsAttachmentDisposition(t *testing.T) {
	dir := t.TempDir()
	attachmentPath := filepath.Join(dir, "doc.txt")
	require.NoError(t, os.WriteFile(attachmentPath, []byte("attachment-body"), 0o644))

	pngBytes, err := base64.StdEncoding.DecodeString(
		"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==",
	)
	require.NoError(t, err)
	pngB64 := base64.StdEncoding.EncodeToString(pngBytes)

	emailID := "550e8400-e29b-41d4-a716-446655440000"
	html := `<html><body><img src="data:image/png;base64,` + pngB64 + `"></body></html>`

	builder := &MessageBuilder{}
	msg, err := builder.Build(email.Payload{
		Id:       emailID,
		From:     "sender@example.com",
		ReplyTo:  "sender@example.com",
		To:       "recipient@example.com",
		Subject:  "Mixed",
		BodyHTML: html,
		Attachments: email.AttachmentList{
			{Path: attachmentPath, Name: "doc.txt"},
		},
	}, "")
	require.NoError(t, err)

	raw := string(msg)
	require.Contains(t, raw, "Content-Disposition: inline; filename=\"inline-1.png\"")
	require.Contains(t, raw, "Content-Disposition: attachment; filename=\"doc.txt\"")
	require.True(t, strings.Count(raw, "Content-ID:") == 1)
}

func TestHeaderParam(t *testing.T) {
	require.Equal(t, `filename="doc.pdf"`, headerParam("filename", "doc.pdf"))
	require.Equal(t, `filename="my file.pdf"`, headerParam("filename", "my file.pdf"))
	require.Equal(t, `filename="a\"b.pdf"`, headerParam("filename", `a"b.pdf`))
	require.Equal(t, `filename="a\\b.pdf"`, headerParam("filename", `a\b.pdf`))

	encoded := headerParam("filename", "contratto_è_più.pdf")
	require.True(t, strings.HasPrefix(encoded, "filename*=utf-8''"), encoded)
	require.Contains(t, encoded, "%C3%A8")
	require.NotContains(t, encoded, "è")

	injected := headerParam("filename","a.pdf\r\nBcc: sender@example.com")
	require.NotContains(t, injected, "\r")
	require.NotContains(t, injected, "\n")
}

func TestBuildEncodesNonASCIIAttachmentName(t *testing.T) {
	dir := t.TempDir()
	attachmentPath := filepath.Join(dir, "doc.txt")
	require.NoError(t, os.WriteFile(attachmentPath, []byte("attachment-body"), 0o644))

	builder := &MessageBuilder{}
	msg, err := builder.Build(email.Payload{
		Id:       "550e8400-e29b-41d4-a716-446655440000",
		From:     "sender@example.com",
		ReplyTo:  "sender@example.com",
		To:       "recipient@example.com",
		Subject:  "Non ASCII",
		BodyText: "body",
		Attachments: email.AttachmentList{
			{Path: attachmentPath, Name: "contratto_è_più.pdf"},
		},
	}, "")
	require.NoError(t, err)

	raw := string(msg)
	require.Contains(t, raw, "Content-Disposition: attachment; filename*=utf-8''contratto_%C3%A8_pi%C3%B9.pdf")
	require.NotContains(t, raw, "è")
}

func TestExtractInlineImagesRejectsInvalidBase64(t *testing.T) {
	builder := &MessageBuilder{}
	_, _, err := builder.extractInlineImages(
		`<img src="data:image/png;base64,====">`,
		"550e8400-e29b-41d4-a716-446655440000",
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to decode inline image")
}

func TestExtractInlineImagesHandlesMultipleImages(t *testing.T) {
	jpegB64 := base64.StdEncoding.EncodeToString([]byte("fake-jpeg"))
	gifB64 := base64.StdEncoding.EncodeToString([]byte("fake-gif"))
	emailID := "550e8400-e29b-41d4-a716-446655440000"

	html := `<img src="data:image/jpeg;base64,` + jpegB64 + `"><img src='data:image/gif;base64,` + gifB64 + `'>`

	builder := &MessageBuilder{}
	rewritten, images, err := builder.extractInlineImages(html, emailID)
	require.NoError(t, err)
	require.Len(t, images, 2)
	require.Equal(t, "image/jpeg", images[0].MimeType)
	require.Equal(t, "image/gif", images[1].MimeType)
	require.Equal(t, "jpg", extensionForImageSubtype("jpeg"))
	require.Contains(t, rewritten, "cid:inline-1-"+emailID+"@mailculator.local")
	require.Contains(t, rewritten, "cid:inline-2-"+emailID+"@mailculator.local")
	require.NotContains(t, rewritten, "data:image")
}
