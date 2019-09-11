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
	"go.smartmachine.io/awsci-api/pkg/util"
	"log"
)

func cognitoResource(ctx context.Context, event cfn.Event) (physicalResourceID string, data map[string]interface{}, err error) {
	log.Printf("Event Received: %+v", event)

	sess := &session.Session{}
	sess, err = session.NewSession()

	if err != nil {
		util.LogAWSError("AWS Session Error: %+v", err)
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

		log.Printf("Cognito CreateUserPoolDomain Request: %+v", createUserPoolDomainRequest)

		createUserPoolDomainResponse, err = cognitoSvc.CreateUserPoolDomain(createUserPoolDomainRequest)

		if err != nil {
			util.LogAWSError("Cognito CreateUserPoolDomain Error: %+v", err)
			return
		}

		log.Printf("Cognito CreateUserPoolDomain Response: %+v", createUserPoolDomainResponse)

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

		log.Printf("Cognito UpdateUserPoolClient Request: %+v", updateClientRequest)

		updateClientResponse, err = cognitoSvc.UpdateUserPoolClient(updateClientRequest)

		if err != nil {
			util.LogAWSError("Cognito UpdateUserPoolClient Error: %+v", err)
			return
		}

		log.Printf("Cognito UpdateUserPoolClient Response: %+v", updateClientResponse)

		route53Svc := route53.New(sess)

		listHostedZonesResponse := &route53.ListHostedZonesByNameOutput{}
		listHostedZonesRequest := &route53.ListHostedZonesByNameInput{
			DNSName: aws.String(baseDomain + "."),
		}

		log.Printf("Route53 ListHostedZonesByName Request: %+v", listHostedZonesRequest)

		listHostedZonesResponse, err = route53Svc.ListHostedZonesByName(listHostedZonesRequest)
		if err != nil {
			util.LogAWSError("Route53 ListHostedZonesByName Error: %+v", err)
			return
		}

		log.Printf("ListHostedZones Response: %+v", listHostedZonesResponse)

		zoneId := ""
		zoneId, err = extractZoneId(listHostedZonesResponse, baseDomain)
		if err != nil {
			log.Printf("extractZoneId: %+v", err)
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

		log.Printf("Route53 ChangeResourceRecordSets Request: %+v", changeResourceRecordRequest)

		changeResourceRecordResponse, err = route53Svc.ChangeResourceRecordSets(changeResourceRecordRequest)
		if err != nil {
			util.LogAWSError("Route53 ChangeResourceRecordSets Error: %+v", err)
			return
		}

		log.Printf("Route53 ChangeResourceRecordSets Response: %+v", changeResourceRecordResponse)

		data = map[string]interface{}{
			"CloudFrontDomain":    *createUserPoolDomainResponse.CloudFrontDomain,
			"RecordChangeStatus":  *changeResourceRecordResponse.ChangeInfo.Status,
			"RecordChangeComment": *changeResourceRecordResponse.ChangeInfo.Comment,
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

		log.Printf("Cognito UpdateUserPoolClient Request: %+v", updateClientRequest)

		updateClientResponse, err = cognitoSvc.UpdateUserPoolClient(updateClientRequest)

		if err != nil {
			util.LogAWSError("Cognito UpdateUserPoolClient Error: %+v", err)
			return
		}

		log.Printf("Cognito UpdateUserPoolClient Response: %+v", updateClientResponse)

		emptyString := ""

		data = map[string]interface{}{
			"CloudFrontDomain":    emptyString,
			"RecordChangeStatus":  emptyString,
			"RecordChangeComment": emptyString,
		}

	case cfn.RequestDelete:
		route53Svc := route53.New(sess)

		listHostedZonesResponse := &route53.ListHostedZonesByNameOutput{}
		listHostedZonesRequest := &route53.ListHostedZonesByNameInput{
			DNSName: aws.String(baseDomain + "."),
		}

		log.Printf("Route53 ListHostedZones Request: %+v", listHostedZonesRequest)

		listHostedZonesResponse, err = route53Svc.ListHostedZonesByName(listHostedZonesRequest)
		if err != nil {
			util.LogAWSError("Route53 ListHostedZones Error: %+v", err)
			return
		}

		log.Printf("Route53 ListHostedZones Response: %+v", listHostedZonesResponse)

		zoneId := ""
		zoneId, err = extractZoneId(listHostedZonesResponse, baseDomain)
		if err != nil {
			log.Printf("extractZoneId: %+v", err)
			return
		}

		cognitoSvc := cognito.New(sess)

		describeUserPoolDomainResponse := &cognito.DescribeUserPoolDomainOutput{}
		describeUserPoolDomainRequest := &cognito.DescribeUserPoolDomainInput{
			Domain: aws.String(authDomain),
		}

		log.Printf("Cognito DescribeUserPoolDomain Request: +%v", describeUserPoolDomainRequest)

		describeUserPoolDomainResponse, err = cognitoSvc.DescribeUserPoolDomain(describeUserPoolDomainRequest)
		if err != nil {
			util.LogAWSError("Cognito DescribeUserPoolDomain Error: %+v", err)
		}

		log.Printf("Cognito DescribeUserPoolDomain Response: %+v", describeUserPoolDomainResponse)

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

		log.Printf("Route53 ChangeResourceRecordSets Request: %+v", changeResourceRecordRequest)

		changeResourceRecordResponse, err = route53Svc.ChangeResourceRecordSets(changeResourceRecordRequest)
		if err != nil {
			util.LogAWSError("Route53 ChangeResourceRecordSets Error: %+v", err)
			return
		}

		log.Printf("Route53 ChangeResourceRecordSets Response: %+v", changeResourceRecordResponse)

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

		log.Printf("Cognito UpdateUserPoolClient Request: %+v", updateClientRequest)

		updateClientResponse, err = cognitoSvc.UpdateUserPoolClient(updateClientRequest)

		if err != nil {
			util.LogAWSError("Cognito UpdateUserPoolClient Error: %+v", err)
			return
		}

		log.Printf("Cognito UpdateUserPoolClient Response: %+v", updateClientResponse)

		deleteUserPoolDomainResponse := &cognito.DeleteUserPoolDomainOutput{}
		deleteUserPoolDomainRequest := &cognito.DeleteUserPoolDomainInput{
			Domain:     &authDomain,
			UserPoolId: &userPoolId,
		}

		log.Printf("Cognito DeleteUserPoolDomain Request: %+v", deleteUserPoolDomainRequest)

		deleteUserPoolDomainResponse, err = cognitoSvc.DeleteUserPoolDomain(deleteUserPoolDomainRequest)
		if err != nil {
			util.LogAWSError("Cognito DeleteUserPoolDomain Error: %+v", err)
			return
		}

		log.Printf("Cognito DeleteUserPoolDomain Response: %+v", deleteUserPoolDomainResponse)
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
