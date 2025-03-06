#!/bin/sh

awslocal dynamodb create-table \
    --table-name Outbox \
    --attribute-definitions \
       AttributeName=id,AttributeType=S \
    --key-schema AttributeName=id,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST

awslocal dynamodb create-table \
    --table-name OutboxLock \
    --attribute-definitions \
       AttributeName=id,AttributeType=S \
    --key-schema AttributeName=id,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST