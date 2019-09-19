package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"go.uber.org/zap"
)

type UserInfoRequest struct{
	AccessToken *string `json:"access_token"`
}

type UserInfoResponse struct {
}

func UserInfo(ctx context.Context, request UserInfoRequest) (*UserInfoResponse, error) {
	// Setup structured logging
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	log := logger.Sugar()

	log.Infow("UserInfo()", "Request", request)

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	dbSvc := dynamodb.New(sess)
	
	dbSvc.GetItem(&dynamodb.GetItemInput{
		AttributesToGet:          nil,
		ConsistentRead:           nil,
		ExpressionAttributeNames: nil,
		Key:                      nil,
		ProjectionExpression:     nil,
		ReturnConsumedCapacity:   nil,
		TableName:                nil,
	})




	return &UserInfoResponse{}, nil

}

func main() {
	lambda.Start(UserInfo)
}