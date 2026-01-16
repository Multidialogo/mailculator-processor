//go:build unit

package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuery_WhenDatabaseHasRecords_ShouldReturnEmails(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "status", "eml_file_path", "payload_file_path", "reason", "version", "updated_at"}).
		AddRow("test-id-1", "READY", "", "/path/to/payload", "", 1, now).
		AddRow("test-id-2", "READY", "", "/path/to/payload2", "some reason", 2, now)

	mock.ExpectQuery("SELECT id, status, eml_file_path, payload_file_path, reason, version, updated_at FROM emails").
		WithArgs("READY", 25).
		WillReturnRows(rows)

	sut := NewOutboxWithDB(db)

	emails, err := sut.Query(context.TODO(), StatusReady, 25)

	assert.NoError(t, err)
	require.Len(t, emails, 2)
	assert.Equal(t, "test-id-1", emails[0].Id)
	assert.Equal(t, "READY", emails[0].Status)
	assert.Equal(t, "", emails[0].EmlFilePath)
	assert.Equal(t, "test-id-2", emails[1].Id)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestQuery_WhenDatabaseReturnsError_ShouldReturnError(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectedError := errors.New("database error")
	mock.ExpectQuery("SELECT").WillReturnError(expectedError)

	sut := NewOutboxWithDB(db)

	_, err = sut.Query(context.TODO(), StatusReady, 25)

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestQuery_WhenDatabaseHasNoRecords_ShouldReturnEmptySlice(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "status", "eml_file_path", "payload_file_path", "reason", "version", "updated_at"})

	mock.ExpectQuery("SELECT").
		WithArgs("READY", 10).
		WillReturnRows(rows)

	sut := NewOutboxWithDB(db)

	emails, err := sut.Query(context.TODO(), StatusReady, 10)

	assert.NoError(t, err)
	assert.Len(t, emails, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdate_WhenUpdateSucceeds_ShouldReturnNoError(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE emails").
		WithArgs("PROCESSING", "", "test-id", "READY").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO email_statuses").
		WithArgs("test-id", "PROCESSING", "").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	sut := NewOutboxWithDB(db)

	err = sut.Update(context.TODO(), "test-id", StatusProcessing, "", nil)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdate_WhenNoRowsAffected_ShouldReturnLockError(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE emails").
		WithArgs("PROCESSING", "", "test-id", "READY").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	sut := NewOutboxWithDB(db)

	err = sut.Update(context.TODO(), "test-id", StatusProcessing, "", nil)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrLockNotAcquired)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdate_WhenDatabaseReturnsError_ShouldReturnError(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectedError := errors.New("database error")
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE emails").
		WithArgs("PROCESSING", "", "test-id", "READY").
		WillReturnError(expectedError)
	mock.ExpectRollback()

	sut := NewOutboxWithDB(db)

	err = sut.Update(context.TODO(), "test-id", StatusProcessing, "", nil)

	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReady_WhenUpdateSucceeds_ShouldReturnNoError(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE emails").
		WithArgs("READY", "", "test-id", "INTAKING").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO email_statuses").
		WithArgs("test-id", "READY", "").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	sut := NewOutboxWithDB(db)

	err = sut.Ready(context.TODO(), "test-id", "", nil)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReady_WhenNoRowsAffected_ShouldReturnLockError(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE emails").
		WithArgs("READY", "", "test-id", "INTAKING").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	sut := NewOutboxWithDB(db)

	err = sut.Ready(context.TODO(), "test-id", "", nil)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrLockNotAcquired)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetExpectedFromStatus_ShouldReturnCorrectTransitions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		toStatus   string
		fromStatus string
	}{
		{StatusIntaking, StatusAccepted},
		{StatusReady, StatusIntaking},
		{StatusProcessing, StatusReady},
		{StatusSent, StatusProcessing},
		{StatusFailed, StatusProcessing},
		{StatusInvalid, StatusIntaking},
		{StatusCallingSentCallback, StatusSent},
		{StatusCallingFailedCallback, StatusFailed},
		{StatusSentAcknowledged, StatusCallingSentCallback},
		{StatusFailedAcknowledged, StatusCallingFailedCallback},
	}

	for _, tc := range testCases {
		t.Run(tc.toStatus, func(t *testing.T) {
			result := getExpectedFromStatus(tc.toStatus)
			assert.Equal(t, tc.fromStatus, result)
		})
	}
}

func TestCreate_WhenInsertSucceeds_ShouldReturnNoError(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO emails").
		WithArgs("test-id", "ACCEPTED", "/path/to/payload").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO email_statuses").
		WithArgs("test-id", "ACCEPTED", "").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	sut := NewOutboxWithDB(db)

	err = sut.Create(context.TODO(), "test-id", StatusAccepted, "/path/to/payload")

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_WhenDeleteSucceeds_ShouldReturnNoError(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("DELETE FROM emails").
		WithArgs("test-id").
		WillReturnResult(sqlmock.NewResult(0, 1))

	sut := NewOutboxWithDB(db)

	err = sut.Delete(context.TODO(), "test-id")

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestShouldRetryMySQL_WhenErrLockNotAcquired_ShouldReturnFalse(t *testing.T) {
	t.Parallel()

	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sut := NewOutboxWithDB(db)

	result := sut.shouldRetryMySQL(ErrLockNotAcquired)

	assert.False(t, result)
}

func TestShouldRetryMySQL_WhenNilError_ShouldReturnFalse(t *testing.T) {
	t.Parallel()

	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sut := NewOutboxWithDB(db)

	result := sut.shouldRetryMySQL(nil)

	assert.False(t, result)
}

func TestShouldRetryMySQL_WhenGenericError_ShouldReturnFalse(t *testing.T) {
	t.Parallel()

	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sut := NewOutboxWithDB(db)

	result := sut.shouldRetryMySQL(errors.New("some generic error"))

	assert.False(t, result)
}

func TestBackoffDuration_ShouldReturnPositiveDuration(t *testing.T) {
	t.Parallel()

	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sut := NewOutboxWithDB(db)

	for attempt := 0; attempt < 10; attempt++ {
		duration := sut.backoffDuration(attempt)
		assert.GreaterOrEqual(t, int64(duration), int64(0))
		assert.LessOrEqual(t, duration, maxDelay)
	}
}
