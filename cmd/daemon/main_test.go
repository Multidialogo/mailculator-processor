package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"mailculator-processor/internal/outbox"
	"mailculator-processor/internal/testutils/facades"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_Main_WhenSigtermSignal_WillGracefullyShutdown(t *testing.T) {
	runFn = func(ctx context.Context) {
		time.Sleep(200 * time.Millisecond)
	}

	var sendSignalError error
	go func() {
		time.Sleep(100 * time.Millisecond)
		p, sendSignalError := os.FindProcess(os.Getpid())
		if sendSignalError != nil {
			return
		}
		sendSignalError = p.Signal(syscall.SIGTERM)
	}()

	require.NotPanics(t, main)
	require.Nilf(t, sendSignalError, "failed to send signal: %v", sendSignalError)
}

func TestMainComplete(t *testing.T) {
	oFacade, err := facades.NewOutboxFacade(outbox.TableName, outbox.StatusMeta)
	require.NoError(t, err)

	fixtures := make([]string, 0)

	dir := t.TempDir()
	for i := 0; i < 5; i++ {
		filePath, _ := oFacade.AddEmlFile(dir)
		emailId, _ := oFacade.AddEmail(context.TODO(), filePath)
		fixtures = append(fixtures, emailId)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	run(ctx)

	awsConfig := facades.NewAwsConfigFromEnv()
	db := dynamodb.NewFromConfig(awsConfig)
	for _, value := range fixtures {
		query := fmt.Sprintf("SELECT Attributes.Latest FROM \"%v\" WHERE Id=? AND Status=?", outbox.TableName)
		params, _ := attributevalue.MarshalList([]interface{}{value, outbox.StatusMeta})
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
