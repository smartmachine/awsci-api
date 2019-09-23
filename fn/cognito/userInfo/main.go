package main

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/fatih/structs"
	"go.smartmachine.io/awsci-api/pkg/oauth"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"io/ioutil"
)

type UserInfoRequest struct{
	AccessToken *string `json:"access_token"`
}

type UserInfoResponse struct {
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	FamilyName    string `json:"family_name"`
	Name          string `json:"name"`
	Sub           string `json:"sub"`
	Username      string `json:"username"`
}

func UserInfo(ctx context.Context, request UserInfoRequest) (*UserInfoResponse, error) {
	// Setup structured logging
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	log := logger.Sugar()

	log.Infow("UserInfo()", "Request", request)

	tokenSource, err := oauth.GetOauthTokenSource(ctx, *request.AccessToken)
	if err != nil {
		log.Errorw("unable to obtain oauth token source", "Error", structs.Map(err))
		return nil, err
	}

	client := oauth2.NewClient(ctx, tokenSource)

	resp, err := client.Get("https://auth.awsci.io/oauth2/userInfo")

	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorw("error reading body", "Error", err)
		return nil, err
	}

	userInfoResponse := &UserInfoResponse{}
	err = json.Unmarshal(body, userInfoResponse)
	if err != nil {
		log.Errorw("error unmarshalling json", "Error", err)
		return nil, err
	}

	log.Infow("Cognito userInfo", "userInfo", userInfoResponse)
	return userInfoResponse, nil
}

func main() {
	lambda.Start(UserInfo)
}