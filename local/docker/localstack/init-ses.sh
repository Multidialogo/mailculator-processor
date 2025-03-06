#!/bin/sh

awslocal ses verify-email-identity --email-address sender@test.multidialogo.it

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