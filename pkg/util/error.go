package util

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"log"
)

type LambdaError struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func NewError(message string, status int) error {
	return &LambdaError{Message: message, Status: status}
}

func (le *LambdaError) Error() string {
	return le.Message
}

func LogAWSError(format string, err error, v ...interface{}) {
	if aerr, ok := err.(awserr.Error); ok {
		log.Printf(format, aerr, v)
	} else {
		log.Printf(format, err, v)
	}
}