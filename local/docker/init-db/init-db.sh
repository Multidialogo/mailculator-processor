#!/bin/sh

aws dynamodb create-table \
    --table-name Outbox \
    --attribute-definitions \
       AttributeName=MessageId,AttributeType=S \
    --key-schema AttributeName=MessageId,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST \
    --table-class STANDARD \
    --endpoint-url http://db:8000
