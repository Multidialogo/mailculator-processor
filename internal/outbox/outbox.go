package outbox

import (
	"context"
	"fmt"
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
	tableName   = "Outbox"
	statusIndex = "StatusIndex"
	statusMeta  = "_META"
)

type Email struct {
	Id              string
	Status          string
	EmlFilePath     string
	SuccessCallback string
	FailureCallback string
}

type Outbox struct {
	db *dynamodb.Client
}

func NewOutbox(db *dynamodb.Client) *Outbox {
	return &Outbox{db: db}
}

func (o *Outbox) Query(ctx context.Context, status string, limit int) ([]Email, error) {
	query := fmt.Sprintf("SELECT Id, Status, Attributes FROM \"%v\".\"%v\" WHERE Status=? AND Attributes.Latest =?", tableName, statusIndex)
	params, err := attributevalue.MarshalList([]interface{}{statusMeta, status})
	if err != nil {
		return []Email{}, err
	}

	stmt := &dynamodb.ExecuteStatementInput{
		Parameters: params,
		Statement:  aws.String(query),
		Limit:      aws.Int32(int32(limit)),
	}

	res, err := o.db.ExecuteStatement(ctx, stmt)
	if err != nil {
		return []Email{}, err
	}

	return new(emailMarshaller).UnmarshalList(res.Items)
}

func (o *Outbox) Insert(ctx context.Context, email Email) error {
	metaStmt := fmt.Sprintf("INSERT INTO \"%v\" VALUE {'Id': ?, 'Status': ?, 'Attributes': ?}", tableName)
	metaAttrs := new(emailMarshaller).GetMetaAttributes(email)
	metaParams, err := attributevalue.MarshalList([]interface{}{email.Id, statusMeta, metaAttrs})
	if err != nil {
		return err
	}

	inStmt := fmt.Sprintf("INSERT INTO \"%v\" VALUE {'Id': ?, 'Status': ?}", tableName)
	inParams, err := attributevalue.MarshalList([]interface{}{email.Id, email.Status, map[string]interface{}{}})
	if err != nil {
		return err
	}

	ti := &dynamodb.ExecuteTransactionInput{
		TransactStatements: []types.ParameterizedStatement{
			{Statement: aws.String(metaStmt), Parameters: metaParams},
			{Statement: aws.String(inStmt), Parameters: inParams},
		},
	}

	_, err = o.db.ExecuteTransaction(ctx, ti)
	return err
}

func (o *Outbox) Update(ctx context.Context, id string, status string) error {
	metaStmt := fmt.Sprintf("UPDATE \"%v\" SET Attributes.Latest=? WHERE Id=? AND Status=?", tableName)
	metaParams, err := attributevalue.MarshalList([]interface{}{status, id, statusMeta})
	if err != nil {
		return err
	}

	inStmt := fmt.Sprintf("INSERT INTO \"%v\" VALUE {'Id': ?, 'Status': ?, 'Attributes': ?}", tableName)
	inParams, err := attributevalue.MarshalList([]interface{}{id, status, map[string]interface{}{}})
	if err != nil {
		return err
	}

	ti := &dynamodb.ExecuteTransactionInput{
		TransactStatements: []types.ParameterizedStatement{
			{Statement: aws.String(metaStmt), Parameters: metaParams},
			{Statement: aws.String(inStmt), Parameters: inParams},
		},
	}

	_, err = o.db.ExecuteTransaction(ctx, ti)
	return err
}

type emailItemRow struct {
	Id         string                 `dynamodbav:"Id"`
	Status     string                 `dynamodbav:"Status"`
	Attributes map[string]interface{} `dynamodbav:"Attributes"`
}

func (email emailItemRow) GetKey() map[string]types.AttributeValue {
	id, err := attributevalue.Marshal(email.Id)
	if err != nil {
		panic(err)
	}

	status, err := attributevalue.Marshal(email.Status)
	if err != nil {
		panic(err)
	}

	return map[string]types.AttributeValue{"Id": id, "Status": status}
}

type emailMarshaller struct{}

func (m *emailMarshaller) GetMetaAttributes(email Email) map[string]interface{} {
	return map[string]interface{}{
		"Latest":          email.Status,
		"EMLFilePath":     email.EmlFilePath,
		"SuccessCallback": email.SuccessCallback,
		"FailureCallback": email.FailureCallback,
	}
}

func (m *emailMarshaller) UnmarshalList(attrsList []map[string]types.AttributeValue) (emails []Email, err error) {
	var items []emailItemRow
	err = attributevalue.UnmarshalListOfMaps(attrsList, &items)
	if err != nil {
		return []Email{}, err
	}

	for _, item := range items {
		emails = append(emails, Email{
			Id:              item.Id,
			Status:          fmt.Sprint(item.Attributes["Latest"]),
			EmlFilePath:     fmt.Sprint(item.Attributes["EMLFilePath"]),
			SuccessCallback: fmt.Sprint(item.Attributes["SuccessCallback"]),
			FailureCallback: fmt.Sprint(item.Attributes["FailureCallback"]),
		})
	}

	return emails, nil
}
