package smtp

import (
	"crypto/tls"
	"fmt"
	"net/mail"
	"net/smtp"

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
	cfg     Config
	builder *MessageBuilder
}

func New(cfg Config) *Client {
	return &Client{
		cfg:     cfg,
		builder: &MessageBuilder{},
	}
}

func (c *Client) Send(payload email.Payload, attachmentsBasePath string) error {
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
