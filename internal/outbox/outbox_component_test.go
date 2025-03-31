package outbox

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mailculator-processor/internal/testutils/facades"
)

var fixtures map[string]string

func deleteFixtures(t *testing.T, db *dynamodb.Client) {
	if len(fixtures) == 0 {
		t.Log("no fixtures to delete")
		return
	}

	t.Logf("deleting fixtures: %v", fixtures)

	query := fmt.Sprintf("DELETE FROM \"%v\" WHERE Id=? AND Status=?", "Outbox")
	for id, status := range fixtures {
		params, _ := attributevalue.MarshalList([]interface{}{id, status})
		stmt := &dynamodb.ExecuteStatementInput{Statement: aws.String(query), Parameters: params}

		if _, err := db.ExecuteStatement(context.TODO(), stmt); err != nil {
			t.Errorf("error while deleting fixture %s, error: %v", id, err)
		}
	}
}

func TestOutboxComponentWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("component tests are skipped in short mode")
	}

	awsConfig := facades.NewAwsConfigFromEnv()
	db := dynamodb.NewFromConfig(awsConfig)
	sut := NewOutbox(db)

	fixtures = map[string]string{}
	defer deleteFixtures(t, db)

	of := facades.NewOutboxFacade()

	// no record in db, should return 0
	res, err := sut.Query(context.TODO(), "PENDING", 25)
	require.NoError(t, err)
	require.Len(t, res, 0)

	// insert two records in db
	id, err := of.AddEmail(context.TODO())
	require.NoErrorf(t, err, "failed inserting id %s, error: %v", id, err)
	fixtures[id] = "PENDING"

	anotherId, err := of.AddEmail(context.TODO())
	require.NoErrorf(t, err, "failed inserting id %s, error: %v", anotherId, err)
	fixtures[anotherId] = "PENDING"

	// filtering by status PENDING should return 2 records at this point
	res, err = sut.Query(context.TODO(), "PENDING", 25)
	require.NoError(t, err)
	require.Len(t, res, 2)

	// same filter with limit 1 should give only 1 record
	res, err = sut.Query(context.TODO(), "PENDING", 1)
	require.NoError(t, err)
	require.Len(t, res, 1)

	// filtering by status PROCESSING should return 0 records at this point
	res, err = sut.Query(context.TODO(), "PROCESSING", 25)
	require.NoError(t, err)
	require.Len(t, res, 0)

	// update fixture to status PROCESSING
	err = sut.Update(context.TODO(), id, "PROCESSING")
	require.NoError(t, err)
	fixtures[id] = "PROCESSING"

	// filtering by status PENDING should return 1 record at this point, with status PENDING
	res, err = sut.Query(context.TODO(), "PENDING", 25)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "PENDING", res[0].Status)

	// filtering by status PROCESSING should return 1 records at this point, with status PROCESSING
	res, err = sut.Query(context.TODO(), "PROCESSING", 25)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "PROCESSING", res[0].Status)

	// item already is in status PROCESSING, so it should return error
	err = sut.Update(context.TODO(), id, "PROCESSING")
	assert.Error(t, err)

	// status cannot be rolled back
	err = sut.Update(context.TODO(), id, "PENDING")
	if err == nil {
		fixtures[id] = "PENDING"
		t.Errorf("expected error, got nil")
	}
}
