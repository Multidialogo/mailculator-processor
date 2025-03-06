package email

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"mailculator-processor/internal/awsutils"
)

func redirectPanic(err error) {
	r := recover()
	if r != nil {
		err = errors.New(fmt.Sprint(r))
	}
}

type emailItem struct {
	Id         string                 `dynamodbav:"id"`
	Attributes map[string]interface{} `dynamodbav:"attributes"`
}

func (email emailItem) GetKey() map[string]types.AttributeValue {
	id, err := attributevalue.Marshal(email.Id)
	if err != nil {
		panic(err)
	}

	return map[string]types.AttributeValue{"id": id}
}

type emailMarshaller struct{}

func (m *emailMarshaller) itemToModel(item emailItem) (Email, error) {
	return Email{
		Id:              item.Id,
		Status:          item.Attributes["status"].(string),
		EmlFilePath:     item.Attributes["path"].(string),
		SuccessCallback: item.Attributes["success_callback"].(string),
		FailureCallback: item.Attributes["failure_callback"].(string),
	}, nil
}

func (m *emailMarshaller) Marshal(email Email) (map[string]types.AttributeValue, error) {
	return attributevalue.MarshalMap(
		emailItem{
			Id: email.Id,
			Attributes: map[string]interface{}{
				"status": "status",
			},
		},
	)
}

func (m *emailMarshaller) Unmarshal(attrs map[string]types.AttributeValue) (email Email, err error) {
	defer redirectPanic(err)

	item := emailItem{}
	err = attributevalue.UnmarshalMap(attrs, &item)
	if err != nil {
		return Email{}, err
	}

	return m.itemToModel(item)
}

func (m *emailMarshaller) UnmarshalListOfMaps(attrsList []map[string]types.AttributeValue) (emails []Email, err error) {
	defer redirectPanic(err)

	var items []emailItem
	err = attributevalue.UnmarshalListOfMaps(attrsList, &items)
	if err != nil {
		return []Email{}, err
	}

	for _, item := range items {
		email, err := m.itemToModel(item)
		if err != nil {
			return []Email{}, err
		}

		emails = append(emails, email)
	}

	return emails, nil
}

type DataMapper struct {
	scanner           *awsutils.DynamoDbScanner[Email]
	projectionBuilder expression.ProjectionBuilder
}

func NewDataMapper(client *dynamodb.Client) *DataMapper {
	return &DataMapper{
		scanner: awsutils.NewDynamoDbScanner(client, &emailMarshaller{}, "Outbox"),
		projectionBuilder: expression.NamesList(
			expression.Name("id"),
			expression.Name("attributes.status"),
			expression.Name("attributes.eml_filepath"),
			expression.Name("attributes.success_callback"),
			expression.Name("attributes.failure_callback"),
		),
	}
}

func (m *DataMapper) FindReady(ctx context.Context) ([]Email, error) {
	filterBuilder := expression.Name("attributes.status").Equal(expression.Value("READY"))

	emails, err := m.scanner.Scan(ctx, filterBuilder, m.projectionBuilder)
	if err != nil {
		return []Email{}, err
	}

	return emails, nil
}

type lockItem struct {
	Id string `dynamodbav:"id"`
}

func (lock lockItem) GetKey() map[string]types.AttributeValue {
	id, err := attributevalue.Marshal(lock.Id)
	if err != nil {
		panic(err)
	}

	return map[string]types.AttributeValue{"id": id}
}

type lockMarshaller struct{}

func (m *lockMarshaller) Marshal(lock Lock) (map[string]types.AttributeValue, error) {
	item := lockItem{Id: lock.Id}

	return attributevalue.MarshalMap(item)
}

func (m *lockMarshaller) Unmarshal(item map[string]types.AttributeValue) (lock Lock, err error) {
	defer redirectPanic(err)

	err = attributevalue.UnmarshalMap(item, &lock)
	if err != nil {
		return Lock{}, err
	}

	return Lock{Id: lock.Id}, nil
}

func (m *lockMarshaller) UnmarshalListOfMaps(attrsList []map[string]types.AttributeValue) (locks []Lock, err error) {
	defer redirectPanic(err)

	var items []lockItem
	err = attributevalue.UnmarshalListOfMaps(attrsList, &items)
	if err != nil {
		return []Lock{}, err
	}

	for _, item := range items {
		locks = append(locks, Lock{
			Id: item.Id,
		})
	}

	return locks, err
}

type LockDataMapper struct {
	batchInserter *awsutils.DynamoDbBatchInserter[Lock]
}

func NewLockDataMapper(client *dynamodb.Client) *LockDataMapper {
	return &LockDataMapper{
		batchInserter: awsutils.NewDynamoDbBatchInserter(client, &lockMarshaller{}, "OutboxLock"),
	}
}

func (m *LockDataMapper) BatchInsert(ctx context.Context, locks []Lock) ([]Lock, error) {
	if len(locks) == 0 {
		return []Lock{}, nil
	}

	var items []lockItem
	for _, lock := range locks {
		items = append(items, lockItem{Id: lock.Id})
	}

	return m.batchInserter.BatchInsert(ctx, locks)
}
