package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"go.uber.org/zap"
)

type UserInfoRequest struct{
	User *string `json:"user"`
}

type UserInfoResponse struct {
}

func UserInfo(ctx context.Context, request UserInfoRequest) (*UserInfoResponse, error) {
	// Setup structured logging
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	log := logger.Sugar()

	log.Infow("UserInfo()", "Request", request)

	return &UserInfoResponse{}, nil

}

func main() {
	lambda.Start(UserInfo)
}