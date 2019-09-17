package ssm

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/fatih/structs"
	"go.uber.org/zap"
)

type ClientInfo struct{
	ClientID    *string `json:"client_id"`
	CallbackURL *string `json:"callback_url"`
}

func GetClientInfo() (*ClientInfo, error) {

	// Setup structured logging
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	log := logger.Sugar()

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	ssmSvc := ssm.New(sess)

	getParametersRequest := &ssm.GetParametersInput{
		Names:          []*string{
			aws.String("/cognito/client/id"),
			aws.String("/cognito/client/callbackUrl"),
		},
		WithDecryption: aws.Bool(false),
	}

	log.Infow("SSM GetParameters Request", "Request", structs.Map(getParametersRequest))

	getParametersResponse, err := ssmSvc.GetParameters(getParametersRequest)
	if err != nil {
		log.Errorw("SSM GetParameters Error", "Error", err)
		return nil, err
	}

	log.Infow("SSM GetParameters Response", "Response", structs.Map(getParametersResponse))

	info := &ClientInfo{}

	for _, param := range getParametersResponse.Parameters {
		switch *param.Name {
		case "/cognito/client/id":
			info.ClientID = param.Value
		case "/cognito/client/callbackUrl":
			info.CallbackURL = param.Value
		}
	}

	if info.ClientID == nil && info.CallbackURL == nil {
		log.Errorw("unable to extract all parameters from ssm", "ClientInfo", info)
		return nil, fmt.Errorf("unable to extract all parameters from ssm: %+v", info)
	}

	return info, nil
}
