package email

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"mailculator-processor/internal/config"
	"testing"
)

type lockerMock struct {
	lockedIds []string
}

func (l *lockerMock) BatchLock(ids []string) ([]string, error) {
	l.lockedIds = ids
	return ids, nil
}

func TestFinderTestSuite(t *testing.T) {
	suite.Run(t, &FinderTestSuite{})
}

type FinderTestSuite struct {
	suite.Suite
	dynamodbClient *dynamodb.Client
	locker         *lockerMock
	fixtureIds     []string
}

func (suite *FinderTestSuite) SetupTest() {
	cfg := config.NewConfig()
	suite.dynamodbClient = dynamodb.NewFromConfig(cfg.Aws.DynamoDb)
	suite.locker = &lockerMock{}
}

func (suite *FinderTestSuite) TearDownTest() {
	statementRequests := make([]types.BatchStatementRequest, len(suite.fixtureIds))
	for i, fixtureId := range suite.fixtureIds {
		params, err := attributevalue.MarshalList([]interface{}{fixtureId})
		if err != nil {
			suite.FailNow("failed to marshal attribute during teardown")
		}

		statementRequests[i] = types.BatchStatementRequest{
			Statement:  aws.String(fmt.Sprintf("DELETE FROM \"%v\" WHERE id=?", "Outbox")),
			Parameters: params,
		}
	}

	_, err := suite.dynamodbClient.BatchExecuteStatement(context.TODO(), &dynamodb.BatchExecuteStatementInput{
		Statements: statementRequests,
	})

	if err != nil {
		suite.FailNow("failed to execute batch statement")
	}
}

func (suite *FinderTestSuite) insertWithStatus(status string) *Email {
	email := Email{
		Id:         uuid.NewString(),
		Attributes: map[string]any{"status": status, "eml_filepath": "", "success_callback": "", "failure_callback": ""},
	}

	item, err := attributevalue.MarshalMap(email)
	if err != nil {
		suite.FailNow("unable to insert fixture")
	}

	_, err = suite.dynamodbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String("Outbox"), Item: item,
	})

	if err != nil {
		suite.FailNow("unable to insert fixture", err.Error())
	}

	suite.fixtureIds = append(suite.fixtureIds, email.Id)
	return &email
}

func (suite *FinderTestSuite) Test_Finder_FindAndLock() {
	ready := suite.insertWithStatus("READY")
	suite.insertWithStatus("NOT-READY")

	sut := &Finder{locker: suite.locker, outboxTableName: "Outbox", dynamoDbClient: suite.dynamodbClient}
	found, err := sut.FindAndLock()

	suite.Require().Nil(err)
	suite.Require().Equal(1, len(found))
	suite.Assert().Equal(ready.Id, found[0].Id)
	suite.Require().Equal(1, len(suite.locker.lockedIds))
	suite.Assert().Equal(ready.Id, suite.locker.lockedIds[0])
}
