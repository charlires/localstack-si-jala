# create main and dlq queues
awslocal sqs create-queue --queue-name log-queue
awslocal sqs create-queue --queue-name log-queue-dlq

# awslocal sqs get-queue-attributes --queue-url http://localstack:4566/queue/log-queue --attribute-names All

awslocal sqs set-queue-attributes \
    --queue-url http://localstack:4566/queue/log-queue \
    --attributes file:///docker-entrypoint-initaws.d/localstack_sqs_set_queue_attributes.json

# create input and output buckets
awslocal s3api create-bucket --bucket log-bucket-input
awslocal s3api put-bucket-notification-configuration \
    --bucket log-bucket-input \
    --notification-configuration file:///docker-entrypoint-initaws.d/localstack_s3_notification_config.json

awslocal dynamodb create-table \
--table-name log-table-output \
--attribute-definitions \
    AttributeName=id,AttributeType=S \
    AttributeName=date,AttributeType=S \
--key-schema \
    AttributeName=id,KeyType=HASH \
    AttributeName=date,KeyType=RANGE \
--billing-mode=PAY_PER_REQUEST