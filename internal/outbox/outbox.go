package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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

type Email struct {
	Id          string
	Status      string
	EmlFilePath string
	UpdatedAt   string
	Reason      string
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

	stmt := &dynamodb.ExecuteStatementInput{
		Parameters: params,
		Statement:  aws.String(query),
	}

	res, err := o.db.ExecuteStatement(ctx, stmt)
	if err != nil {
		return []Email{}, err
	}

	if len(res.Items) > limit {
		res.Items = append([]map[string]types.AttributeValue{}, res.Items[:limit]...)
	}

	return new(emailMarshaller).UnmarshalList(res.Items)
}

func (o *Outbox) Update(ctx context.Context, id string, status string, errorReason string) error {
	metaStmt := fmt.Sprintf("UPDATE \"%v\" SET Attributes.Latest=?, Attributes.UpdatedAt=?, Attributes.Reason=? WHERE Id=? AND Status=?", o.tableName)
	metaParams, _ := attributevalue.MarshalList([]any{
		status,
		time.Now().Format(time.RFC3339),
		errorReason,
		id,
		StatusMeta,
	})

	inStmt := fmt.Sprintf("INSERT INTO \"%v\" VALUE {'Id': ?, 'Status': ?, 'Attributes': ?}", o.tableName)
	inParams, _ := attributevalue.MarshalList([]any{id, status, map[string]any{}})

	ti := &dynamodb.ExecuteTransactionInput{
		TransactStatements: []types.ParameterizedStatement{
			{Statement: aws.String(metaStmt), Parameters: metaParams},
			{Statement: aws.String(inStmt), Parameters: inParams},
		},
	}

	_, err := o.db.ExecuteTransaction(ctx, ti)
	return err
}

type emailItemRow struct {
	Id         string         `dynamodbav:"Id"`
	Status     string         `dynamodbav:"Status"`
	Attributes map[string]any `dynamodbav:"Attributes"`
}

type emailMarshaller struct{}

func (m *emailMarshaller) UnmarshalList(attrsList []map[string]types.AttributeValue) (emails []Email, err error) {
	var items []emailItemRow
	err = attributevalue.UnmarshalListOfMaps(attrsList, &items)
	if err != nil {
		return []Email{}, err
	}

	for _, item := range items {
		emails = append(emails, Email{
			Id:          item.Id,
			Status:      fmt.Sprint(item.Attributes["Latest"]),
			EmlFilePath: fmt.Sprint(item.Attributes["EMLFilePath"]),
			UpdatedAt:   fmt.Sprint(item.Attributes["UpdatedAt"]),
			Reason:      fmt.Sprint(item.Attributes["Reason"]),
		})
	}

	return emails, nil
}
