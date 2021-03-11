# Simple SQS processor written in Golang
this is a show case on how to use localstack as an alternative for local development using AWS services

## How to run
run localstack, this will run s3, sqs and dynamodb and create the resources in `localstack_setup/001_localstack_setup.sh`
```
docker-compose up
```

run the processor
```
go run main.go
```

you are done :) enjoy!

## Usefull commands
put an object in the input bucket 
```
aws --endpoint-url=http://localhost:4566 \
    s3api put-object \
    --bucket log-bucket-input \
    --key dummy.json \
    --body debug_files/dummy.json
```

get messages from the the dead letter queue
```
aws --endpoint-url=http://localhost:4566 \
    sqs receive-message \
    --queue-url http://localhost:4566/000000000000/log-queue-dlq \
    --attribute-names All \
    --message-attribute-names All \
    --max-number-of-messages 10
```

delete message from the dead letter queue
```
aws --endpoint-url=http://localhost:4566 \
    sqs delete-message \
    --queue-url http://localhost:4566/000000000000/log-queue-dlq \
    --receipt-handle HERE
```

scan dynamo table to view all the records
```
aws --endpoint-url=http://localhost:4566 \
    dynamodb scan \
    --table-name log-table-output
```