package main

import (
	"context"
	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
)

func cognitoResource(ctx context.Context, event cfn.Event) (physicalResourceID string, data map[string]interface{}, err error) {



	return
}

func main() {
	lambda.Start(cfn.LambdaWrap(cognitoResource))
}
