//go:build unit

package outbox

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dynamodbMock struct {
	statementOutput   *dynamodb.ExecuteStatementOutput
	transactionOutput *dynamodb.ExecuteTransactionOutput
	returnError       error
}

func (m *dynamodbMock) ExecuteStatement(_ context.Context, _ *dynamodb.ExecuteStatementInput, _ ...func(options *dynamodb.Options)) (*dynamodb.ExecuteStatementOutput, error) {
	return m.statementOutput, m.returnError
}

func (m *dynamodbMock) ExecuteTransaction(_ context.Context, _ *dynamodb.ExecuteTransactionInput, _ ...func(options *dynamodb.Options)) (*dynamodb.ExecuteTransactionOutput, error) {
	return m.transactionOutput, m.returnError
}

func TestQuery_WhenDatabaseHasRecord_ShouldReturnMarshalledEmail(t *testing.T) {
	t.Parallel()

	record, _ := attributevalue.MarshalMap(map[string]interface{}{
		"Id":     "12345",
		"Status": "_META",
		"Attributes": map[string]interface{}{
			"Latest":          "PENDING",
			"EMLFilePath":     "/efs/email.eml",
			"SuccessCallback": "curl http://localhost:8000/success",
			"FailureCallback": "curl http://localhost:8000/failure",
		},
	})

	dbMock := &dynamodbMock{
		statementOutput: &dynamodb.ExecuteStatementOutput{
			Items: []map[string]types.AttributeValue{record},
		},
	}

	sut := &Outbox{db: dbMock}

	actual, err := sut.Query(context.TODO(), "ANY", 10)
	assert.NoError(t, err)
	require.Len(t, actual, 1)
	assert.Equal(t, actual[0].Status, "PENDING")
}

func TestQuery_WhenDatabaseReturnError_ShouldReturnSameError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("some error")
	dbMock := &dynamodbMock{returnError: expectedError}
	sut := &Outbox{db: dbMock}

	_, err := sut.Query(context.TODO(), "ANY", 10)
	assert.ErrorIs(t, expectedError, err)
}

func TestUpdate_WhenDatabaseReturnNoError_ShouldReturnNoError(t *testing.T) {
	t.Parallel()

	dbMock := &dynamodbMock{transactionOutput: &dynamodb.ExecuteTransactionOutput{}}
	sut := &Outbox{db: dbMock}

	err := sut.Update(context.TODO(), "12345", "PENDING")
	assert.NoError(t, err)
}

func TestUpdate_WhenDatabaseReturnError_ShouldReturnSameError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("some error")
	dbMock := &dynamodbMock{returnError: expectedError}
	sut := &Outbox{db: dbMock}

	err := sut.Update(context.TODO(), "12345", "PENDING")
	assert.ErrorIs(t, expectedError, err)
}
