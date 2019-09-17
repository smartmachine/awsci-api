package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"go.smartmachine.io/awsci-api/pkg/ssm"
	"go.smartmachine.io/awsci-api/pkg/util"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"io/ioutil"
	"time"
)

type LoginRequest struct {
	Code string `json:"code"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	User        string `json:"user"`
}

type Session struct {
 	User         string    `json:"user"`
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token"`
	Expiry       time.Time `json:"expiry"`
}

var cognitoConfig = &oauth2.Config{
	Endpoint:     oauth2.Endpoint{
		AuthURL:   "https://auth.awsci.io/oauth2/authorize",
		TokenURL:  "https://auth.awsci.io/oauth2/token",
		AuthStyle: oauth2.AuthStyleAutoDetect,
	},
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

	cognitoConfig.ClientID = *info.ClientID
	cognitoConfig.RedirectURL = *info.CallbackURL

	token, err := cognitoConfig.Exchange(oauth2.NoContext, request.Code)
	if err != nil {
		if _, ok := err.(*oauth2.RetrieveError); ok {
			return nil, fmt.Errorf("oauth2 token exchange error")
		}
		return nil, fmt.Errorf(err.Error())
	}

	log.Infow("obtained token", "Token", token)

	tokenSource := cognitoConfig.TokenSource(oauth2.NoContext, token)
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)

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

	user := userInfo["username"].(string)

	log.Infow("Cognito userInfo", "userInfo", userInfo)

	curTok, err := tokenSource.Token()
	if err != nil {
		return nil, util.NewError(fmt.Sprintf("tokensource error: %+v", err), 400)
	}

	log.Infow("current token", "Token", curTok )

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	ciSession := Session{
		User:         user,
		AccessToken:  curTok.AccessToken,
		TokenType:    curTok.TokenType,
		RefreshToken: curTok.RefreshToken,
		Expiry:       curTok.Expiry,
	}

	item, err := dynamodbattribute.MarshalMap(ciSession)

	db := dynamodb.New(sess)
	tableName := "cognito_sessions2"
	_, err = db.PutItem(&dynamodb.PutItemInput{
		Item:     item,
		TableName: &tableName,
	})

	if err != nil {
		return nil, util.NewError(fmt.Sprintf("dynamodb.PutItem error: %+v", err), 400)
	}

	return &LoginResponse{
		AccessToken: curTok.AccessToken,
		User:        user,
	}, nil

}

func main() {
	lambda.Start(Login)
}