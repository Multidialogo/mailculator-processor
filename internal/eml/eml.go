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

func (w *Writer) writeAttachment(multipartWriter *multipart.Writer, path string, data []byte) error {
	mimeType, err := w.detectFileMime(path)
	if err != nil {
		return fmt.Errorf("failed to detect file mime type: %w", err)
	}

	headers := textproto.MIMEHeader{
		"Content-Type":              []string{mimeType},
		"Content-Disposition":       []string{fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(path))},
		"Content-Transfer-Encoding": []string{"base64"},
	}

	part, err := multipartWriter.CreatePart(headers)
	if err != nil {
		return fmt.Errorf("failed to create attachment part: %w", err)
	}

	base64Encoder := base64.NewEncoder(base64.StdEncoding, part)
	defer base64Encoder.Close()

	if _, err = base64Encoder.Write(data); err != nil {
		return fmt.Errorf("failed to write attachment data: %w", err)
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

func (w *Writer) Write(target io.Writer, data EML) error {
	msg := &mail.Message{}
	w.addStandardHeadersToMessage(msg, data)

	orderedStandardHeaders := []string{"From", "Reply-To", "To", "Date", "Subject", "Content-Type"}

	for _, key := range orderedStandardHeaders {
		if values, exists := msg.Header[key]; exists {
			for _, value := range values {
				if _, err := target.Write([]byte(fmt.Sprintf("%s: %s\r\n", key, value))); err != nil {
					return fmt.Errorf("failed to write header %s: %w", key, err)
				}
			}
		}
	}

	for key, values := range msg.Header {
		if w.isHeaderInList(orderedStandardHeaders, key) {
			continue
		}

		for _, value := range values {
			if _, err := target.Write([]byte(fmt.Sprintf("%s: %s\r\n", key, value))); err != nil {
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

		if err = w.writeAttachment(multipartWriter, attachment, attachmentData); err != nil {
			return err
		}
	}

	if err := multipartWriter.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return nil
}
