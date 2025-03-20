package smtp

import (
	"github.com/stretchr/testify/suite"
	"os"
	"strconv"
	"sync"
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
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	cfg := Config{
		User:             os.Getenv("SMTP_USER"),
		Password:         os.Getenv("SMTP_PASS"),
		Host:             os.Getenv("SMTP_HOST"),
		Port:             port,
		From:             os.Getenv("SMTP_FROM"),
		AllowInsecureTls: true,
	}

	suite.sut = New(cfg)
}

func (suite *ClientTestSuite) TestClientSendIntegration() {
	var wg sync.WaitGroup

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := suite.sut.Send("testdata/smol.EML")
			suite.Require().NoError(err)
		}()
	}

	wg.Wait()
}
