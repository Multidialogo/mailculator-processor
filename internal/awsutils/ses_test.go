package awsutils

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/stretchr/testify/suite"
	"mailculator-processor/internal/config"
	"testing"
)

func TestSesEmailClientTestSuite(t *testing.T) {
	suite.Run(t, &SesEmailClientTestSuite{})
}

type SesEmailClientTestSuite struct {
	suite.Suite
	client *ses.Client
}

func (suite *SesEmailClientTestSuite) SetupTest() {
	suite.client = ses.NewFromConfig(config.NewConfig().Aws.Ses)
}

func (suite *SesEmailClientTestSuite) Test_SesEmailClient_Send() {
	raw := `
	From: sender@test.multidialogo.it
	To: recipient@test.multidialogo.it
	Date: Thu, 01 Jan 1970 00:00:00 +0000
	Subject: Test Email
	Content-Type: multipart/mixed; boundary="message1"
	X-Custom-Header: CustomHeaderValue
	`

	sut := &SesEmailClient{client: suite.client}
	err := sut.Send(context.TODO(), []byte(raw))
	suite.Assert().Nil(err, err)
}
