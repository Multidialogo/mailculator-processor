package facades

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

type MySQLOutboxFacade struct {
	db *sql.DB
}

func NewMySQLConfigFromEnv() string {
	host := os.Getenv("MYSQL_HOST")
	port := os.Getenv("MYSQL_PORT")
	user := os.Getenv("MYSQL_USER")
	password := os.Getenv("MYSQL_PASSWORD")
	database := os.Getenv("MYSQL_DATABASE")
	tls := os.Getenv("MYSQL_TLS")

	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "3306"
	}
	if user == "" {
		user = "root"
	}
	if password == "" {
		password = "test"
	}
	if database == "" {
		database = "mailculator_test"
	}
	if tls == "" {
		tls = "true"
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&tls=%s", user, password, host, port, database, tls)
}

func NewMySQLOutboxFacade() (*MySQLOutboxFacade, error) {
	dsn := NewMySQLConfigFromEnv()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql connection: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping mysql: %w", err)
	}

	return &MySQLOutboxFacade{db: db}, nil
}

func (f *MySQLOutboxFacade) GetDB() *sql.DB {
	return f.db
}

func (f *MySQLOutboxFacade) Close() error {
	return f.db.Close()
}

func (f *MySQLOutboxFacade) AddEmail(ctx context.Context, payloadFilePath string) (string, error) {
	if payloadFilePath == "" {
		payloadFilePath = "/path/to/payload.json"
	}

	id := uuid.NewString()
	status := "READY"

	query := `
		INSERT INTO emails (id, status, payload_file_path)
		VALUES (?, ?, ?)
	`

	_, err := f.db.ExecContext(ctx, query, id, status, payloadFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to insert email: %w", err)
	}

	// Insert initial history
	historyQuery := `
		INSERT INTO email_statuses (email_id, status, reason)
		VALUES (?, ?, ?)
	`
	_, _ = f.db.ExecContext(ctx, historyQuery, id, status, "test fixture")

	return id, nil
}

func (f *MySQLOutboxFacade) AddEmailWithStatus(ctx context.Context, status string, payloadFilePath string) (string, error) {
	if payloadFilePath == "" {
		payloadFilePath = "/path/to/payload.json"
	}

	id := uuid.NewString()

	query := `
		INSERT INTO emails (id, status, payload_file_path)
		VALUES (?, ?, ?)
	`

	_, err := f.db.ExecContext(ctx, query, id, status, payloadFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to insert email: %w", err)
	}

	// Insert initial history
	historyQuery := `
		INSERT INTO email_statuses (email_id, status, reason)
		VALUES (?, ?, ?)
	`
	_, _ = f.db.ExecContext(ctx, historyQuery, id, status, "test fixture")

	return id, nil
}

func (f *MySQLOutboxFacade) AddEmailWithPayload(ctx context.Context, status string, payloadFilePath string) (string, error) {
	if payloadFilePath == "" {
		return "", fmt.Errorf("payload file path is required")
	}

	id := uuid.NewString()

	query := `
		INSERT INTO emails (id, status, payload_file_path)
		VALUES (?, ?, ?)
	`

	_, err := f.db.ExecContext(ctx, query, id, status, payloadFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to insert email: %w", err)
	}

	historyQuery := `
		INSERT INTO email_statuses (email_id, status, reason)
		VALUES (?, ?, ?)
	`
	_, _ = f.db.ExecContext(ctx, historyQuery, id, status, "test fixture")

	return id, nil
}

func (f *MySQLOutboxFacade) DeleteEmail(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}

	// History will be deleted via CASCADE
	query := `DELETE FROM emails WHERE id = ?`
	_, err := f.db.ExecContext(ctx, query, id)
	return err
}

func (f *MySQLOutboxFacade) GetEmailStatus(ctx context.Context, id string) (string, error) {
	var status string
	err := f.db.QueryRowContext(ctx, "SELECT status FROM emails WHERE id = ?", id).Scan(&status)
	return status, err
}

// WaitForReady waits for MariaDB to be ready by checking for the zz-finish marker file
// This is used in tests to wait for MariaDB initialization to complete
func WaitForMySQLReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		facade, err := NewMySQLOutboxFacade()
		if err == nil {
			facade.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for MySQL to be ready")
}
