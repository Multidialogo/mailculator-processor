package email

import (
	"crypto/tls"
	"errors"
	"fmt"
	"mailculator-processor/internal/config"
	"net/smtp"
	"os"
)

type Client struct {
	smtpClient *smtp.Client
	from       string
}

func (c *Client) Close() {
	_ = c.smtpClient.Close()
}

func (c *Client) Send(email Email) (bool, error) {
	if err := c.smtpClient.Mail(c.from); err != nil {
		return false, err
	}

	if err := c.smtpClient.Rcpt(email.To); err != nil {
		return false, err
	}

	w, err := c.smtpClient.Data()
	if err != nil {
		return false, err
	}

	raw, err := os.ReadFile(email.EmlFilePath)
	if err != nil {
		return false, err
	}

	if _, err = w.Write(raw); err != nil {
		return false, err
	}

	// TODO defer close
	if err = w.Close(); err != nil {
		return false, err
	}

	// TODO defer quit
	err = c.smtpClient.Quit()
	return err == nil, nil
}

type ClientFactory struct {
	cfg config.SmtpConfig
}

func NewClientFactory(cfg config.SmtpConfig) *ClientFactory {
	return &ClientFactory{cfg: cfg}
}

func (f *ClientFactory) New() (*Client, error) {
	c, err := smtp.Dial(fmt.Sprintf("%s:%d", f.cfg.Host, f.cfg.Port))
	if err != nil {
		return nil, err
	}

	if err = c.Hello("localhost"); err != nil {
		return nil, err
	}

	if ok, _ := c.Extension("STARTTLS"); !ok {
		return nil, errors.New("smtp: server doesn't support STARTTLS")
	}

	tlsCfg := &tls.Config{InsecureSkipVerify: f.cfg.AllowInsecureTls}

	if err = c.StartTLS(tlsCfg); err != nil {
		return nil, err
	}

	if ok, _ := c.Extension("AUTH"); !ok {
		return nil, errors.New("smtp: server doesn't support AUTH")
	}

	auth := smtp.PlainAuth("", f.cfg.Username, f.cfg.Password, f.cfg.Host)

	if err = c.Auth(auth); err != nil {
		return nil, err
	}

	return &Client{smtpClient: c, from: f.cfg.From}, nil
}
