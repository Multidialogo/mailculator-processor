package outbox

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"
)

const (
	StatusReady                 = "READY"
	StatusProcessing            = "PROCESSING"
	StatusSent                  = "SENT"
	StatusFailed                = "FAILED"
	StatusCallingSentCallback   = "CALLING-SENT-CALLBACK"
	StatusCallingFailedCallback = "CALLING-FAILED-CALLBACK"
	StatusSentAcknowledged      = "SENT-ACKNOWLEDGED"
	StatusFailedAcknowledged    = "FAILED-ACKNOWLEDGED"
)

const (
	statusIndex = "StatusIndex"
	StatusMeta  = "_META"
)

const (
	maxAttempts = 8
	baseDelay   = 30 * time.Millisecond
	maxDelay    = 1 * time.Second
)

type Email struct {
	Id          string
	Status      string
	EmlFilePath string
	UpdatedAt   string
	Reason      string
	TTL         int64
}

type dynamodbInterface interface {
	ExecuteStatement(context.Context, *dynamodb.ExecuteStatementInput, ...func(options *dynamodb.Options)) (*dynamodb.ExecuteStatementOutput, error)
	ExecuteTransaction(context.Context, *dynamodb.ExecuteTransactionInput, ...func(options *dynamodb.Options)) (*dynamodb.ExecuteTransactionOutput, error)
}

type Outbox struct {
	db        dynamodbInterface
	tableName string
}

func NewOutbox(db *dynamodb.Client, tableName string) *Outbox {
	return &Outbox{
		db:        db,
		tableName: tableName,
	}
}

func (o *Outbox) Query(ctx context.Context, status string, limit int) ([]Email, error) {
	query := fmt.Sprintf("SELECT Id, Status, Attributes FROM \"%v\".\"%v\" WHERE Status=? AND Attributes.Latest =?", o.tableName, statusIndex)
	params, _ := attributevalue.MarshalList([]any{StatusMeta, status})

	var items []map[string]types.AttributeValue
	var nextToken *string
	done := false

	for !done {
		stmt := &dynamodb.ExecuteStatementInput{
			Parameters: params,
			Statement:  aws.String(query),
			NextToken:  nextToken,
		}

		res, err := o.db.ExecuteStatement(ctx, stmt)
		if err != nil {
			return []Email{}, err
		}

		items = append(items, res.Items...)

		if len(items) > limit {
			items = append([]map[string]types.AttributeValue{}, items[:limit]...)
			done = true
			break
		}

		if res.NextToken != nil {
			nextToken = res.NextToken
		} else {
			done = true
		}
	}

	return new(emailMarshaller).UnmarshalList(items)
}

func (o *Outbox) shouldRetryPartiQL(err error) bool {
	var tce *types.TransactionCanceledException
	if errors.As(err, &tce) {
		for _, r := range tce.CancellationReasons {
			if r.Code == nil {
				// unknown reason -> conservative: retry
				return true
			}

			code := *r.Code
			if code == "TransactionConflict" {
				return true
			}
		}

		return false
	}

	var ptee *types.ProvisionedThroughputExceededException
	if errors.As(err, &ptee) {
		return true
	}

	var ise *types.InternalServerError
	if errors.As(err, &ise) {
		return true
	}

	var riue *types.ResourceInUseException
	if errors.As(err, &riue) {
		return true
	}

	var rle *types.RequestLimitExceeded
	if errors.As(err, &rle) {
		return true
	}

	var tipe *types.TransactionInProgressException
	if errors.As(err, &tipe) {
		return true
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "ThrottlingException", "Throttling", "RequestLimitExceeded", "ServiceUnavailable":
			return true
		}
	}

	return false
}

func (o *Outbox) backoffDuration(attempt int) time.Duration {
	max := min(time.Duration(1<<uint(attempt))*baseDelay, maxDelay)
	if max <= 0 {
		max = baseDelay
	}

	return time.Duration(rand.Int63n(int64(max)))
}

func (o *Outbox) Update(ctx context.Context, id string, status string, errorReason string, ttl int64) error {
	metaStmt := fmt.Sprintf("UPDATE \"%v\" SET Attributes.Latest=?, Attributes.UpdatedAt=?, Attributes.Reason=? WHERE Id=? AND Status=?", o.tableName)
	metaParams, _ := attributevalue.MarshalList([]any{
		status,
		time.Now().Format(time.RFC3339),
		errorReason,
		id,
		StatusMeta,
	})

	inStmt := fmt.Sprintf("INSERT INTO \"%v\" VALUE {'Id': ?, 'Status': ?, 'Attributes': ?}", o.tableName)
	inParams, _ := attributevalue.MarshalList([]any{id, status, map[string]any{"TTL": ttl}})

	var err error
	for attempt := range maxAttempts {
		ti := &dynamodb.ExecuteTransactionInput{
			TransactStatements: []types.ParameterizedStatement{
				{Statement: aws.String(metaStmt), Parameters: metaParams},
				{Statement: aws.String(inStmt), Parameters: inParams},
			},
		}

		_, err = o.db.ExecuteTransaction(ctx, ti)
		if err == nil || !o.shouldRetryPartiQL(err) {
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

type emailItemRow struct {
	Id         string         `dynamodbav:"Id"`
	Status     string         `dynamodbav:"Status"`
	Attributes map[string]any `dynamodbav:"Attributes"`
}

type emailMarshaller struct{}

func (m *emailMarshaller) unmarshalTTL(value any) (int64, error) {
	if value == nil {
		return 0, nil
	}

	switch v := value.(type) {
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	case int:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("TTL must be an integer, got %T", value)
	}
}

func (m *emailMarshaller) UnmarshalList(attrsList []map[string]types.AttributeValue) (emails []Email, err error) {
	var items []emailItemRow
	err = attributevalue.UnmarshalListOfMaps(attrsList, &items)
	if err != nil {
		return []Email{}, err
	}

	for _, item := range items {
		ttl, err := m.unmarshalTTL(item.Attributes["TTL"])
		if err != nil {
			return []Email{}, fmt.Errorf("error unmarshalling TTL for email %s: %w", item.Id, err)
		}

		emails = append(emails, Email{
			Id:          item.Id,
			Status:      fmt.Sprint(item.Attributes["Latest"]),
			EmlFilePath: fmt.Sprint(item.Attributes["EMLFilePath"]),
			UpdatedAt:   fmt.Sprint(item.Attributes["UpdatedAt"]),
			Reason:      fmt.Sprint(item.Attributes["Reason"]),
			TTL:         ttl,
		})
	}

	return emails, nil
}
