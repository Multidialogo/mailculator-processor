package smtp

import (
	"crypto/tls"
	"fmt"
	"net/mail"
	"net/smtp"
	"os"

	"mailculator-processor/internal/email"
)

type Config struct {
	User             string
	Password         string
	Host             string
	Port             int
	From             string
	AllowInsecureTls bool
}

type Client struct {
	cfg                Config
	builder            *MessageBuilder
	maxAttachmentsSize int
}

func New(cfg Config, maxAttachmentsSize int) *Client {
	return &Client{
		cfg:                cfg,
		builder:            &MessageBuilder{},
		maxAttachmentsSize: maxAttachmentsSize,
	}
}

func (c *Client) Send(payload email.Payload, attachmentsBasePath string) error {
	if err := c.checkAttachmentsSize(payload.Attachments, attachmentsBasePath); err != nil {
		return err
	}

	message, err := c.builder.Build(payload, attachmentsBasePath)
	if err != nil {
		return err
	}

	tlsCfg := &tls.Config{
		ServerName:         c.cfg.Host,
		InsecureSkipVerify: c.cfg.AllowInsecureTls,
	}

	server := fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port)
	client, err := smtp.Dial(server)
	if err != nil {
		return err
	}

	defer func() { _ = client.Close() }()

	if err := client.Hello("localhost"); err != nil {
		return err
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(tlsCfg); err != nil {
			return err
		}
	}

	if c.cfg.User != "" {
		auth := smtp.PlainAuth("", c.cfg.User, c.cfg.Password, c.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}

	from, err := mail.ParseAddress(c.cfg.From)
	if err != nil {
		return err
	}

	to, err := mail.ParseAddress(payload.To)
	if err != nil {
		return err
	}

	if err := client.Mail(from.Address); err != nil {
		return err
	}
	if err := client.Rcpt(to.Address); err != nil {
		return err
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	if err := client.Quit(); err != nil {
		return err
	}

	return nil
}

type AttachmentsSizeExceededError struct {
	TotalSize int64
	MaxSize   int
}

func (e *AttachmentsSizeExceededError) Error() string {
	return fmt.Sprintf(
		"la dimensione totale degli allegati (%d bytes) supera il limite massimo consentito di %d bytes",
		e.TotalSize, e.MaxSize,
	)
}

func (e *AttachmentsSizeExceededError) Reason() string {
	return "La dimensione totale degli allegati supera il limite massimo consentito"
}

func (c *Client) checkAttachmentsSize(attachments email.AttachmentList, basePath string) error {
	if c.maxAttachmentsSize <= 0 || len(attachments) == 0 {
		return nil
	}

	var totalSize int64
	for _, att := range attachments {
		info, err := os.Stat(basePath + att.Path)
		if err != nil {
			return fmt.Errorf("impossibile verificare la dimensione dell'allegato %s: %w", att.Name, err)
		}
		totalSize += info.Size()
	}

	if totalSize > int64(c.maxAttachmentsSize) {
		return &AttachmentsSizeExceededError{TotalSize: totalSize, MaxSize: c.maxAttachmentsSize}
	}

	return nil
}
