package email

import (
	"github.com/stretchr/testify/suite"
	"mailculator-processor/internal/config"
	"testing"
)

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, &ClientTestSuite{})
}

type ClientTestSuite struct {
	suite.Suite
	cfg config.SmtpConfig
}

func (suite *ClientTestSuite) SetupTest() {
	suite.cfg = config.NewConfig().Smtp
}

func (suite *ClientTestSuite) TestSenderSend() {
	email := Email{
		Id:              "fake-id",
		Status:          "READY",
		To:              "fake@email.com",
		EmlFilePath:     "testdata/large.EML",
		SuccessCallback: "",
		FailureCallback: "",
	}

	factory := NewClientFactory(suite.cfg)
	sut, err := factory.New()
	suite.Require().NoError(err)

	defer sut.Close()

	ok, err := sut.Send(email)
	suite.Require().NoError(err)
	suite.Assert().True(ok)
}
