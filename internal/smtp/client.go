package smtp

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net/mail"
	"os"
	"strings"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
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
	cfg Config
}

func New(cfg Config) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) Send(emlFilePath string) error {
	tlsCfg := &tls.Config{
		ServerName:         c.cfg.Host,
		InsecureSkipVerify: c.cfg.AllowInsecureTls,
	}

	server := fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port)
	client, err := smtp.DialStartTLS(server, tlsCfg)
	if err != nil {
		return err
	}

	defer func() { _ = client.Quit() }()
	defer func() { _ = client.Close() }()

	auth := sasl.NewPlainClient("", c.cfg.User, c.cfg.Password)
	if err := client.Auth(auth); err != nil {
		return err
	}

	reader, err := os.Open(emlFilePath)
	if err != nil {
		return err
	}

	defer func() { _ = reader.Close() }()

	rcpt, err := c.parseRecipient(reader)
	if err != nil {
		return err
	}

	return client.SendMail(c.cfg.From, []string{rcpt}, reader)
}

func (c *Client) parseRecipient(reader *os.File) (string, error) {
	defer func() { _, _ = reader.Seek(0, 0) }()

	scanner := bufio.NewScanner(reader)
	for i := 0; i < 256 && scanner.Scan(); i++ {
		line := scanner.Text()
		if strings.HasPrefix(line, "To:") {
			normal := strings.TrimPrefix(line, "To:")
			normal = strings.TrimSpace(normal)
			addr, err := mail.ParseAddress(normal)
			if err != nil {
				return "", err
			}

			return addr.Address, nil
		}
	}

	return "", fmt.Errorf("could not find recipient in reader")
}
