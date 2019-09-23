package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"go.smartmachine.io/awsci-api/pkg/oauth"
	"go.uber.org/zap"
)

type RefreshRequest struct {
	AccessToken string `json:"access_token"`
}

type RefreshResponse struct {
	AccessToken string `json:"access_token"`
}

func Refresh(ctx context.Context, request *RefreshRequest) (*RefreshResponse, error) {

	// Setup structured logging
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	log := logger.Sugar()

	log.Infow("Refresh Request", "Request", request)

	tokenSource, err := oauth.GetOauthTokenSource(ctx, request.AccessToken)
	if err != nil {
		log.Errorw("unable to obtain a TokenSource", "Error", err)
		return nil, err
	}

	token, err := tokenSource.Token()
	if err != nil {
		log.Errorw("unable to obtain a Token", "Error", err)
		return nil, err
	}
	return &RefreshResponse{AccessToken: token.AccessToken}, nil
}

func main() {
	lambda.Start(Refresh)
}