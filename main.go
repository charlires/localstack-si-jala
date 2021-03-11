package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sqs"
	guuid "github.com/google/uuid"
	mq "github.com/remind101/mq-go"
)

type FileContent struct {
	ID      string    `json:"id,omitempty"`
	Message string    `json:"message,omitempty"`
	Error   bool      `json:"error,omitempty"`
	Date    time.Time `json:"date,omitempty"`
}

func main() {

	conf := aws.NewConfig().
		WithRegion("us-east-1").
		WithEndpoint("http://localhost:4566").
		WithS3ForcePathStyle(true)

	awsSession := session.Must(session.NewSession(conf))

	// Create a downloader with the session and default options
	downloader := s3manager.NewDownloader(awsSession)

	dynamoSvc := dynamodb.New(awsSession)

	queueURL := "http://localhost:4566/000000000000/log-queue"
	bucketName := "log-bucket-input"
	tableName := "log-table-output"

	h := mq.HandlerFunc(func(m *mq.Message) error {
		// fmt.Printf("Received message: %s", aws.StringValue(m.SQSMessage.Body))
		var s3Event events.S3Event
		if err := json.Unmarshal([]byte(*m.SQSMessage.Body), &s3Event); err != nil {
			return err
		}

		for _, record := range s3Event.Records {
			s3Record := record.S3
			fmt.Printf("[%s - %s] Bucket = %s, Key = %s \n", record.EventSource, record.EventTime, s3Record.Bucket.Name, s3Record.Object.Key)

			// Create a file to write the S3 Object contents to.
			tempFile := fmt.Sprintf("tmp_files/%s", s3Record.Object.Key)
			f, err := os.Create(tempFile)
			if err != nil {
				return fmt.Errorf("failed to create file %q, %v", tempFile, err)
			}

			// Write the contents of S3 Object to the file
			n, err := downloader.Download(f, &s3.GetObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(s3Record.Object.Key),
			})
			if err != nil {
				return fmt.Errorf("failed to download file, %v", err)
			}

			// Read file content
			data, err := ioutil.ReadFile(tempFile)
			fmt.Printf("file content, %s\n", string(data))

			var fileContent FileContent
			if err := json.Unmarshal(data, &fileContent); err != nil {
				return err
			}

			if fileContent.Error {
				return fmt.Errorf("intentional error!!")
			}

			fileContent.Date = time.Now()
			fileContent.ID = guuid.NewString()
			av, err := dynamodbattribute.MarshalMap(fileContent)
			// Create item in table Movies
			input := &dynamodb.PutItemInput{
				Item:      av,
				TableName: aws.String(tableName),
			}

			_, err = dynamoSvc.PutItem(input)
			if err != nil {
				return fmt.Errorf("error calling PutItem: %v", err)
			}

			fmt.Printf("file downloaded, %d bytes\n", n)
			err = os.Remove(tempFile)
			if err != nil {
				fmt.Printf("failed to delete file, %v", err)
				return err
			}

		}

		// Returning no error signifies the message was processed successfully.
		// The Server will queue the message for deletion.
		return nil
	})

	stop := make(chan os.Signal, 1)
	signal.Notify(
		stop,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGHUP,
	)

	// Configure SQS message processor
	s := mq.NewServer(queueURL, h)
	s.Logger = log.Default()

	mq.WithClient(sqs.New(awsSession))(s)

	// Start a loop to receive SQS messages and pass them to the Handler.
	s.Start()

	// Catch application's interrupt signals (Kill, Hang up and Interrupt)
	<-stop
	defer s.Shutdown(context.Background())
}
