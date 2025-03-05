package email

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

type unlocker interface {
	Unlock(id string) error
}

type SESSender struct {
	client   *ses.Client
	unlocker unlocker
}

func NewSESSender(cfg aws.Config) *SESSender {
	client := ses.NewFromConfig(cfg)
	return &SESSender{client: client}
}

func (s *SESSender) SendAndUnlock(ctx context.Context, email Email) error {
	sesInput := &ses.SendRawEmailInput{
		RawMessage: &types.RawMessage{
			Data: email.AsRaw(),
		},
	}

	_, err := s.client.SendRawEmail(ctx, sesInput)
	if err != nil {
		return err
	}

	return s.unlocker.Unlock(email.Id)
}
