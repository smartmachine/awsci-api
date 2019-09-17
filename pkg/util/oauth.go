package util

import (
	"golang.org/x/oauth2"
	ghoauth "golang.org/x/oauth2/amazon"
)

var AwsCiConf = &oauth2.Config{
	ClientID:     "9ba972db1d356346f618",
	ClientSecret: "96cb66b6a3f9e21cb0b38a9130f0c10f3708ada0",
	Endpoint:     ghoauth.Endpoint,
}