package eml

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/h2non/filetype"
)

type EML struct {
	MessageId     string
	From          string
	ReplyTo       string
	To            string
	Subject       string
	BodyHTML      string
	BodyText      string
	Date          time.Time
	Attachments   []string
	CustomHeaders map[string]string
}

type Writer struct{}

func (w *Writer) addStandardHeadersToMessage(msg *mail.Message, data EML) {
	msg.Header = make(mail.Header)
	msg.Header["From"] = []string{data.From}

	if data.ReplyTo != data.From {
		msg.Header["Reply-To"] = []string{data.ReplyTo}
	}

	msg.Header["To"] = []string{data.To}
	msg.Header["Date"] = []string{data.Date.Format(time.RFC1123Z)}
	msg.Header["Subject"] = []string{data.Subject}
	msg.Header["Content-Type"] = []string{fmt.Sprintf("multipart/mixed; boundary=\"%s\"", data.MessageId)}

	for key, value := range data.CustomHeaders {
		msg.Header[key] = []string{value}
	}
}

func (w *Writer) writePart(multipartWriter *multipart.Writer, contentType, charset, body string) error {
	headers := textproto.MIMEHeader{
		"Content-Type":              []string{fmt.Sprintf("%s; %s", contentType, charset)},
		"Content-Transfer-Encoding": []string{"quoted-printable"},
	}

	part, err := multipartWriter.CreatePart(headers)
	if err != nil {
		return fmt.Errorf("failed to create part: %w", err)
	}

	writer := quotedprintable.NewWriter(part)
	defer writer.Close()

	if _, err = writer.Write([]byte(body)); err != nil {
		return fmt.Errorf("failed to write part body: %w", err)
	}

	// Ensure a blank line after the part content for proper MIME formatting
	if _, err = part.Write([]byte("\r\n")); err != nil {
		return fmt.Errorf("failed to write blank line after part: %w", err)
	}

	return nil
}

func (w *Writer) detectFileMimeFromKnownExtension(extension string) string {
	switch strings.ToLower(extension) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

func (w *Writer) detectFileMime(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	buffer := make([]byte, 261)
	_, err = file.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	kind, _ := filetype.Match(buffer)
	if kind == filetype.Unknown {
		return w.detectFileMimeFromKnownExtension(filepath.Ext(path)), nil
	}

	return kind.MIME.Value, nil
}

func (w *Writer) writeAttachment(target io.Writer, boundary string, path string, data []byte) error {
	mimeType, err := w.detectFileMime(path)
	if err != nil {
		return fmt.Errorf("failed to detect file mime type: %w", err)
	}

	// Write boundary
	if _, err := target.Write([]byte(fmt.Sprintf("--%s\r\n", boundary))); err != nil {
		return fmt.Errorf("failed to write boundary: %w", err)
	}

	// Write headers with proper MIME folding
	contentDisposition := fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(path))
	if err := w.writeFoldedHeader(target, "Content-Disposition", contentDisposition); err != nil {
		return fmt.Errorf("failed to write Content-Disposition header: %w", err)
	}

	if err := w.writeFoldedHeader(target, "Content-Type", mimeType); err != nil {
		return fmt.Errorf("failed to write Content-Type header: %w", err)
	}

	if err := w.writeFoldedHeader(target, "Content-Transfer-Encoding", "base64"); err != nil {
		return fmt.Errorf("failed to write Content-Transfer-Encoding header: %w", err)
	}

	// Write blank line after headers
	if _, err := target.Write([]byte("\r\n")); err != nil {
		return fmt.Errorf("failed to write newline after attachment headers: %w", err)
	}

	// Write base64 encoded data
	base64Encoder := base64.NewEncoder(base64.StdEncoding, target)
	defer base64Encoder.Close()

	if _, err = base64Encoder.Write(data); err != nil {
		return fmt.Errorf("failed to write attachment data: %w", err)
	}

	// Ensure a blank line after the attachment content for proper MIME formatting
	if _, err := target.Write([]byte("\r\n")); err != nil {
		return fmt.Errorf("failed to write blank line after attachment: %w", err)
	}

	return nil
}

