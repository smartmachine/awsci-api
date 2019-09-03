package main

import (
	"context"
	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
)

func cognitoResource(ctx context.Context, event cfn.Event) (physicalResourceID string, data map[string]interface{}, err error) {
	log.Printf("Event Received: %+v", event)
	return
}

func main() {
	lambda.Start(cfn.LambdaWrap(cognitoResource))
}
