package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/go-github/v28/github"
	"log"
)

func Github(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("request: %+v", request)
	client := github.NewClient(nil)
	repos, _, err := client.Repositories.List(context.Background(), "knutster", &github.RepositoryListOptions{})
	jsonRepos, err := json.Marshal(repos)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body: fmt.Sprintf("{\"repos\": %s", jsonRepos),
	}, err
}

func main() {
  lambda.Start(Github)
}