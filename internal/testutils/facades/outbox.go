package facades

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

func NewAwsConfigFromEnv() aws.Config {
	return aws.Config{
		Region: os.Getenv("AWS_REGION"),
		Credentials: credentials.NewStaticCredentialsProvider(
			os.Getenv("AWS_ACCESS_KEY_ID"),
			os.Getenv("AWS_SECRET_ACCESS_KEY"),
			"",
		),
		BaseEndpoint: aws.String(os.Getenv("AWS_BASE_ENDPOINT")),
	}
}

type OutboxFacade struct {
	db         *dynamodb.Client
	tableName  string
	statusMeta string
}

func NewOutboxFacade(tableName string, statusMeta string) (*OutboxFacade, error) {
	var err error = nil
	if tableName == "" {
		err = fmt.Errorf("table name is required")
	}
	if statusMeta == "" {
		err = fmt.Errorf("status meta is required")
	}
	cfg := NewAwsConfigFromEnv()
	return &OutboxFacade{
		db:         dynamodb.NewFromConfig(cfg),
		tableName:  tableName,
		statusMeta: statusMeta,
	}, err
}

func (of *OutboxFacade) AddEmail(ctx context.Context, emlFilePath string) (string, error) {
	if emlFilePath == "" {
		emlFilePath = "testdata/smol.EML"
	}
	metaStmt := fmt.Sprintf("INSERT INTO \"%v\" VALUE {'Id': ?, 'Status': ?, 'Attributes': ?}", of.tableName)
	id := uuid.NewString()
	status := "READY"
	ttl := time.Now().Add(1 * time.Hour).Unix()
	metaParams, err := attributevalue.MarshalList([]any{
		id,
		of.statusMeta,
		map[string]any{
			"Latest":      status,
			"CreatedAt":   time.Now().Format(time.RFC3339),
			"EMLFilePath": emlFilePath,
			"TTL":         ttl,
		},
	})
	if err != nil {
		return "", err
	}

	inStmt := fmt.Sprintf("INSERT INTO \"%v\" VALUE {'Id': ?, 'Status': ?, 'Attributes': ?}", of.tableName)
	inParams, err := attributevalue.MarshalList([]any{id, status, map[string]any{"TTL": ttl}})
	if err != nil {
		return "", err
	}

	ti := &dynamodb.ExecuteTransactionInput{
		TransactStatements: []types.ParameterizedStatement{
			{Statement: aws.String(metaStmt), Parameters: metaParams},
			{Statement: aws.String(inStmt), Parameters: inParams},
		},
	}

	_, err = of.db.ExecuteTransaction(ctx, ti)
	return id, err
}

func (of *OutboxFacade) AddEmlFile(destDirPath string) (string, error) {
	srcFile, err := os.Open("testdata/sample.eml")
	if err != nil {
		return "", err
	}
	defer srcFile.Close()

	if destDirPath == "" {
		destDirPath = "/"
	}
	if !strings.HasSuffix(destDirPath, "/") {
		destDirPath += "/"
	}

	destFilePath := fmt.Sprintf("%s%s.eml", destDirPath, uuid.NewString())

	destFile, err := os.Create(destFilePath)
	if err != nil {
		return destFilePath, err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return destFilePath, err
}

func (of *OutboxFacade) DeleteEmail(ctx context.Context, id string) error {
	var err error = nil
	if id == "" {
		err = fmt.Errorf("id is required")
		return err
	}
	query := fmt.Sprintf("SELECT Status FROM \"%v\" WHERE Id=?", of.tableName)
	params, _ := attributevalue.MarshalList([]any{id})
	stmt := &dynamodb.ExecuteStatementInput{Statement: aws.String(query), Parameters: params}

	res, err := of.db.ExecuteStatement(ctx, stmt)
	if res != nil {
		query = fmt.Sprintf("DELETE FROM \"%v\" WHERE Id=? AND Status=?", of.tableName)
		for _, item := range res.Items {
			params, _ = attributevalue.MarshalList([]any{
				id,
				item["Status"].(*types.AttributeValueMemberS).Value,
			})
			stmt = &dynamodb.ExecuteStatementInput{Statement: aws.String(query), Parameters: params}
			_, err = of.db.ExecuteStatement(ctx, stmt)
			if err != nil {
				log.Println(err)
			}
		}
	}

	return err
}
