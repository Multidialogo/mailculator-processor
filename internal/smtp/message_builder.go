package smtp

import (
	"bytes"
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

	"mailculator-processor/internal/email"
)

type MessageBuilder struct{}

func (b *MessageBuilder) Build(payload email.Payload, attachmentsBasePath string) ([]byte, error) {
	msg := &mail.Message{}
	b.addStandardHeadersToMessage(msg, payload)

	orderedStandardHeaders := []string{"From", "Reply-To", "To", "Date", "Subject", "Content-Type"}
	var buf bytes.Buffer

	for _, key := range orderedStandardHeaders {
		if values, exists := msg.Header[key]; exists {
			for _, value := range values {
				if err := b.writeFoldedHeader(&buf, key, value); err != nil {
					return nil, fmt.Errorf("failed to write header %s: %w", key, err)
				}
			}
		}
	}

	for key, values := range msg.Header {
		if b.isHeaderInList(orderedStandardHeaders, key) {
			continue
		}

		for _, value := range values {
			if err := b.writeFoldedHeader(&buf, key, value); err != nil {
				return nil, fmt.Errorf("failed to write custom header %s: %w", key, err)
			}
		}
	}

	if _, err := buf.Write([]byte("\r\n")); err != nil {
		return nil, fmt.Errorf("failed to write newline after custom headers: %w", err)
	}

	multipartWriter := multipart.NewWriter(&buf)
	if err := multipartWriter.SetBoundary(payload.Id); err != nil {
		return nil, fmt.Errorf("failed to write multipart boundary: %w", err)
	}

	if payload.BodyText != "" {
		if err := b.writePart(multipartWriter, "text/plain", "charset=utf-8", payload.BodyText); err != nil {
			return nil, err
		}
	}

	if payload.BodyHTML != "" {
		if err := b.writePart(multipartWriter, "text/html", "charset=utf-8", payload.BodyHTML); err != nil {
			return nil, err
		}
	}

	for _, attachment := range b.resolveAttachments(payload.Attachments, attachmentsBasePath) {
		attachmentData, err := os.ReadFile(attachment)
		if err != nil {
			return nil, fmt.Errorf("failed to read attachment: %w", err)
		}

		if err = b.writeAttachment(&buf, payload.Id, attachment, attachmentData); err != nil {
			return nil, err
		}
	}

	if _, err := buf.Write([]byte(fmt.Sprintf("\r\n--%s--\r\n", payload.Id))); err != nil {
		return nil, fmt.Errorf("failed to write final boundary: %w", err)
	}

	return buf.Bytes(), nil
}

func (b *MessageBuilder) resolveAttachments(attachments []string, basePath string) []string {
	if len(attachments) == 0 {
		return nil
	}

	attachmentsWithBasePath := make([]string, len(attachments))
	for i, attachment := range attachments {
		attachmentsWithBasePath[i] = basePath + attachment
	}

	return attachmentsWithBasePath
}

func (b *MessageBuilder) addStandardHeadersToMessage(msg *mail.Message, data email.Payload) {
	msg.Header = make(mail.Header)
	msg.Header["From"] = []string{data.From}

	if data.ReplyTo != data.From {
		msg.Header["Reply-To"] = []string{data.ReplyTo}
	}

	msg.Header["To"] = []string{data.To}
	msg.Header["Date"] = []string{time.Now().Format(time.RFC1123Z)}
	msg.Header["Subject"] = []string{data.Subject}
	msg.Header["Content-Type"] = []string{fmt.Sprintf("multipart/mixed; boundary=\"%s\"", data.Id)}

	for key, value := range data.CustomHeaders {
		msg.Header[key] = []string{value}
	}
}

func (b *MessageBuilder) writePart(multipartWriter *multipart.Writer, contentType, charset, body string) error {
	headers := textproto.MIMEHeader{
		"Content-Type":              []string{fmt.Sprintf("%s; %s", contentType, charset)},
		"Content-Transfer-Encoding": []string{"quoted-printable"},
	}

	part, err := multipartWriter.CreatePart(headers)
	if err != nil {
		return fmt.Errorf("failed to create part: %w", err)
	}

	writer := quotedprintable.NewWriter(part)
	if _, err = writer.Write([]byte(body)); err != nil {
		return fmt.Errorf("failed to write part body: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close quoted-printable writer: %w", err)
	}

	if _, err = part.Write([]byte("\r\n")); err != nil {
		return fmt.Errorf("failed to write blank line after part: %w", err)
	}

	return nil
}

func (b *MessageBuilder) detectFileMimeFromKnownExtension(extension string) string {
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

func (b *MessageBuilder) detectFileMime(path string) (string, error) {
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
		return b.detectFileMimeFromKnownExtension(filepath.Ext(path)), nil
	}

	return kind.MIME.Value, nil
}

type lineBreakWriter struct {
	w           io.Writer
	lineLength  int
	currentLine int
}

