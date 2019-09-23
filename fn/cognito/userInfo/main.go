package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/fatih/structs"
	"go.smartmachine.io/awsci-api/pkg/oauth"
	"go.uber.org/zap"
	"io/ioutil"
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

	client, err := oauth.GetOauthClient(*request.AccessToken)
	if err != nil {
		log.Errorw("unable to obtain oauth client", "Error", structs.Map(err))
		return nil, err
	}

	resp, err := client.Get("https://auth.awsci.io/oauth2/userInfo")

	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorw("error reading body", "Error", err)
		return nil, err
	}

	userInfo := make(map[string]interface{})
	err = json.Unmarshal(body, &userInfo)
	if err != nil {
		log.Errorw("error unmarshalling json", "Error", err)
		return nil, err
	}

	log.Infow("Cognito userInfo", "userInfo", userInfo)

	return &UserInfoResponse{}, nil

}

func main() {
	lambda.Start(UserInfo)
}