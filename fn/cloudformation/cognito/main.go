package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	cognito "github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/fatih/structs"
	"go.uber.org/zap"
)

func cognitoResource(ctx context.Context, event cfn.Event) (physicalResourceID string, data map[string]interface{}, err error) {

	// Setup structured logging
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	log := logger.Sugar().With("RequestID", event.RequestID)

	log.Infow("event received", "Event", event)

	sess := &session.Session{}
	sess, err = session.NewSession()

	if err != nil {
		log.Errorw("AWS session error", "Error", err)
		return
	}

	certArn := event.ResourceProperties["CertificateArn"].(string)
	authDomain := event.ResourceProperties["AuthDomain"].(string)
	baseDomain := event.ResourceProperties["BaseDomain"].(string)
	userPoolId := event.ResourceProperties["UserPoolId"].(string)
	userPoolClientId := event.ResourceProperties["UserPoolClientId"].(string)
	callbackUrl := event.ResourceProperties["CallbackUrl"].(string)
	logoutUrl := event.ResourceProperties["LogoutUrl"].(string)

	physicalResourceID = userPoolClientId

	switch event.RequestType {
	case cfn.RequestCreate:
		cognitoSvc := cognito.New(sess)

		createUserPoolDomainResponse := &cognito.CreateUserPoolDomainOutput{}
		createUserPoolDomainRequest := &cognito.CreateUserPoolDomainInput{
			CustomDomainConfig: &cognito.CustomDomainConfigType{
				CertificateArn: &certArn,
			},
			Domain:     &authDomain,
			UserPoolId: &userPoolId,
		}

		log.Infow("Cognito CreateUserPoolDomain Request", "Request", structs.Map(createUserPoolDomainRequest))

		createUserPoolDomainResponse, err = cognitoSvc.CreateUserPoolDomain(createUserPoolDomainRequest)

		if err != nil {
			log.Errorw("Cognito CreateUserPoolDomain Error", "Error", err)
			return
		}

		log.Infow("Cognito CreateUserPoolDomain Response", "Response", structs.Map(createUserPoolDomainResponse))

		createResourceServerResponse := &cognito.CreateResourceServerOutput{}
		createResourceServerRequest := &cognito.CreateResourceServerInput{
			Identifier: aws.String("https://api.awsci.io"),
			Name:       aws.String("AWSCI Resource Server"),
			Scopes:     []*cognito.ResourceServerScopeType{
				{
					ScopeDescription: aws.String("User Scope"),
					ScopeName:        aws.String("user"),
				},
				{
					ScopeDescription: aws.String("Admin Scope"),
					ScopeName:        aws.String("admin"),
				},
			},
			UserPoolId: &userPoolId,
		}

		log.Infow("Cognito CreateResourceServer Request", "Request", structs.Map(createResourceServerRequest))

		createResourceServerResponse, err = cognitoSvc.CreateResourceServer(createResourceServerRequest)

		if err != nil {
			log.Errorw("Cognito CreateResourceServer Error", "Error", err)
		}

		log.Infow("Cognito CreateResourceServer Response", "Response", structs.Map(createResourceServerResponse))

		updateClientResponse := &cognito.UpdateUserPoolClientOutput{}
		updateClientRequest := &cognito.UpdateUserPoolClientInput{
			UserPoolId:                      aws.String(userPoolId),
			ClientId:                        aws.String(userPoolClientId),
			RefreshTokenValidity:            aws.Int64(30),
			ExplicitAuthFlows:               []*string{aws.String(cognito.AuthFlowTypeUserPasswordAuth)},
			SupportedIdentityProviders:      []*string{aws.String("COGNITO")},
			CallbackURLs:                    []*string{aws.String(callbackUrl)},
			LogoutURLs:                      []*string{aws.String(logoutUrl)},
			AllowedOAuthFlows:               []*string{aws.String(cognito.OAuthFlowTypeCode)},
			AllowedOAuthScopes:              []*string{
				aws.String("https://api.awsci.io/user"),
				aws.String("https://api.awsci.io/admin"),
				aws.String("email"),
				aws.String("openid"),
				aws.String("profile"),
			},
			AllowedOAuthFlowsUserPoolClient: aws.Bool(true),
		}

		log.Infow("Cognito UpdateUserPoolClient Request", "Request", structs.Map(updateClientRequest))

		updateClientResponse, err = cognitoSvc.UpdateUserPoolClient(updateClientRequest)

		if err != nil {
			log.Errorw("Cognito UpdateUserPoolClient Error", "Error", err)
			return
		}

		log.Infow("Cognito UpdateUserPoolClient Response", "Response", structs.Map(updateClientResponse))

		route53Svc := route53.New(sess)

		listHostedZonesResponse := &route53.ListHostedZonesByNameOutput{}
		listHostedZonesRequest := &route53.ListHostedZonesByNameInput{
			DNSName: aws.String(baseDomain + "."),
		}

		log.Infow("Route53 ListHostedZonesByName Request", "Request", structs.Map(listHostedZonesRequest))

		listHostedZonesResponse, err = route53Svc.ListHostedZonesByName(listHostedZonesRequest)
		if err != nil {
			log.Errorw("Route53 ListHostedZonesByName Error", "Error", err)
			return
		}

		log.Infow("Route53 ListHostedZones Response", "Response", structs.Map(listHostedZonesResponse))

		zoneId := ""
		zoneId, err = extractZoneId(listHostedZonesResponse, baseDomain)
		if err != nil {
			log.Errorw("Route53 Zone Extraction Error", "Error", err)
			return
		}

		changeResourceRecordResponse := &route53.ChangeResourceRecordSetsOutput{}
		changeResourceRecordRequest := &route53.ChangeResourceRecordSetsInput{
			ChangeBatch: &route53.ChangeBatch{
				Changes: []*route53.Change{
					{
						Action: aws.String(route53.ChangeActionCreate),
						ResourceRecordSet: &route53.ResourceRecordSet{
							Name: aws.String(authDomain),
							AliasTarget: &route53.AliasTarget{
								DNSName:              createUserPoolDomainResponse.CloudFrontDomain,
								EvaluateTargetHealth: aws.Bool(false),
								// CloudFront Hosted Zone ID: https://docs.aws.amazon.com/general/latest/gr/rande.html#cf_region
								HostedZoneId: aws.String("Z2FDTNDATAQYW2"),
							},
							Type: aws.String(route53.RRTypeA),
						},
					},
				},
				Comment: aws.String("api domain for cognito"),
			},
			HostedZoneId: &zoneId,
		}

		log.Infow("Route53 ChangeResourceRecordSets Request", "Request", structs.Map(changeResourceRecordRequest))

		changeResourceRecordResponse, err = route53Svc.ChangeResourceRecordSets(changeResourceRecordRequest)
		if err != nil {
			log.Errorw("Route53 ChangeResourceRecordSets Error", "Error", err)
			return
		}

		log.Infow("Route53 ChangeResourceRecordSets Response", "Response", structs.Map(changeResourceRecordResponse))

		data = map[string]interface{}{
			"message": "custom resource created",
		}

	case cfn.RequestUpdate:
		cognitoSvc := cognito.New(sess)

		updateClientResponse := &cognito.UpdateUserPoolClientOutput{}
		updateClientRequest := &cognito.UpdateUserPoolClientInput{
			UserPoolId:                      aws.String(userPoolId),
			ClientId:                        aws.String(userPoolClientId),
			RefreshTokenValidity:            aws.Int64(30),
			ExplicitAuthFlows:               []*string{aws.String(cognito.AuthFlowTypeUserPasswordAuth)},
			SupportedIdentityProviders:      []*string{aws.String("COGNITO")},
			CallbackURLs:                    []*string{aws.String(callbackUrl)},
			LogoutURLs:                      []*string{aws.String(logoutUrl)},
			AllowedOAuthFlows:               []*string{aws.String(cognito.OAuthFlowTypeCode)},
			AllowedOAuthScopes:              []*string{aws.String("openid"), aws.String("email"), aws.String("profile")},
			AllowedOAuthFlowsUserPoolClient: aws.Bool(true),
		}

		log.Infow("Cognito UpdateUserPoolClient Request", "Request", structs.Map(updateClientRequest))

		updateClientResponse, err = cognitoSvc.UpdateUserPoolClient(updateClientRequest)

		if err != nil {
			log.Errorw("Cognito UpdateUserPoolClient Error", "Error", err)
			return
		}

		log.Infow("Cognito UpdateUserPoolClient Response", "Response", structs.Map(updateClientResponse))

		data = map[string]interface{}{
			"message": "custom resource updated",
		}

	case cfn.RequestDelete:
		route53Svc := route53.New(sess)

		listHostedZonesResponse := &route53.ListHostedZonesByNameOutput{}
		listHostedZonesRequest := &route53.ListHostedZonesByNameInput{
			DNSName: aws.String(baseDomain + "."),
		}

		log.Infow("Route53 ListHostedZones Request", "Request", structs.Map(listHostedZonesRequest))

		listHostedZonesResponse, err = route53Svc.ListHostedZonesByName(listHostedZonesRequest)
		if err != nil {
			log.Errorw("Route53 ListHostedZones Error", "Error", err)
			return
		}

		log.Infow("Route53 ListHostedZones Response", "Response", structs.Map(listHostedZonesResponse))

		zoneId := ""
		zoneId, err = extractZoneId(listHostedZonesResponse, baseDomain)
		if err != nil {
			log.Errorw("Route53 Zone Extraction Error", "Error", err)
			return
		}

		cognitoSvc := cognito.New(sess)

		describeUserPoolDomainResponse := &cognito.DescribeUserPoolDomainOutput{}
		describeUserPoolDomainRequest := &cognito.DescribeUserPoolDomainInput{
			Domain: aws.String(authDomain),
		}

		log.Infow("Cognito DescribeUserPoolDomain Request", "Request", structs.Map(describeUserPoolDomainRequest))

		describeUserPoolDomainResponse, err = cognitoSvc.DescribeUserPoolDomain(describeUserPoolDomainRequest)
		if err != nil {
			log.Errorw("Cognito DescribeUserPoolDomain Error", "Error", err)
		}

		log.Infow("Cognito DescribeUserPoolDomain Response", "Response", structs.Map(describeUserPoolDomainResponse))

		changeResourceRecordResponse := &route53.ChangeResourceRecordSetsOutput{}
		changeResourceRecordRequest := &route53.ChangeResourceRecordSetsInput{
			ChangeBatch: &route53.ChangeBatch{
				Changes: []*route53.Change{
					{
						Action: aws.String(route53.ChangeActionDelete),
						ResourceRecordSet: &route53.ResourceRecordSet{
							Name: aws.String(authDomain),
							AliasTarget: &route53.AliasTarget{
								DNSName:              describeUserPoolDomainResponse.DomainDescription.CloudFrontDistribution,
								EvaluateTargetHealth: aws.Bool(false),
								// CloudFront Hosted Zone ID: https://docs.aws.amazon.com/general/latest/gr/rande.html#cf_region
								HostedZoneId: aws.String("Z2FDTNDATAQYW2"),
							},
							Type: aws.String(route53.RRTypeA),
						},
					},
				},
				Comment: aws.String("api domain for Cognito"),
			},
			HostedZoneId: &zoneId,
		}

		log.Infow("Route53 ChangeResourceRecordSets Request", "Request", structs.Map(changeResourceRecordRequest))

		changeResourceRecordResponse, err = route53Svc.ChangeResourceRecordSets(changeResourceRecordRequest)
		if err != nil {
			log.Errorw("Route53 ChangeResourceRecordSets Error", "Error", err)
			return
		}

		log.Infow("Route53 ChangeResourceRecordSets Response", "Response", structs.Map(changeResourceRecordResponse))

		updateClientResponse := &cognito.UpdateUserPoolClientOutput{}
		updateClientRequest := &cognito.UpdateUserPoolClientInput{
			UserPoolId:                      aws.String(userPoolId),
			ClientId:                        aws.String(userPoolClientId),
			RefreshTokenValidity:            aws.Int64(30),
			ExplicitAuthFlows:               []*string{},
			SupportedIdentityProviders:      []*string{},
			CallbackURLs:                    []*string{},
			LogoutURLs:                      []*string{},
			AllowedOAuthFlows:               []*string{},
			AllowedOAuthScopes:              []*string{},
			AllowedOAuthFlowsUserPoolClient: aws.Bool(false),
		}

		log.Infow("Cognito UpdateUserPoolClient Request", "Request", structs.Map(updateClientRequest))

		updateClientResponse, err = cognitoSvc.UpdateUserPoolClient(updateClientRequest)

		if err != nil {
			log.Errorw("Cognito UpdateUserPoolClient Error", "Error", err)
			return
		}

		log.Infow("Cognito UpdateUserPoolClient Response", "Response", structs.Map(updateClientResponse))

		deleteResourceServerResponse := &cognito.DeleteResourceServerOutput{}
		deleteResourceServerRequest := &cognito.DeleteResourceServerInput{
			Identifier: aws.String("https://api.awsci.io"),
			UserPoolId: &userPoolId,
		}

		log.Infow("Cognito DeleteResourceServer Request", "Request", structs.Map(deleteResourceServerRequest))

		deleteResourceServerResponse, err = cognitoSvc.DeleteResourceServer(deleteResourceServerRequest)

		if err != nil {
			log.Errorw("Cognito DeleteResourceServer Error", "Error", structs.Map(err))
		}

		log.Infow("Cognito DeleteResourceServer Response", "Response", structs.Map(deleteResourceServerResponse))

		deleteUserPoolDomainResponse := &cognito.DeleteUserPoolDomainOutput{}
		deleteUserPoolDomainRequest := &cognito.DeleteUserPoolDomainInput{
			Domain:     &authDomain,
			UserPoolId: &userPoolId,
		}

		log.Infow("Cognito DeleteUserPoolDomain Request", "Request", structs.Map(deleteUserPoolDomainRequest))

		deleteUserPoolDomainResponse, err = cognitoSvc.DeleteUserPoolDomain(deleteUserPoolDomainRequest)
		if err != nil {
			log.Errorw("Cognito DeleteUserPoolDomain Error", "Error", err)
			return
		}

		log.Infow("Cognito DeleteUserPoolDomain Response", "Response", structs.Map(deleteUserPoolDomainResponse))

		data = map[string]interface{}{
			"message": "custom resource deleted",
		}
	}

	return
}

func extractZoneId(zones *route53.ListHostedZonesByNameOutput, domain string) (string, error) {
	for _, zone := range zones.HostedZones {
		if *zone.Name == domain+"." {
			return *zone.Id, nil
		}
	}
	return "", fmt.Errorf("unable to find HostedZone for domain %s", domain)
}

func main() {
	lambda.Start(cfn.LambdaWrap(cognitoResource))
}
