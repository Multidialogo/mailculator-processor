//go:build unit

package smtp

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var client *Client

func init() {
	cfg := Config{User: "", Password: "", Host: "smtp.gmail.com", Port: 587, From: "", AllowInsecureTls: false}

	client = New(cfg)
}

func TestClientSendError(t *testing.T) {
	err := client.Send("testdata/missing.EML")
	assert.Equal(t, "open testdata/missing.EML: no such file or directory", err.Error())
}

func TestClientSendWithNoRecipient(t *testing.T) {
	err := client.Send("testdata/no_recipient.EML")
	assert.Equal(t, "could not find recipient in reader", err.Error())
}

func TestClientSendWithFakeRecipient(t *testing.T) {
	err := client.Send("testdata/fake_recipient.EML")
	assert.Equal(t, "mail: missing '@' or angle-addr", err.Error())
}
