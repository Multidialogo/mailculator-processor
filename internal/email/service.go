package email

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

const (
	tableName     = "Outbox"
	lockTableName = "OutboxLock"
)

type Service struct {
	db *dynamodb.Client
}

func NewService(db *dynamodb.Client) *Service {
	return &Service{db: db}
}

func (s *Service) FindReady(ctx context.Context) ([]Email, error) {
	query := fmt.Sprintf("SELECT id, attributes FROM \"%v\" WHERE attributes.status = 'READY'", tableName)

	stmt := &dynamodb.ExecuteStatementInput{
		Statement: aws.String(query),
	}

	res, err := s.db.ExecuteStatement(ctx, stmt)
	if err != nil {
		return []Email{}, err
	}

	marshaller := &emailMarshaller{}
	emails, err := marshaller.UnmarshalListOfMaps(res.Items)
	if err != nil {
		return []Email{}, err
	}

	return emails, nil
}

func (s *Service) UpdateStatus(ctx context.Context, id string, status string) error {
	query := fmt.Sprintf("UPDATE \"%v\" SET attributes.status=? WHERE id=?", tableName)

	params, err := attributevalue.MarshalList([]interface{}{status, id})
	if err != nil {
		return err
	}

	stmt := &dynamodb.ExecuteStatementInput{
		Statement:  aws.String(query),
		Parameters: params,
	}

	_, err = s.db.ExecuteStatement(ctx, stmt)
	return err
}

func (s *Service) BatchLock(ctx context.Context, emails []Email) ([]Email, error) {
	query := fmt.Sprintf("INSERT INTO \"%v\" VALUE {'id': ?}", lockTableName)

	var acquired []Email
	for _, email := range emails {
		params, err := attributevalue.MarshalList([]interface{}{email.Id})
		if err != nil {
			return []Email{}, err
		}

		stmt := &dynamodb.ExecuteStatementInput{
			Statement:  aws.String(query),
			Parameters: params,
		}

		_, err = s.db.ExecuteStatement(ctx, stmt)
		if err != nil {
			return []Email{}, err
		}

		acquired = append(acquired, email)
	}

	return acquired, nil
}
