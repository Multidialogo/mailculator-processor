package outbox

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/suite"
	"mailculator-processor/internal/testutils"
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
	res, err := suite.db.ExecuteStatement(context.TODO(), stmt)
	suite.Require().NoError(err)

	var items []emailItemRow
	_ = attributevalue.UnmarshalListOfMaps(res.Items, &items)

	query = fmt.Sprintf("DELETE FROM \"%v\" WHERE Id=? AND Status=?", "Outbox")
	for _, item := range items {
		params, _ := attributevalue.MarshalList([]interface{}{item.Id, item.Status})
		stmt = &dynamodb.ExecuteStatementInput{Statement: aws.String(query), Parameters: params}

		_, err = suite.db.ExecuteStatement(context.TODO(), stmt)
		suite.Assert().NoError(err)
	}
}

func (suite *OutboxTestSuite) TestMainOutboxQueryInsertUpdateIntegration() {
	ctx := context.TODO()

	// no record in db, should return 0
	res, err := suite.sut.Query(ctx, "PENDING", 25)
	suite.Assert().NoError(err)
	suite.Assert().Len(res, 0)

	// insert a record in db
	of := testutils.NewOutboxFacade()
	id, err := of.AddEmail(ctx)
	suite.Assert().NoError(err)

	// filtering by status PENDING should return 1 record at this point, the same record inserted before
	res, err = suite.sut.Query(ctx, "PENDING", 25)
	suite.Assert().Len(res, 1)
	suite.Assert().Equal(id, res[0].Id)
	suite.Assert().Equal("PENDING", res[0].Status)

	// filtering by status PROCESSING should return 0 records at this point
	res, err = suite.sut.Query(ctx, "PROCESSING", 25)
	suite.Assert().Len(res, 0)

	// update fixture to status PROCESSING
	err = suite.sut.Update(ctx, id, "PROCESSING")
	suite.Assert().NoError(err)

	// filtering by status PENDING should return 0 records at this point
	res, err = suite.sut.Query(ctx, "PENDING", 25)
	suite.Assert().Len(res, 0)

	// filtering by status PROCESSING should return 1 records at this point
	res, err = suite.sut.Query(ctx, "PROCESSING", 25)
	suite.Assert().Len(res, 1)
	suite.Assert().Equal(id, res[0].Id)
	suite.Assert().Equal("PROCESSING", res[0].Status)

	// item already is in status PROCESSING, so it should return error
	err = suite.sut.Update(ctx, id, "PROCESSING")
	suite.Assert().Error(err)

	// status cannot be rolled back
	err = suite.sut.Update(ctx, id, "PENDING")
	suite.Assert().Error(err)
}
