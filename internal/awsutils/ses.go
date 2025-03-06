package awsutils

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

type SesEmailClient struct {
	client *ses.Client
}

func NewSesEmailClient(client *ses.Client) *SesEmailClient {
	return &SesEmailClient{
		client: client,
	}
}

func (c *SesEmailClient) Send(ctx context.Context, raw []byte) error {
	sesInput := &ses.SendRawEmailInput{
		RawMessage: &types.RawMessage{
			Data: raw,
		},
	}

	_, err := c.client.SendRawEmail(ctx, sesInput)
	return err
}
