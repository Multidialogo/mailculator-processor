package facades

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"io"
	"os"
	"strings"
)

const (
	tableName  = "Outbox"
	statusMeta = "_META"
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
	db *dynamodb.Client
}

func NewOutboxFacade() *OutboxFacade {
	cfg := NewAwsConfigFromEnv()
	return &OutboxFacade{db: dynamodb.NewFromConfig(cfg)}
}

func (of *OutboxFacade) seeder(emlFilePath string) map[string]any {
	id := uuid.NewString()
	return map[string]any{
		"Id":              id,
		"Status":          "READY",
		"EmlFilePath":     emlFilePath,
		"SuccessCallback": fmt.Sprintf("curl -X /success/%s", id),
		"FailureCallback": fmt.Sprintf("curl -X /failure/%s", id),
	}
}

func (of *OutboxFacade) getMetaAttributes(email map[string]any) map[string]any {
	return map[string]any{
		"Latest":          email["Status"],
		"EMLFilePath":     email["EmlFilePath"],
		"SuccessCallback": email["SuccessCallback"],
		"FailureCallback": email["FailureCallback"],
	}
}

func (of *OutboxFacade) AddEmail(ctx context.Context, emlFilePath string) (string, error) {
	if emlFilePath == "" {
		emlFilePath = "testdata/smol.EML"
	}
	metaStmt := fmt.Sprintf("INSERT INTO \"%v\" VALUE {'Id': ?, 'Status': ?, 'Attributes': ?}", tableName)
	email := of.seeder(emlFilePath)
	metaAttrs := of.getMetaAttributes(email)
	metaParams, err := attributevalue.MarshalList([]any{
		email["Id"], statusMeta, metaAttrs,
	})
	if err != nil {
		return "", err
	}

	inStmt := fmt.Sprintf("INSERT INTO \"%v\" VALUE {'Id': ?, 'Status': ?}", tableName)
	inParams, err := attributevalue.MarshalList([]any{
		email["Id"], email["Status"], map[string]any{},
	})
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
	return fmt.Sprint(email["Id"]), err
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
