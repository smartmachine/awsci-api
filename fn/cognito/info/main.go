package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"go.smartmachine.io/awsci-api/pkg/ssm"
	"go.uber.org/zap"
)



type ClientInfoResponse struct {
	ClientId *string `json:"client_id"`
	CallbackURL *string `json:"callback_url"`
}

func ClientInfo(ctx context.Context, request interface{}) (*ClientInfoResponse, error) {
	// Setup structured logging
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	log := logger.Sugar()

	log.Infow("ClientInfo()", "Request", request)

	info, err := ssm.GetClientInfo()
	if err != nil {
		return nil, err
	}

	log.Infow("retrieved client info", "Info", info)

	return &ClientInfoResponse{
		ClientId:    info.ClientID,
		CallbackURL: info.CallbackURL,
	}, nil

}

func main() {
	lambda.Start(ClientInfo)
}