package email

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/suite"
	"log"
	"mailculator-processor/internal/config"
	"testing"
)

func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, &ServiceTestSuite{})
}

type ServiceTestSuite struct {
	suite.Suite
	db          *dynamodb.Client
	insertedIds []string
}

func (suite *ServiceTestSuite) SetupTest() {
	suite.db = dynamodb.NewFromConfig(config.NewConfig().Aws.DynamoDb)
	suite.insertedIds = []string{}
}

func (suite *ServiceTestSuite) TearDownTest() {
	outboxQuery := fmt.Sprintf("DELETE FROM \"%v\" WHERE id=?", tableName)
	outboxLockQuery := fmt.Sprintf("DELETE FROM \"%v\" WHERE id=?", tableName)

	for _, id := range suite.insertedIds {
		params, err := attributevalue.MarshalList([]interface{}{id})
		if err != nil {
			log.Println("error marshalling inserted id", err)
			continue
		}

		stmt := &dynamodb.ExecuteStatementInput{
			Statement:  aws.String(outboxQuery),
			Parameters: params,
		}

		if _, err = suite.db.ExecuteStatement(context.TODO(), stmt); err != nil {
			log.Println("error deleting inserted email", err)
			continue
		}

		stmt = &dynamodb.ExecuteStatementInput{
			Statement:  aws.String(outboxLockQuery),
			Parameters: params,
		}

		if _, err = suite.db.ExecuteStatement(context.TODO(), stmt); err != nil {
			log.Println("error deleting inserted lock", err)
			continue
		}
	}
}

func (suite *ServiceTestSuite) insertFixture(email Email) error {
	query := fmt.Sprintf("INSERT INTO \"%v\" VALUE {'id': ?, 'attributes': ?}", tableName)

	marshaller := &emailMarshaller{}
	params, err := marshaller.Marshal(email)
	if err != nil {
		return err
	}

	stmt := &dynamodb.ExecuteStatementInput{
		Statement:  aws.String(query),
		Parameters: []types.AttributeValue{params["id"], params["attributes"]},
	}

	if _, err = suite.db.ExecuteStatement(context.TODO(), stmt); err != nil {
		return err
	}

	suite.insertedIds = append(suite.insertedIds, email.Id)
	return nil
}

func (suite *ServiceTestSuite) Test_FindReady_Integration() {
	fixtures := []Email{
		{
			Id:              "fake-id-0",
			Status:          "",
			EmlFilePath:     "testdata/none.EML",
			SuccessCallback: "echo /success/0",
			FailureCallback: "echo /failure/0",
		},
		{
			Id:              "fake-id-1",
			Status:          "READY",
			EmlFilePath:     "testdata/none.EML",
			SuccessCallback: "echo /success/1",
			FailureCallback: "echo /failure/1",
		},
		{
			Id:              "fake-id-2",
			Status:          "SENT",
			EmlFilePath:     "testdata/none.EML",
			SuccessCallback: "echo /success/2",
			FailureCallback: "echo /failure/2",
		},
		{
			Id:              "fake-id-3",
			Status:          "SENT-ACK",
			EmlFilePath:     "testdata/none.EML",
			SuccessCallback: "echo /success/3",
			FailureCallback: "echo /failure/3",
		},
		{
			Id:              "fake-id-4",
			Status:          "FAILED",
			EmlFilePath:     "testdata/none.EML",
			SuccessCallback: "echo /success/4",
			FailureCallback: "echo /failure/4",
		},
		{
			Id:              "fake-id-5",
			Status:          "FAILED-ACK",
			EmlFilePath:     "testdata/none.EML",
			SuccessCallback: "echo /success/5",
			FailureCallback: "echo /failure/5",
		},
		{
			Id:              "fake-id-6",
			Status:          "READY",
			EmlFilePath:     "testdata/none.EML",
			SuccessCallback: "echo /success/6",
			FailureCallback: "echo /failure/6",
		},
	}

	for _, fixture := range fixtures {
		err := suite.insertFixture(fixture)
		suite.Require().NoError(err)
	}

	sut := NewService(suite.db)
	found, err := sut.FindReady(context.TODO())

	suite.Require().NoError(err)
	suite.Assert().ElementsMatch([]Email{fixtures[1], fixtures[6]}, found)
}
