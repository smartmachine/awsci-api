package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"go.smartmachine.io/awsci-api/pkg/oauth"
	"go.smartmachine.io/awsci-api/pkg/ssm"
	"go.smartmachine.io/awsci-api/pkg/util"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"io/ioutil"
)

type LoginRequest struct {
	Code string `json:"code"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
}

func Login(ctx context.Context, request *LoginRequest) (*LoginResponse, error) {

	// Setup structured logging
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	log := logger.Sugar()


	log.Infow("Login()", "Request", request)

	if request.Code == "" {
		return nil, util.NewError("code is invalid", 400)
	}

	info, err := ssm.GetClientInfo()
	if err != nil {
		return nil, err
	}

	log.Infow("retrieved client info", "Info", info)

	cognitoConfig, err := oauth.NewCognitoConfig()
	if err != nil {
		return nil, err
	}

	token, err := cognitoConfig.Exchange(ctx, request.Code)
	if err != nil {
		if _, ok := err.(*oauth2.RetrieveError); ok {
			return nil, fmt.Errorf("oauth2 token exchange error")
		}
		return nil, fmt.Errorf(err.Error())
	}

	log.Infow("obtained token", "Token", token)

	tokenSource := cognitoConfig.TokenSource(ctx, token)
	oauthClient := oauth2.NewClient(ctx, tokenSource)

	resp, err := oauthClient.Get("https://auth.awsci.io/oauth2/userInfo")

	if err != nil {
		return nil, util.NewError(fmt.Sprintf("cognito userInfo failed: %+v", err), 400)
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
	user := userInfo["username"].(string)

	curTok, err := tokenSource.Token()
	if err != nil {
		return nil, err
	}

	log.Infow("current token", "Token", curTok )

	ciSession := oauth.CognitoSession{
		User:         user,
		AccessToken:  curTok.AccessToken,
		TokenType:    curTok.TokenType,
		RefreshToken: curTok.RefreshToken,
		Expiry:       curTok.Expiry,
	}

	err = ciSession.SaveSession()

	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken: curTok.AccessToken,
	}, nil

}

func main() {
	lambda.Start(Login)
}