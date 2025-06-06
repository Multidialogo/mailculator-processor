//go:build repository

package outbox

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mailculator-processor/internal/testutils/facades"
)

const outboxTableName = "Outbox"

var fixtures []string

func deleteFixtures(t *testing.T, of *facades.OutboxFacade) {
	if len(fixtures) == 0 {
		t.Log("no fixtures to delete")
		return
	}

	t.Logf("deleting fixtures: %v", fixtures)

	for _, value := range fixtures {
		err := of.DeleteEmail(context.Background(), value)
		if err != nil {
			t.Errorf("error while deleting fixture %s, error: %v", value, err)
		}
	}
}

func TestOutboxComponentWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("component tests are skipped in short mode")
	}

	awsConfig := facades.NewAwsConfigFromEnv()
	db := dynamodb.NewFromConfig(awsConfig)
	sut := NewOutbox(db, outboxTableName)
	of, err := facades.NewOutboxFacade(outboxTableName, StatusMeta)

	fixtures = make([]string, 0)
	defer deleteFixtures(t, of)

	// no record in db, should return 0
	res, err := sut.Query(context.TODO(), StatusReady, 25)
	require.NoError(t, err)
	require.Len(t, res, 0)

	// insert two records in db
	id, err := of.AddEmail(context.TODO(), "")
	require.NoErrorf(t, err, "failed inserting id %s, error: %v", id, err)
	fixtures = append(fixtures, id)

	anotherId, err := of.AddEmail(context.TODO(), "")
	require.NoErrorf(t, err, "failed inserting id %s, error: %v", anotherId, err)
	fixtures = append(fixtures, anotherId)

	// filtering by status READY should return 2 records at this point
	res, err = sut.Query(context.TODO(), StatusReady, 25)
	require.NoError(t, err)
	require.Len(t, res, 2)

	// same filter with limit 1 should give only 1 record
	res, err = sut.Query(context.TODO(), StatusReady, 1)
	require.NoError(t, err)
	require.Len(t, res, 1)

	// filtering by status PROCESSING should return 0 records at this point
	res, err = sut.Query(context.TODO(), StatusProcessing, 25)
	require.NoError(t, err)
	require.Len(t, res, 0)

	// update fixture to status PROCESSING
	err = sut.Update(context.TODO(), id, StatusProcessing, "")
	require.NoError(t, err)

	// filtering by status READY should return 1 record at this point, with status READY
	res, err = sut.Query(context.TODO(), StatusReady, 25)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, StatusReady, res[0].Status)

	// filtering by status PROCESSING should return 1 records at this point, with status PROCESSING
	res, err = sut.Query(context.TODO(), StatusProcessing, 25)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, StatusProcessing, res[0].Status)

	// item already is in status PROCESSING, so it should return error
	err = sut.Update(context.TODO(), id, StatusProcessing, "")
	assert.Error(t, err)

	// status cannot be rolled back
	err = sut.Update(context.TODO(), id, StatusReady, "")
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}
