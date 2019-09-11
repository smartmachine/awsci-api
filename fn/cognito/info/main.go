package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"go.smartmachine.io/awsci-api/pkg/util"
	"log"
)



type ClientInfoResponse struct {
	ClientId *string `json:"client_id"`
	CallbackURL *string `json:"callback_url"`
}

func ClientInfo(ctx context.Context, request interface{}) (*ClientInfoResponse, error) {
	log.Printf("request: %+v", request)

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	ssmSvc := ssm.New(sess)

	getParametersRequest := &ssm.GetParametersInput{
		Names:          []*string{
			aws.String("/cognito/client/id"),
			aws.String("/cognito/client/callbackUrl"),
		},
		WithDecryption: aws.Bool(false),
	}

	log.Printf("ssm.GetParametersRequest: %+v", getParametersRequest)

	getParametersResponse, err := ssmSvc.GetParameters(getParametersRequest)
	if err != nil {
		util.LogAWSError("ssm.GetParameters error: %+v", err)
		return nil, util.NewError(fmt.Sprintf("ssm.GetParameters error: %+v", err), 400)
	}

	log.Printf("ssm.GetParametersResponse: %+v", getParametersResponse)

	response := &ClientInfoResponse{}

	for _, param := range getParametersResponse.Parameters {
		switch *param.Name {
		case "/cognito/client/id":
			response.ClientId = param.Value
		case "/cognito/client/callbackUrl":
			response.CallbackURL = param.Value
		}
	}

	return response, nil

}

func main() {
	lambda.Start(ClientInfo)
}