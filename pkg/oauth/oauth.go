package oauth

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/fatih/structs"
	"go.smartmachine.io/awsci-api/pkg/ssm"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"net/http"
	"strings"
	"time"
)

type CognitoSession struct {
	User         string    `json:"user"`
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token"`
	Expiry       time.Time `json:"expiry"`
}

func NewCognitoConfig() (*oauth2.Config, error) {
	// Setup structured logging
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	log := logger.Sugar()

	config :=  &oauth2.Config{
		Endpoint:     oauth2.Endpoint{
			AuthURL:   "https://auth.awsci.io/oauth2/authorize",
			TokenURL:  "https://auth.awsci.io/oauth2/token",
			AuthStyle: oauth2.AuthStyleAutoDetect,
		},
	}

	info, err := ssm.GetClientInfo()
	if err != nil {
		return nil, err
	}

	log.Infow("retrieved client info", "Info", info)

	config.ClientID = *info.ClientID
	config.RedirectURL = *info.CallbackURL

	return config, nil
}

func (cognitoSession CognitoSession) SaveSession() error {
	item, err := dynamodbattribute.MarshalMap(cognitoSession)

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	db := dynamodb.New(sess)
	tableName := "cognito_sessions"
	_, err = db.PutItem(&dynamodb.PutItemInput{
		Item:     item,
		TableName: &tableName,
	})

	if err != nil {
		return err
	}

	return nil
}

func GetOauthClient(bearerToken string) (*http.Client, error) {
	// Setup structured logging
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	log := logger.Sugar()

	accessToken := bearerToken
	if strings.HasPrefix(bearerToken, "Bearer ") {
		accessToken = bearerToken[7:]
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	dbSvc := dynamodb.New(sess)

	queryRequest := &dynamodb.QueryInput{
		TableName: aws.String("cognito_sessions"),
		IndexName: aws.String("AccessTokenIndex"),
		KeyConditionExpression: aws.String("access_token = :tok"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":tok": {
				S: &accessToken,
			},
		},
	}

	log.Infow("DynamoDB Query Request", "Request", structs.Map(queryRequest))

	queryResponse, err := dbSvc.Query(queryRequest)
	if err != nil {
		log.Errorw("DynamoDB Query Error", "Error", structs.Map(err))
		return nil, err
	}

	log.Infow("DynamoDB Query Response", "Response", structs.Map(queryResponse))

	cognitoSession := &CognitoSession{}
	err = dynamodbattribute.UnmarshalMap(queryResponse.Items[0], cognitoSession)
	if err != nil {
		return nil, err
	}

	token := &oauth2.Token{
		AccessToken:  cognitoSession.AccessToken,
		TokenType:    cognitoSession.TokenType,
		RefreshToken: cognitoSession.RefreshToken,
		Expiry:       cognitoSession.Expiry,
	}

	config, err := NewCognitoConfig()
	if err != nil {
		log.Errorw("unable to obtain config", "Error", structs.Map(err))
		return nil, err
	}

	tokenSource := config.TokenSource(context.Background(), token)
	newTok, err := tokenSource.Token()
	if err != nil {
		log.Errorw("unable to obtain token", "Error", structs.Map(err))
		return nil, err
	}

	if newTok.AccessToken != token.AccessToken {
		cognitoSession.AccessToken = newTok.AccessToken
		cognitoSession.Expiry = newTok.Expiry
		err = cognitoSession.SaveSession()
		if err != nil {
			return nil, err
		}
	}

	return oauth2.NewClient(context.Background(), tokenSource), nil
}

