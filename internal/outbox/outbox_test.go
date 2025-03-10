package outbox

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
)

func TestOutboxTestSuiteIntegration(t *testing.T) {
	suite.Run(t, &OutboxTestSuite{})
}

type OutboxTestSuite struct {
	suite.Suite
	db       *dynamodb.Client
	sut      *Outbox
	inserted []string
}

func (suite *OutboxTestSuite) SetupTest() {
	cfg := aws.Config{
		Region: os.Getenv("AWS_REGION"),
		Credentials: credentials.NewStaticCredentialsProvider(
			os.Getenv("AWS_ACCESS_KEY_ID"),
			os.Getenv("AWS_SECRET_ACCESS_KEY"),
			"",
		),
		BaseEndpoint: aws.String(os.Getenv("AWS_BASE_ENDPOINT")),
	}

	suite.db = dynamodb.NewFromConfig(cfg)
	suite.sut = NewOutbox(suite.db)
}

func (suite *OutboxTestSuite) TearDownTest() {
	query := fmt.Sprintf("SELECT Id, Status FROM \"%v\"", "Outbox")
	stmt := &dynamodb.ExecuteStatementInput{Statement: aws.String(query)}
	res, _ := suite.db.ExecuteStatement(context.TODO(), stmt)

	var items []emailItemRow
	_ = attributevalue.UnmarshalListOfMaps(res.Items, &items)

	query = fmt.Sprintf("DELETE FROM \"%v\" WHERE Id=? AND Status=?", "Outbox")
	for _, item := range items {
		params, _ := attributevalue.MarshalList([]interface{}{item.Id, item.Status})
		stmt = &dynamodb.ExecuteStatementInput{Statement: aws.String(query), Parameters: params}

		_, err := suite.db.ExecuteStatement(context.TODO(), stmt)
		suite.Require().NoError(err)
	}
}

func (suite *OutboxTestSuite) seeder(index int) Email {
	id := uuid.NewString()
	return Email{
		Id:              id,
		Status:          "PENDING",
		EmlFilePath:     "testdata/smol.EML",
		SuccessCallback: fmt.Sprintf("curl -X /success/%v", index),
		FailureCallback: fmt.Sprintf("curl -X /failure/%v", index),
	}
}

func (suite *OutboxTestSuite) TestMainOutboxQueryInsertUpdateIntegration() {
	ctx := context.TODO()
	res, err := suite.sut.Query(ctx, "PENDING", 25)
	suite.Assert().NoError(err)
	suite.Assert().Len(res, 0)

	fx := suite.seeder(0)
	err = suite.sut.Insert(ctx, fx)
	suite.Assert().NoError(err)

	res, err = suite.sut.Query(ctx, "PENDING", 25)
	suite.Assert().Len(res, 1)
	suite.Assert().Equal(fx.Id, res[0].Id)
	suite.Assert().Equal("PENDING", res[0].Status)

	err = suite.sut.Insert(ctx, fx)
	suite.Assert().Error(err)

	res, err = suite.sut.Query(ctx, "PENDING", 25)
	suite.Assert().Len(res, 1)
	suite.Assert().Equal(fx.Id, res[0].Id)
	suite.Assert().Equal("PENDING", res[0].Status)

	res, err = suite.sut.Query(ctx, "PROCESSING", 25)
	suite.Assert().Len(res, 0)

	err = suite.sut.Update(ctx, fx.Id, "PROCESSING")
	suite.Assert().NoError(err)

	res, err = suite.sut.Query(ctx, "PENDING", 25)
	suite.Assert().Len(res, 0)

	res, err = suite.sut.Query(ctx, "PROCESSING", 25)
	suite.Assert().Len(res, 1)
	suite.Assert().Equal(fx.Id, res[0].Id)
	suite.Assert().Equal("PROCESSING", res[0].Status)

	err = suite.sut.Update(ctx, fx.Id, "PROCESSING")
	suite.Assert().Error(err)

	res, err = suite.sut.Query(ctx, "PROCESSING", 25)
	suite.Assert().Len(res, 1)
	suite.Assert().Equal(fx.Id, res[0].Id)
	suite.Assert().Equal("PROCESSING", res[0].Status)
}
