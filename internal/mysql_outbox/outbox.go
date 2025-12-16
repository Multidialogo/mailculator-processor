package mysql_outbox

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"math/rand"
	"time"

	"mailculator-processor/internal/outbox"

	"github.com/go-sql-driver/mysql"
)

const (
	maxAttempts = 8
	baseDelay   = 30 * time.Millisecond
	maxDelay    = 1 * time.Second
)

var ErrLockNotAcquired = errors.New("lock not acquired: record was modified by another process")

// MySQL error numbers for retryable errors
var retryableErrNos = map[uint16]bool{
	1205: true, // Lock wait timeout exceeded
	1213: true, // Deadlock found
	1040: true, // Too many connections
	1203: true, // Max user connections exceeded
}

// sqlDBInterface defines the minimal interface for database operations
type sqlDBInterface interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type Outbox struct {
	db sqlDBInterface
}

func NewOutbox(db *sql.DB) *Outbox {
	return &Outbox{
		db: db,
	}
}

// NewOutboxWithDB creates an Outbox with a custom database interface (for testing)
func NewOutboxWithDB(db sqlDBInterface) *Outbox {
	return &Outbox{
		db: db,
	}
}

// shouldRetryMySQL checks if the error is a transient MySQL error that should be retried.
// It returns false for ErrLockNotAcquired (optimistic lock conflict).
func (o *Outbox) shouldRetryMySQL(err error) bool {
	if err == nil {
		return false
	}

	// Don't retry optimistic lock conflicts
	if errors.Is(err, ErrLockNotAcquired) {
		return false
	}

	// Check for MySQL-specific errors
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return retryableErrNos[mysqlErr.Number]
	}

	// Check for connection errors
	if errors.Is(err, driver.ErrBadConn) {
		return true
	}

	return false
}

// backoffDuration calculates the backoff duration for a given retry attempt
// using exponential backoff with jitter.
func (o *Outbox) backoffDuration(attempt int) time.Duration {
	max := min(time.Duration(1<<uint(attempt))*baseDelay, maxDelay)
	if max <= 0 {
		max = baseDelay
	}

	return time.Duration(rand.Int63n(int64(max)))
}

// executeInTransaction executes the given function within a database transaction.
// It handles commit on success and rollback on error.
func (o *Outbox) executeInTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (o *Outbox) Query(ctx context.Context, status string, limit int) ([]outbox.Email, error) {
	// FOR UPDATE SKIP LOCKED ensures:
	// - Rows currently locked by other transactions are skipped
	// - Reduces contention when multiple workers poll simultaneously
	query := `
		SELECT id, status, eml_file_path, payload_file_path, reason, version, updated_at
		FROM emails
		WHERE status = ?
		ORDER BY updated_at ASC
		LIMIT ?
		FOR UPDATE SKIP LOCKED
	`

	rows, err := o.db.QueryContext(ctx, query, status, limit)
	if err != nil {
		return []outbox.Email{}, err
	}
	defer rows.Close()

	var emails []outbox.Email
	for rows.Next() {
		var e outbox.Email
		var emlFilePath, payloadFilePath, reason sql.NullString
		var updatedAt time.Time

		err := rows.Scan(
			&e.Id,
			&e.Status,
			&emlFilePath,
			&payloadFilePath,
			&reason,
			&e.Version,
			&updatedAt,
		)
		if err != nil {
			return []outbox.Email{}, err
		}

		e.EmlFilePath = emlFilePath.String
		e.PayloadFilePath = payloadFilePath.String
		e.Reason = reason.String
		e.UpdatedAt = updatedAt.Format(time.RFC3339)

		emails = append(emails, e)
	}

	if err = rows.Err(); err != nil {
		return []outbox.Email{}, err
	}

	return emails, nil
}

