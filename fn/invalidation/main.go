package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
)

func Invalidate(ctx context.Context, event *events.S3Event) error {
	log.Printf("event: %+v", event)
	return nil
}

func main() {
	lambda.Start(Invalidate)
}