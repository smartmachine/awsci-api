package repo

import (
	"context"
	"fmt"
	"github.com/google/go-github/v28/github"
	"go.smartmachine.io/awsci-api/pkg/util"
	"golang.org/x/oauth2"
)

func ListRepos() error {

	tokenSource := util.AwsCiConf.TokenSource(oauth2.NoContext, token)
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := github.NewClient(oauthClient)

	repos, _, err := client.Repositories.List(context.Background(), *user.Login, nil)
	if err != nil {
		return util.NewError(fmt.Sprintf("github.Repositories.List failed: %+v", err), 400)
	}

	var repoNames = []string{}
	for _, repo := range repos {
		repoNames = append(repoNames, *repo.Name)
	}
	return nil
}
