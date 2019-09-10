package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/go-github/v28/github"
	"github.com/satori/go.uuid"
	"go.smartmachine.io/awsci-api/pkg/util"
	"golang.org/x/oauth2"
	"log"
)

type LoginRequest struct {
	Code string `json:"code"`
}

type LoginResponse struct {
	User    string `json:"user"`
	Session string `json:"session"`
}

type Session struct {
	SessionId string `json:"session_id"`
	Login     string `json:"github"`
	AuthToken string `json:"auth_token"`
}



func Login(ctx context.Context, request *LoginRequest) (*LoginResponse, error) {
	log.Printf("request: %+v", request)

	if request.Code == "" {
		return nil, util.NewError("code is invalid", 400)
	}

	token, err := util.AwsCiConf.Exchange(oauth2.NoContext, request.Code)
	if err != nil {
		return nil, util.NewError(fmt.Sprintf("oauth2.Exchange failed: %+v", err), 400)
	}

	log.Printf("obtained token: %+v", token)

	tokenSource := util.AwsCiConf.TokenSource(oauth2.NoContext, token)
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := github.NewClient(oauthClient)

	user, _, err := client.Users.Get(context.Background(), "")
	if err != nil {
		return nil, util.NewError(fmt.Sprintf("github.Users.Get failed: %+v", err), 400)
	}


	curTok, err := tokenSource.Token()
	if err != nil {
		return nil, util.NewError(fmt.Sprintf("tokensource error: %+v", err), 400)
	}

	log.Printf("current token: %+v", curTok )

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	sessionId := uuid.NewV4().String()

	ciSession := Session{
		Login:     *user.Login,
		AuthToken: curTok.AccessToken,
		SessionId: sessionId,
	}

	item, err := dynamodbattribute.MarshalMap(ciSession)

	db := dynamodb.New(sess)
	tableName := "sessions"
	_, err = db.PutItem(&dynamodb.PutItemInput{
		Item:     item,
		TableName: &tableName,
	})

	if err != nil {
		return nil, util.NewError(fmt.Sprintf("dynamodb.PutItem error: %+v", err), 400)
	}

	return &LoginResponse{
		User:    *user.Login,
		Session: sessionId,
	}, nil

}

func main() {
	lambda.Start(Login)
}