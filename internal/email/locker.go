package email

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Locker struct {
	dynamoDbClient *dynamodb.Client
	lockTableName  string
}

func NewLocker(dynamoDbClient *dynamodb.Client, lockTableName string) *Locker {
	return &Locker{
		dynamoDbClient: dynamoDbClient,
		lockTableName:  lockTableName,
	}
}

func (l *Locker) BatchLock(ids []string) ([]string, error) {
	batchSize := 25 // DynamoDB allows a maximum batch size of 25 items.
	start := 0
	end := start + batchSize

	var unprocessedIds []string

	for start < batchSize && start < len(ids) {
		if end > len(ids) {
			end = len(ids)
		}

		batchIds := ids[start:end]
		var writeReqs []types.WriteRequest

		for _, id := range batchIds {
			item, err := attributevalue.MarshalMap(Lock{Id: id})
			if err != nil {
				return nil, err
			}

			writeReqs = append(
				writeReqs,
				types.WriteRequest{PutRequest: &types.PutRequest{Item: item}},
			)
		}

		writeRequestInput := map[string][]types.WriteRequest{l.lockTableName: writeReqs}

		writeResponse, err := l.dynamoDbClient.BatchWriteItem(context.TODO(), &dynamodb.BatchWriteItemInput{RequestItems: writeRequestInput})
		if err != nil {
			return nil, err
		}

		unprocessedItems, hasUnprocessedItems := writeResponse.UnprocessedItems[l.lockTableName]
		if hasUnprocessedItems {
			for _, unprocessedItem := range unprocessedItems {
				lockObj := Lock{}
				err = attributevalue.UnmarshalMap(unprocessedItem.PutRequest.Item, &lockObj)
				if err != nil {
					return nil, err
				}

				unprocessedIds = append(unprocessedIds, lockObj.Id)
			}
		}

		start = end
		end += batchSize
	}

	var acquired []string
	for _, id := range ids {
		// I do this because go mod tidy cannot fucking import slices for reasons
		unprocessed := false
		for _, unprocessedId := range unprocessedIds {
			if id == unprocessedId {
				unprocessed = true
				continue
			}
		}

		if !unprocessed {
			acquired = append(acquired, id)
		}
	}

	return acquired, nil
}

func (l *Locker) Unlock(id string) error {
	// TODO implement
	return nil
}
