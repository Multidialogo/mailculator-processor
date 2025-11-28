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
		},
		"TTL": "invalid-string-value",
	})

	dbMock := &dynamodbMock{
		statementOutput: &dynamodb.ExecuteStatementOutput{
			Items: []map[string]types.AttributeValue{record},
		},
	}

	sut := &Outbox{db: dbMock}

	_, err := sut.Query(context.TODO(), "ANY", 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal failed")
	assert.Contains(t, err.Error(), "cannot unmarshal string into Go value type int64")
}

func TestQuery_WhenTTLIsAtRoot_ShouldUseRootTTL(t *testing.T) {
	t.Parallel()

	expectedTTL := int64(1234567890)
	record, _ := attributevalue.MarshalMap(map[string]any{
		"Id":     "12345",
		"Status": "_META",
		"Attributes": map[string]any{
			"Latest":      "READY",
			"CreatedAt":   time.Now().Format(time.RFC3339),
			"EMLFilePath": "/efs/email.eml",
		},
		"TTL": expectedTTL,
	})

	dbMock := &dynamodbMock{
		statementOutput: &dynamodb.ExecuteStatementOutput{
			Items: []map[string]types.AttributeValue{record},
		},
	}

	sut := &Outbox{db: dbMock}

	emails, err := sut.Query(context.TODO(), "ANY", 10)
	assert.NoError(t, err)
	assert.Len(t, emails, 1)
	assert.Equal(t, expectedTTL, emails[0].TTL)
}

func TestQuery_WhenTTLIsOnlyInAttributes_ShouldUseAttributesTTL(t *testing.T) {
	t.Parallel()

	expectedTTL := int64(9876543210)
	record, _ := attributevalue.MarshalMap(map[string]any{
		"Id":     "12345",
		"Status": "_META",
		"Attributes": map[string]any{
			"Latest":      "READY",
			"CreatedAt":   time.Now().Format(time.RFC3339),
			"EMLFilePath": "/efs/email.eml",
			"TTL":         expectedTTL, // TTL nel vecchio formato (in Attributes)
		},
		// Nota: nessun TTL alla radice
	})

	dbMock := &dynamodbMock{
		statementOutput: &dynamodb.ExecuteStatementOutput{
			Items: []map[string]types.AttributeValue{record},
		},
	}

	sut := &Outbox{db: dbMock}

	emails, err := sut.Query(context.TODO(), "ANY", 10)
	assert.NoError(t, err)
	assert.Len(t, emails, 1)
	assert.Equal(t, expectedTTL, emails[0].TTL)
}

func TestQuery_WhenTTLIsMissingEverywhere_ShouldReturnZeroTTL(t *testing.T) {
	t.Parallel()

	record, _ := attributevalue.MarshalMap(map[string]any{
		"Id":     "12345",
		"Status": "_META",
		"Attributes": map[string]any{
			"Latest":      "READY",
			"CreatedAt":   time.Now().Format(time.RFC3339),
			"EMLFilePath": "/efs/email.eml",
			// Nota: nessun TTL né in Attributes né alla radice
		},
	})

	dbMock := &dynamodbMock{
		statementOutput: &dynamodb.ExecuteStatementOutput{
			Items: []map[string]types.AttributeValue{record},
		},
	}

	sut := &Outbox{db: dbMock}

	emails, err := sut.Query(context.TODO(), "ANY", 10)
	assert.NoError(t, err)
	assert.Len(t, emails, 1)
	assert.Equal(t, int64(0), emails[0].TTL)
}

func TestQuery_WhenTTLIsAtRootAndInAttributes_ShouldPreferRootTTL(t *testing.T) {
	t.Parallel()

	rootTTL := int64(1111111111)
	attributesTTL := int64(2222222222) // Questo dovrebbe essere ignorato

	record, _ := attributevalue.MarshalMap(map[string]any{
		"Id":     "12345",
		"Status": "_META",
		"Attributes": map[string]any{
			"Latest":      "READY",
			"CreatedAt":   time.Now().Format(time.RFC3339),
			"EMLFilePath": "/efs/email.eml",
			"TTL":         attributesTTL, // TTL nel vecchio formato
		},
		"TTL": rootTTL, // TTL nel nuovo formato - dovrebbe avere priorità
	})

	dbMock := &dynamodbMock{
		statementOutput: &dynamodb.ExecuteStatementOutput{
			Items: []map[string]types.AttributeValue{record},
		},
	}

	sut := &Outbox{db: dbMock}

	emails, err := sut.Query(context.TODO(), "ANY", 10)
	assert.NoError(t, err)
	assert.Len(t, emails, 1)
	assert.Equal(t, rootTTL, emails[0].TTL) // Dovrebbe usare il TTL alla radice
}
