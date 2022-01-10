package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/aws/aws-lambda-go/lambda"
)

var (
	exportCompleteInterval = 10 * time.Second
	s3BucketName           = os.Getenv("S3_BUCKET_NAME")
)

func CloudwatchS3Export() {
	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	// Create an Amazon S3 service client
	client := cloudwatchlogs.NewFromConfig(cfg)

	// Get list of all log groups
	resp, err := client.DescribeLogGroups(context.TODO(), &cloudwatchlogs.DescribeLogGroupsInput{
		Limit: aws.Int32(50),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Filter log groups by ExportLogs=true tag
	for _, logGroup := range resp.LogGroups {
		tags, err := client.ListTagsLogGroup(context.TODO(), &cloudwatchlogs.ListTagsLogGroupInput{
			LogGroupName: aws.String(*logGroup.LogGroupName),
		})
		if err != nil {
			log.Fatal(err)
		}

		// Export log for log groups with tag
		if _, ok := tags.Tags["ExportLogs"]; ok {
			y, m, d := time.Now().Add(-24 * time.Hour).Date()
			fromDate := time.Date(y, m, d, 0, 0, 0, 0, time.UTC).UnixMilli()

			y1, m1, d1 := time.Now().Date()
			toDate := time.Date(y1, m1, d1, 0, 0, 0, 0, time.UTC).UnixMilli()

			logDate := fmt.Sprintf("%d-%d-%d", y, m, d)
			arr := strings.Split(*logGroup.LogGroupName, "/")
			shortLogGroupName := arr[len(arr)-1]

			task, err := client.CreateExportTask(context.TODO(), &cloudwatchlogs.CreateExportTaskInput{
				Destination:       aws.String(s3BucketName),
				From:              aws.Int64(fromDate),
				LogGroupName:      logGroup.LogGroupName,
				To:                aws.Int64(toDate),
				DestinationPrefix: aws.String(logDate),
				TaskName:          aws.String(shortLogGroupName + "-" + logDate),
			})
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("Started to export LogGroup: %s, date: %s, at %s\n",
				*logGroup.LogGroupName,
				logDate,
				time.Now().Format(time.UnixDate))

			// wait till the task completed
			var statusCode types.ExportTaskStatusCode
			for {
				exportTask, err := client.DescribeExportTasks(context.TODO(), &cloudwatchlogs.DescribeExportTasksInput{
					TaskId: task.TaskId,
				})
				if err != nil {
					log.Fatal(err)
				}

				// stop waiting when status code is "RUNNING", "PENDING" or "PENDING_CANCEL"
				// after that, run a new export task
				statusCode = exportTask.ExportTasks[0].Status.Code
				if statusCode == types.ExportTaskStatusCodeCompleted ||
					statusCode == types.ExportTaskStatusCodeCancelled ||
					statusCode == types.ExportTaskStatusCodeFailed {

					break
				}
				fmt.Print("Waiting export task to complete\n")
				time.Sleep(exportCompleteInterval)
			}

			fmt.Printf("Completed to export LogGroup: %s, date: %s, at %s with exit status code: %s\n",
				*logGroup.LogGroupName,
				logDate,
				time.Now().Format(time.UnixDate),
				statusCode)
		}
	}
}

func main() {
	lambda.Start(CloudwatchS3Export)
}
