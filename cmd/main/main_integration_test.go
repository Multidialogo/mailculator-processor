//go:build integration

package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mailculator-processor/internal/outbox"
	"mailculator-processor/internal/testutils/facades"
)

const outboxTableName = "Outbox"

func TestMainComplete(t *testing.T) {
	oFacade, err := facades.NewOutboxFacade(outboxTableName, outbox.StatusMeta)
	require.NoError(t, err)

	fixtures := make([]string, 0)

	dir := t.TempDir()
	for i := 0; i < 5; i++ {
		filePath, _ := oFacade.AddEmlFile(dir)
		emailId, _ := oFacade.AddEmail(context.TODO(), filePath)
		fixtures = append(fixtures, emailId)
	}

	srv := &http.Server{
		Addr: ":8081",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("Method error: expected POST received %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}),
	}
	go srv.ListenAndServe()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	run(ctx)
	srv.Shutdown(ctx)

	awsConfig := facades.NewAwsConfigFromEnv()
	db := dynamodb.NewFromConfig(awsConfig)
	for _, value := range fixtures {
		query := fmt.Sprintf("SELECT Attributes.Latest FROM \"%v\" WHERE Id=? AND Status=?", outboxTableName)
		params, _ := attributevalue.MarshalList([]any{value, outbox.StatusMeta})
		stmt := &dynamodb.ExecuteStatementInput{Statement: aws.String(query), Parameters: params}
		res, midErr := db.ExecuteStatement(context.TODO(), stmt)
		require.NoError(t, midErr)
		assert.Len(t, res.Items, 1)
		assert.Equal(t, "SENT-ACKNOWLEDGED", res.Items[0]["Latest"].(*types.AttributeValueMemberS).Value)
	}

	// Delete fixtures.
	for _, value := range fixtures {
		errFix := oFacade.DeleteEmail(context.Background(), value)
		if errFix != nil {
			t.Errorf("error while deleting fixture %s, error: %v", value, errFix)
		}
	}
}
