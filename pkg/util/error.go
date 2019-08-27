package util

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