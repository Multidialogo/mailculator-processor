package email

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type locker interface {
	BatchLock(ids []string) ([]string, error)
}

type Finder struct {
	locker          locker
	dynamoDbClient  *dynamodb.Client
	outboxTableName string
}

func NewFinder(locker locker, client *dynamodb.Client, outboxTableName string) *Finder {
	return &Finder{
		locker:          locker,
		dynamoDbClient:  client,
		outboxTableName: outboxTableName,
	}
}

func (f *Finder) FindAndLock() ([]Email, error) {
	readyEmails, err := f.findReady()
	if err != nil {
		return nil, err
	}

	lockedEmails, err := f.tryAcquireLocks(readyEmails)
	if err != nil {
		return nil, err
	}

	return lockedEmails, nil
}

func (f *Finder) findReady() ([]Email, error) {
	filterExpr := expression.Name("attributes.status").Equal(expression.Value("READY"))

	projectionExpr := expression.NamesList(
		expression.Name("id"),
		expression.Name("attributes.status"),
		expression.Name("attributes.eml_filepath"),
		expression.Name("attributes.success_callback"),
		expression.Name("attributes.failure_callback"),
	)

	expr, err := expression.NewBuilder().WithFilter(filterExpr).WithProjection(projectionExpr).Build()
	if err != nil {
		return nil, err
	}

	scanPaginator := dynamodb.NewScanPaginator(f.dynamoDbClient, &dynamodb.ScanInput{
		TableName:                 aws.String(f.outboxTableName),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
	})

	var emails []Email

	for scanPaginator.HasMorePages() {
		response, err := scanPaginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}

		var email []Email
		err = attributevalue.UnmarshalListOfMaps(response.Items, &email)
		if err != nil {
			return nil, err
		}

		emails = append(emails, email...)
	}

	return emails, nil
}

func (f *Finder) tryAcquireLocks(emails []Email) ([]Email, error) {
	var attemptedIds []string
	for _, email := range emails {
		attemptedIds = append(attemptedIds, email.Id)
	}

	lockedIds, err := f.locker.BatchLock(attemptedIds)
	if err != nil {
		return nil, err
	}

	var lockedEmails []Email
	for _, email := range emails {
		// I do this because go mod tidy cannot fucking import slices
		locked := false
		for _, lockedId := range lockedIds {
			if email.Id == lockedId {
				locked = true
				continue
			}
		}

		if locked {
			lockedEmails = append(lockedEmails, email)
		}
	}

	return lockedEmails, nil
}
