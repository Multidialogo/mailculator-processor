package email

import (
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Email struct {
	Id         string                 `dynamodbav:"id"`
	Attributes map[string]interface{} `dynamodbav:"attributes"`
}

func (email Email) GetKey() map[string]types.AttributeValue {
	id, err := attributevalue.Marshal(email.Id)
	if err != nil {
		panic(err)
	}
	return map[string]types.AttributeValue{"id": id}
}

func (email Email) AsRaw() []byte {
	return nil
}

type Lock struct {
	Id string `dynamodbav:"id"`
}

func (lock Lock) GetKey() map[string]types.AttributeValue {
	id, err := attributevalue.Marshal(lock.Id)
	if err != nil {
		panic(err)
	}
	return map[string]types.AttributeValue{"id": id}
}
