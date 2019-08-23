package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
)

func Github(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("request: %+v", request)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body: "{\"message\": \"Hello ƛ!\"}",
	}, nil
}

func main() {
  lambda.Start(Github)
}