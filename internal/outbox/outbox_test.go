//go:build unit

package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

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

	record, _ := attributevalue.MarshalMap(map[string]any{
		"Id":     "12345",
		"Status": "_META",
		"Attributes": map[string]any{
			"Latest":      "READY",
			"CreatedAt":   time.Now().Format(time.RFC3339),
			"EMLFilePath": "/efs/email.eml",
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
	assert.Equal(t, actual[0].Status, "READY")
}

func TestQueryLimit(t *testing.T) {
	t.Parallel()

	record1, _ := attributevalue.MarshalMap(map[string]any{
		"Id":     "12345",
		"Status": "_META",
		"Attributes": map[string]any{
			"Latest":      "READY",
			"CreatedAt":   time.Now().Format(time.RFC3339),
			"EMLFilePath": "/efs/email.eml",
		},
	})
	record2, _ := attributevalue.MarshalMap(map[string]any{
		"Id":     "12345",
		"Status": "_META",
		"Attributes": map[string]any{
			"Latest":      "READY",
			"CreatedAt":   time.Now().Format(time.RFC3339),
			"EMLFilePath": "/efs/email.eml",
		},
	})

	dbMock := &dynamodbMock{
		statementOutput: &dynamodb.ExecuteStatementOutput{
			Items: []map[string]types.AttributeValue{record1, record2},
		},
	}

	sut := NewOutbox(nil, "Outbox")
	sut.db = dbMock

	actual, err := sut.Query(context.TODO(), "ANY", 1)
	assert.NoError(t, err)
	require.Len(t, actual, 1)
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

	err := sut.Update(context.TODO(), "12345", "READY", "", 1234567890)
	assert.NoError(t, err)
}

func TestUpdate_WhenDatabaseReturnError_ShouldReturnSameError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("some error")
	dbMock := &dynamodbMock{returnError: expectedError}
	sut := &Outbox{db: dbMock}

	err := sut.Update(context.TODO(), "12345", "READY", "", 1234567890)
	assert.ErrorIs(t, expectedError, err)
}

func TestQuery_WhenTTLIsInvalidType_ShouldReturnError(t *testing.T) {
	t.Parallel()

	record, _ := attributevalue.MarshalMap(map[string]any{
		"Id":     "12345",
		"Status": "_META",
		"Attributes": map[string]any{
			"Latest":      "READY",
			"CreatedAt":   time.Now().Format(time.RFC3339),
			"EMLFilePath": "/efs/email.eml",
			"TTL":         "invalid-string-value",
		},
	})

	dbMock := &dynamodbMock{
		statementOutput: &dynamodb.ExecuteStatementOutput{
			Items: []map[string]types.AttributeValue{record},
		},
	}

	sut := &Outbox{db: dbMock}

	_, err := sut.Query(context.TODO(), "ANY", 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error unmarshalling TTL")
	assert.Contains(t, err.Error(), "TTL must be an integer")
}