func (w *Writer) isHeaderInList(slice []string, item string) bool {
	for _, element := range slice {
		if element == item {
			return true
		}
	}
	return false
}

// writeFoldedHeader writes a header line with proper MIME folding (max 76 characters per line)
func (w *Writer) writeFoldedHeader(target io.Writer, key, value string) error {
	headerLine := fmt.Sprintf("%s: %s\r\n", key, value)

	// If the header line is within the 76 character limit, write it directly
	if len(headerLine) <= 76 {
		_, err := target.Write([]byte(headerLine))
		return err
	}

	// Otherwise, fold the header: write the first part, then continue with space
	firstLine := headerLine[:76]
	if !strings.Contains(firstLine, "\r\n") {
		// Find the last space before position 76 to break nicely
		lastSpace := strings.LastIndex(headerLine[:76], " ")
		if lastSpace > len(key)+2 { // Make sure we don't break in the header name
			firstLine = headerLine[:lastSpace]
		}
	}

	_, err := target.Write([]byte(firstLine + "\r\n"))
	if err != nil {
		return err
	}

	// Write continuation lines with leading space
	remaining := headerLine[len(firstLine):]
	for len(remaining) > 0 {
		if len(remaining) <= 75 { // 75 because we add a space at the beginning
			_, err = target.Write([]byte(" " + remaining))
			return err
		}

		// Find break point for continuation
		breakPoint := 74 // Leave room for space + potential line ending
		if breakPoint > len(remaining) {
			breakPoint = len(remaining)
		}

		lastSpace := strings.LastIndex(remaining[:breakPoint], " ")
		if lastSpace > 0 {
			breakPoint = lastSpace
		}

		line := " " + remaining[:breakPoint] + "\r\n"
		_, err = target.Write([]byte(line))
		if err != nil {
			return err
		}
		remaining = remaining[breakPoint:]
	}

	return nil
}

func (w *Writer) Write(target io.Writer, data EML) error {
	msg := &mail.Message{}
	w.addStandardHeadersToMessage(msg, data)

	orderedStandardHeaders := []string{"From", "Reply-To", "To", "Date", "Subject", "Content-Type"}

	// Write standard headers with folding
	for _, key := range orderedStandardHeaders {
		if values, exists := msg.Header[key]; exists {
			for _, value := range values {
				if err := w.writeFoldedHeader(target, key, value); err != nil {
					return fmt.Errorf("failed to write header %s: %w", key, err)
				}
			}
		}
	}

	// Write custom headers with folding
	for key, values := range msg.Header {
		if w.isHeaderInList(orderedStandardHeaders, key) {
			continue
		}

		for _, value := range values {
			if err := w.writeFoldedHeader(target, key, value); err != nil {
				return fmt.Errorf("failed to write custom header %s: %w", key, err)
			}
		}
	}

	if _, err := target.Write([]byte("\r\n")); err != nil {
		return fmt.Errorf("failed to write newline after custom headers: %w", err)
	}

	multipartWriter := multipart.NewWriter(target)
	if err := multipartWriter.SetBoundary(data.MessageId); err != nil {
		return fmt.Errorf("failed to write multipart boundary: %w", err)
	}

	if data.BodyText != "" {
		if err := w.writePart(multipartWriter, "text/plain", "charset=utf-8", data.BodyText); err != nil {
			return err
		}
	}

	if data.BodyHTML != "" {
		if err := w.writePart(multipartWriter, "text/html", "charset=utf-8", data.BodyHTML); err != nil {
			return err
		}
	}

	for _, attachment := range data.Attachments {
		attachmentData, err := os.ReadFile(attachment)
		if err != nil {
			return fmt.Errorf("failed to read attachment: %w", err)
		}

		if err = w.writeAttachment(target, data.MessageId, attachment, attachmentData); err != nil {
			return err
		}
	}

	// Write final boundary
	if _, err := target.Write([]byte(fmt.Sprintf("\r\n--%s--\r\n", data.MessageId))); err != nil {
		return fmt.Errorf("failed to write final boundary: %w", err)
	}

	return nil
}