// Update changes the status of an email using optimistic locking based on version.
// It determines the expected "from" status based on the target "to" status.
// The operation is executed within a transaction with retry logic for transient errors.
// Note: ttl parameter is ignored for MySQL (TTL is a DynamoDB-specific feature).
func (o *Outbox) Update(ctx context.Context, id string, status string, errorReason string, _ *int64) error {
	fromStatus := getExpectedFromStatus(status)

	updateQuery := `
		UPDATE emails
		SET status = ?, reason = ?, version = version + 1
		WHERE id = ? AND status = ?
	`
	historyQuery := `
		INSERT INTO email_statuses (email_id, status, reason)
		VALUES (?, ?, ?)
	`

	var err error
	for attempt := range maxAttempts {
		err = o.executeInTransaction(ctx, func(tx *sql.Tx) error {
			result, execErr := tx.ExecContext(ctx, updateQuery, status, errorReason, id, fromStatus)
			if execErr != nil {
				return execErr
			}

			affected, affErr := result.RowsAffected()
			if affErr != nil {
				return affErr
			}

			if affected == 0 {
				return ErrLockNotAcquired
			}

			_, histErr := tx.ExecContext(ctx, historyQuery, id, status, errorReason)
			return histErr
		})

		if err == nil || !o.shouldRetryMySQL(err) {
			return err
		}

		sleep := o.backoffDuration(attempt)
		timer := time.NewTimer(sleep)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	return err
}

// Ready updates the email to READY status with the eml file path.
// Expected from status is INTAKING.
// The operation is executed within a transaction with retry logic for transient errors.
// Note: ttl parameter is ignored for MySQL (TTL is a DynamoDB-specific feature).
func (o *Outbox) Ready(ctx context.Context, id string, emlFilePath string, _ *int64) error {
	updateQuery := `
		UPDATE emails
		SET status = ?, eml_file_path = ?, version = version + 1
		WHERE id = ? AND status = ?
	`
	historyQuery := `
		INSERT INTO email_statuses (email_id, status, reason)
		VALUES (?, ?, ?)
	`

	var err error
	for attempt := range maxAttempts {
		err = o.executeInTransaction(ctx, func(tx *sql.Tx) error {
			result, execErr := tx.ExecContext(ctx, updateQuery, outbox.StatusReady, emlFilePath, id, outbox.StatusIntaking)
			if execErr != nil {
				return execErr
			}

			affected, affErr := result.RowsAffected()
			if affErr != nil {
				return affErr
			}

			if affected == 0 {
				return ErrLockNotAcquired
			}

			_, histErr := tx.ExecContext(ctx, historyQuery, id, outbox.StatusReady, "")
			return histErr
		})

		if err == nil || !o.shouldRetryMySQL(err) {
			return err
		}

		sleep := o.backoffDuration(attempt)
		timer := time.NewTimer(sleep)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	return err
}

// getExpectedFromStatus returns the expected previous status for a given target status.
// This maps the state machine transitions.
func getExpectedFromStatus(toStatus string) string {
	transitions := map[string]string{
		outbox.StatusIntaking:              outbox.StatusAccepted,
		outbox.StatusReady:                 outbox.StatusIntaking,
		outbox.StatusProcessing:            outbox.StatusReady,
		outbox.StatusSent:                  outbox.StatusProcessing,
		outbox.StatusFailed:                outbox.StatusProcessing,
		outbox.StatusInvalid:               outbox.StatusIntaking,
		outbox.StatusCallingSentCallback:   outbox.StatusSent,
		outbox.StatusCallingFailedCallback: outbox.StatusFailed,
		outbox.StatusSentAcknowledged:      outbox.StatusCallingSentCallback,
		outbox.StatusFailedAcknowledged:    outbox.StatusCallingFailedCallback,
	}

	if from, ok := transitions[toStatus]; ok {
		return from
	}
	return ""
}

// Create inserts a new email into the database (used for testing and by producer)
// The operation is executed within a transaction with retry logic for transient errors.
func (o *Outbox) Create(ctx context.Context, id string, status string, payloadFilePath string) error {
	emailQuery := `
		INSERT INTO emails (id, status, payload_file_path)
		VALUES (?, ?, ?)
	`
	historyQuery := `
		INSERT INTO email_statuses (email_id, status, reason)
		VALUES (?, ?, ?)
	`

	var err error
	for attempt := range maxAttempts {
		err = o.executeInTransaction(ctx, func(tx *sql.Tx) error {
			if _, execErr := tx.ExecContext(ctx, emailQuery, id, status, payloadFilePath); execErr != nil {
				return execErr
			}

			_, histErr := tx.ExecContext(ctx, historyQuery, id, status, "")
			return histErr
		})

		if err == nil || !o.shouldRetryMySQL(err) {
			return err
		}

		sleep := o.backoffDuration(attempt)
		timer := time.NewTimer(sleep)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	return err
}

// Delete removes an email from the database (used for testing cleanup)
func (o *Outbox) Delete(ctx context.Context, id string) error {
	// History will be deleted via CASCADE
	query := `DELETE FROM emails WHERE id = ?`
	_, err := o.db.ExecContext(ctx, query, id)
	return err
}
