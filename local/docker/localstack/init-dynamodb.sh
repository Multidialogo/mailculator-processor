#!/bin/sh

awslocal dynamodb create-table \
    --table-name Outbox \
    --attribute-definitions \
       AttributeName=Id,AttributeType=S \
       AttributeName=Status,AttributeType=S \
    --key-schema \
      AttributeName=Id,KeyType=HASH \
      AttributeName=Status,KeyType=RANGE \
    --billing-mode PAY_PER_REQUEST \
    --global-secondary-indexes \
        "[
            {
                \"IndexName\": \"StatusIndex\",
                \"KeySchema\": [{\"AttributeName\":\"Status\",\"KeyType\":\"HASH\"},
                                {\"AttributeName\":\"Id\",\"KeyType\":\"RANGE\"}],
                \"Projection\":{
                    \"ProjectionType\":\"ALL\"
                }
            }
        ]"
