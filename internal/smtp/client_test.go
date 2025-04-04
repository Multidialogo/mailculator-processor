//go:build unit

package smtp

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, &ClientTestSuite{})
}

type ClientTestSuite struct {
	suite.Suite
	sut *Client
}

func (suite *ClientTestSuite) SetupTest() {
	cfg := Config{User: "", Password: "", Host: "smtp.gmail.com", Port: 587, From: "", AllowInsecureTls: false}

	suite.sut = New(cfg)
}

func (suite *ClientTestSuite) TestClientSendError() {
	err := suite.sut.Send("testdata/missing.EML")
	suite.Assert().Equal("open testdata/missing.EML: no such file or directory", err.Error())
}

func (suite *ClientTestSuite) TestClientSendWithNoRecipient() {
	err := suite.sut.Send("testdata/no_recipient.EML")
	suite.Assert().Equal("could not find recipient in reader", err.Error())
}

func (suite *ClientTestSuite) TestClientSendWithFakeRecipient() {
	err := suite.sut.Send("testdata/fake_recipient.EML")
	suite.Assert().Equal("mail: missing '@' or angle-addr", err.Error())
}
