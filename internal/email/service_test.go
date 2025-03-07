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
	query := fmt.Sprintf("DELETE FROM \"%v\" WHERE id=?", tableName)

	for _, id := range suite.insertedIds {
		params, err := attributevalue.MarshalList([]interface{}{id})
		if err != nil {
			log.Println("error marshalling inserted id", err)
		}

		stmt := &dynamodb.ExecuteStatementInput{
			Statement:  aws.String(query),
			Parameters: params,
		}

		_, err = suite.db.ExecuteStatement(context.TODO(), stmt)
		if err != nil {
			log.Println("error deleting inserted email", err)
		}
	}
}

func (suite *ServiceTestSuite) insert(email Email) error {
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

	_, err = suite.db.ExecuteStatement(context.TODO(), stmt)
	if err != nil {
		return err
	}

	suite.insertedIds = append(suite.insertedIds, email.Id)
	return nil
}

func (suite *ServiceTestSuite) TestFindReadyIntegration() {
	email := Email{
		Id:              "fake-id",
		Status:          "READY",
		EmlFilePath:     "testdata/large.EML",
		SuccessCallback: "/success",
		FailureCallback: "/failure",
	}

	err := suite.insert(email)
	suite.Require().NoError(err)

	sut := NewService(suite.db)
	found, err := sut.FindReady(context.TODO())

	suite.Require().NoError(err)
	suite.Assert().Len(suite.insertedIds, 1)
	suite.Assert().Len(found, 1)
}
