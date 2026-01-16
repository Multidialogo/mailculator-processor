//go:build repository

package outbox

import (
	"context"
	"testing"

	"mailculator-processor/internal/testutils/facades"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fixtures []string

func deleteFixtures(t *testing.T, facade *facades.MySQLOutboxFacade) {
	if len(fixtures) == 0 {
		t.Log("no fixtures to delete")
		return
	}

	t.Logf("deleting fixtures: %v", fixtures)

	for _, id := range fixtures {
		err := facade.DeleteEmail(context.Background(), id)
		if err != nil {
			t.Errorf("error while deleting fixture %s, error: %v", id, err)
		}
	}
}

func TestMySQLOutboxComponentWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("component tests are skipped in short mode")
	}

	facade, err := facades.NewMySQLOutboxFacade()
	require.NoError(t, err, "failed to create MySQL facade")
	defer facade.Close()

	sut := NewOutbox(facade.GetDB())

	fixtures = make([]string, 0)
	defer deleteFixtures(t, facade)

	// no record in db, should return 0
	res, err := sut.Query(context.TODO(), StatusReady, 25)
	require.NoError(t, err)
	require.Len(t, res, 0)

	// insert two records in db
	id, err := facade.AddEmail(context.TODO(), "")
	require.NoErrorf(t, err, "failed inserting id %s, error: %v", id, err)
	fixtures = append(fixtures, id)

	anotherId, err := facade.AddEmail(context.TODO(), "")
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
	err = sut.Update(context.TODO(), id, StatusProcessing, "", nil)
	require.NoError(t, err)

	// filtering by status READY should return 1 record at this point
	res, err = sut.Query(context.TODO(), StatusReady, 25)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, StatusReady, res[0].Status)

	// filtering by status PROCESSING should return 1 record at this point
	res, err = sut.Query(context.TODO(), StatusProcessing, 25)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, StatusProcessing, res[0].Status)

	// item already is in status PROCESSING, trying to update from READY should fail
	err = sut.Update(context.TODO(), id, StatusProcessing, "", nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrLockNotAcquired)
}

func TestMySQLOutboxReadyWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("component tests are skipped in short mode")
	}

	facade, err := facades.NewMySQLOutboxFacade()
	require.NoError(t, err, "failed to create MySQL facade")
	defer facade.Close()

	sut := NewOutbox(facade.GetDB())

	fixtures = make([]string, 0)
	defer deleteFixtures(t, facade)

	// insert a record in INTAKING status
	id, err := facade.AddEmailWithStatus(context.TODO(), StatusIntaking, "")
	require.NoError(t, err)
	fixtures = append(fixtures, id)

	// update to READY
	err = sut.Ready(context.TODO(), id)
	require.NoError(t, err)

	// verify status changed to READY
	res, err := sut.Query(context.TODO(), StatusReady, 25)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, id, res[0].Id)
	assert.Equal(t, StatusReady, res[0].Status)

	// trying to call Ready again should fail (status is now READY, not INTAKING)
	err = sut.Ready(context.TODO(), id)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrLockNotAcquired)
}

func TestMySQLOutboxCreateAndDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("component tests are skipped in short mode")
	}

	facade, err := facades.NewMySQLOutboxFacade()
	require.NoError(t, err, "failed to create MySQL facade")
	defer facade.Close()

	sut := NewOutbox(facade.GetDB())

	// create a new email
	id := "test-create-delete-id"
	err = sut.Create(context.TODO(), id, StatusAccepted, "/path/to/payload.json")
	require.NoError(t, err)

	// verify it exists
	res, err := sut.Query(context.TODO(), StatusAccepted, 25)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, id, res[0].Id)

	// delete it
	err = sut.Delete(context.TODO(), id)
	require.NoError(t, err)

	// verify it's gone
	res, err = sut.Query(context.TODO(), StatusAccepted, 25)
	require.NoError(t, err)
	require.Len(t, res, 0)
}

func TestMySQLOutboxStateTransitions(t *testing.T) {
	if testing.Short() {
		t.Skip("component tests are skipped in short mode")
	}

	facade, err := facades.NewMySQLOutboxFacade()
	require.NoError(t, err, "failed to create MySQL facade")
	defer facade.Close()

	sut := NewOutbox(facade.GetDB())

	// Test the full happy path: ACCEPTED -> INTAKING -> READY -> PROCESSING -> SENT -> CALLING-SENT-CALLBACK -> SENT-ACKNOWLEDGED
	id := "test-transitions-id"
	defer func() {
		_ = sut.Delete(context.TODO(), id)
	}()

	// Create in ACCEPTED status
	err = sut.Create(context.TODO(), id, StatusAccepted, "/path/to/payload.json")
	require.NoError(t, err)

	// ACCEPTED -> INTAKING
	err = sut.Update(context.TODO(), id, StatusIntaking, "", nil)
	require.NoError(t, err)

	status, err := facade.GetEmailStatus(context.TODO(), id)
	require.NoError(t, err)
	assert.Equal(t, StatusIntaking, status)

	// INTAKING -> READY (using Ready method)
	err = sut.Ready(context.TODO(), id)
	require.NoError(t, err)

	status, err = facade.GetEmailStatus(context.TODO(), id)
	require.NoError(t, err)
	assert.Equal(t, StatusReady, status)

	// READY -> PROCESSING
	err = sut.Update(context.TODO(), id, StatusProcessing, "", nil)
	require.NoError(t, err)

	status, err = facade.GetEmailStatus(context.TODO(), id)
	require.NoError(t, err)
	assert.Equal(t, StatusProcessing, status)

	// PROCESSING -> SENT
	err = sut.Update(context.TODO(), id, StatusSent, "", nil)
	require.NoError(t, err)

	status, err = facade.GetEmailStatus(context.TODO(), id)
	require.NoError(t, err)
	assert.Equal(t, StatusSent, status)

	// SENT -> CALLING-SENT-CALLBACK
	err = sut.Update(context.TODO(), id, StatusCallingSentCallback, "", nil)
	require.NoError(t, err)

	status, err = facade.GetEmailStatus(context.TODO(), id)
	require.NoError(t, err)
	assert.Equal(t, StatusCallingSentCallback, status)

	// CALLING-SENT-CALLBACK -> SENT-ACKNOWLEDGED
	err = sut.Update(context.TODO(), id, StatusSentAcknowledged, "", nil)
	require.NoError(t, err)

	status, err = facade.GetEmailStatus(context.TODO(), id)
	require.NoError(t, err)
	assert.Equal(t, StatusSentAcknowledged, status)
}