func newLineBreakWriter(w io.Writer, lineLength int) *lineBreakWriter {
	return &lineBreakWriter{
		w:          w,
		lineLength: lineLength,
	}
}

func (lbw *lineBreakWriter) Write(p []byte) (n int, err error) {
	for len(p) > 0 {
		if lbw.currentLine >= lbw.lineLength {
			if _, err := lbw.w.Write([]byte("\r\n")); err != nil {
				return n, err
			}
			lbw.currentLine = 0
		}

		remaining := lbw.lineLength - lbw.currentLine
		toWrite := remaining
		if toWrite > len(p) {
			toWrite = len(p)
		}

		written, err := lbw.w.Write(p[:toWrite])
		n += written
		lbw.currentLine += written
		p = p[toWrite:]

		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (b *MessageBuilder) writeAttachment(target io.Writer, boundary string, path string, data []byte) error {
	mimeType, err := b.detectFileMime(path)
	if err != nil {
		return fmt.Errorf("failed to detect file mime type: %w", err)
	}

	if _, err := target.Write([]byte(fmt.Sprintf("--%s\r\n", boundary))); err != nil {
		return fmt.Errorf("failed to write boundary: %w", err)
	}

	contentDisposition := fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(path))
	if err := b.writeFoldedHeader(target, "Content-Disposition", contentDisposition); err != nil {
		return fmt.Errorf("failed to write Content-Disposition header: %w", err)
	}

	if err := b.writeFoldedHeader(target, "Content-Type", mimeType); err != nil {
		return fmt.Errorf("failed to write Content-Type header: %w", err)
	}

	if err := b.writeFoldedHeader(target, "Content-Transfer-Encoding", "base64"); err != nil {
		return fmt.Errorf("failed to write Content-Transfer-Encoding header: %w", err)
	}

	if _, err := target.Write([]byte("\r\n")); err != nil {
		return fmt.Errorf("failed to write newline after attachment headers: %w", err)
	}

	lineBreaker := newLineBreakWriter(target, 76)
	base64Encoder := base64.NewEncoder(base64.StdEncoding, lineBreaker)

	if _, err = base64Encoder.Write(data); err != nil {
		return fmt.Errorf("failed to write attachment data: %w", err)
	}

	if err := base64Encoder.Close(); err != nil {
		return fmt.Errorf("failed to close base64 encoder: %w", err)
	}

	if _, err := target.Write([]byte("\r\n")); err != nil {
		return fmt.Errorf("failed to write blank line after attachment: %w", err)
	}

	return nil
}

func (b *MessageBuilder) isHeaderInList(slice []string, item string) bool {
	for _, element := range slice {
		if element == item {
			return true
		}
	}
	return false
}

func (b *MessageBuilder) canFoldHeader(key string) bool {
	emailHeaders := []string{
		"From", "To", "Cc", "Bcc", "Reply-To", "Sender",
		"Resent-From", "Resent-To", "Resent-Cc", "Resent-Bcc", "Resent-Sender",
	}

	for _, emailHeader := range emailHeaders {
		if strings.EqualFold(key, emailHeader) {
			return false
		}
	}

	mimeHeaders := []string{
		"Content-Type", "Content-Disposition", "Content-Transfer-Encoding",
		"Content-ID", "Content-Description",
	}

	for _, mimeHeader := range mimeHeaders {
		if strings.EqualFold(key, mimeHeader) {
			return false
		}
	}

	return true
}

func (b *MessageBuilder) writeFoldedHeader(target io.Writer, key, value string) error {
	headerLine := fmt.Sprintf("%s: %s\r\n", key, value)

	if !b.canFoldHeader(key) {
		if len(headerLine) > 998 {
			maxValueLen := 998 - len(key) - 2
			if maxValueLen > 0 {
				value = value[:maxValueLen]
				headerLine = fmt.Sprintf("%s: %s\r\n", key, value)
			}
		}
		_, err := target.Write([]byte(headerLine))
		return err
	}

	if len(headerLine) > 998 {
		maxValueLen := 998 - len(key) - 2
		if maxValueLen > 0 {
			value = value[:maxValueLen]
			headerLine = fmt.Sprintf("%s: %s\r\n", key, value)
		}
	}

	if len(headerLine) <= 76 {
		_, err := target.Write([]byte(headerLine))
		return err
	}

	firstLine := headerLine[:76]
	if !strings.Contains(firstLine, "\r\n") {
		lastSpace := strings.LastIndex(headerLine[:76], " ")
		if lastSpace > len(key)+2 {
			firstLine = headerLine[:lastSpace]
		}
	}

	_, err := target.Write([]byte(firstLine + "\r\n"))
	if err != nil {
		return err
	}

	remaining := headerLine[len(firstLine):]
	for len(remaining) > 0 {
		if len(remaining) <= 75 {
			_, err = target.Write([]byte(" " + remaining))
			return err
		}

		breakPoint := 74
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
