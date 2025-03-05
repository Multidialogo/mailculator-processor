#!/bin/sh

aws dynamodb create-table \
    --table-name Outbox \
    --attribute-definitions \
       AttributeName=id,AttributeType=S \
    --key-schema AttributeName=id,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST \
    --table-class STANDARD \
    --endpoint-url http://db:8000

aws dynamodb create-table \
    --table-name OutboxLock \
    --attribute-definitions \
       AttributeName=id,AttributeType=S \
    --key-schema AttributeName=id,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST \
    --table-class STANDARD \
    --endpoint-url http://db:8000
