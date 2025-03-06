package awsutils

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type hasId interface {
	GetId() string
}

type marshaller[T hasId] interface {
	Marshal(T) (map[string]types.AttributeValue, error)
	Unmarshal(map[string]types.AttributeValue) (T, error)
	UnmarshalListOfMaps([]map[string]types.AttributeValue) ([]T, error)
}

type DynamoDbBatchInserter[T hasId] struct {
	client     *dynamodb.Client
	marshaller marshaller[T]
	tableName  string
}

func NewDynamoDbBatchInserter[T hasId](client *dynamodb.Client, marshaller marshaller[T], tableName string) *DynamoDbBatchInserter[T] {
	return &DynamoDbBatchInserter[T]{
		client:     client,
		marshaller: marshaller,
		tableName:  tableName,
	}
}

func (facade *DynamoDbBatchInserter[T]) BatchInsert(ctx context.Context, sourceElements []T) ([]T, error) {
	batchSize := 25 // DynamoDB allows a maximum batch size of 25 items.
	start := 0
	end := start + batchSize

	var unprocessedElements []T

	for start < batchSize && start < len(sourceElements) {
		if end > len(sourceElements) {
			end = len(sourceElements)
		}

		batchElements := sourceElements[start:end]
		var writeReqs []types.WriteRequest

		for _, batchElement := range batchElements {
			marshalledItem, err := facade.marshaller.Marshal(batchElement)
			if err != nil {
				return nil, err
			}

			writeReqs = append(
				writeReqs,
				types.WriteRequest{PutRequest: &types.PutRequest{Item: marshalledItem}},
			)
		}

		writeRequestInput := map[string][]types.WriteRequest{facade.tableName: writeReqs}

		writeResponse, err := facade.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{RequestItems: writeRequestInput})
		if err != nil {
			return nil, err
		}

		unprocessedItems, hasUnprocessedItems := writeResponse.UnprocessedItems[facade.tableName]
		if hasUnprocessedItems {
			for _, unprocessedItem := range unprocessedItems {
				unprocessedElement, err := facade.marshaller.Unmarshal(unprocessedItem.PutRequest.Item)
				if err != nil {
					return nil, err
				}
				unprocessedElements = append(unprocessedElements, unprocessedElement)
			}
		}

		start = end
		end += batchSize
	}

	var processedElements []T

	for _, sourceElement := range sourceElements {
		// I do this because go mod tidy cannot fucking import slices for reasons
		processed := true
		for _, unprocessedElement := range unprocessedElements {
			if unprocessedElement.GetId() == sourceElement.GetId() {
				processed = false
				continue
			}
		}

		if !processed {
			processedElements = append(processedElements, sourceElement)
		}
	}

	return processedElements, nil
}

type DynamoDbScanner[T hasId] struct {
	client     *dynamodb.Client
	marshaller marshaller[T]
	tableName  string
}

func NewDynamoDbScanner[T hasId](client *dynamodb.Client, marshaller marshaller[T], tableName string) *DynamoDbScanner[T] {
	return &DynamoDbScanner[T]{
		client:     client,
		marshaller: marshaller,
		tableName:  tableName,
	}
}

func (facade *DynamoDbScanner[T]) Scan(ctx context.Context, filterBuilder expression.ConditionBuilder, projectionBuilder expression.ProjectionBuilder) ([]T, error) {
	expr, err := expression.NewBuilder().
		WithFilter(filterBuilder).
		WithProjection(projectionBuilder).
		Build()

	if err != nil {
		return nil, err
	}

	scanPaginator := dynamodb.NewScanPaginator(facade.client, &dynamodb.ScanInput{
		TableName:                 aws.String(facade.tableName),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
	})

	var items []T

	for scanPaginator.HasMorePages() {
		response, err := scanPaginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		pageItems, err := facade.marshaller.UnmarshalListOfMaps(response.Items)
		if err != nil {
			return nil, err
		}

		items = append(items, pageItems...)
	}

	return items, nil
}
