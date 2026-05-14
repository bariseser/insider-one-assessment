package utils

import (
	"errors"

	"insider-one-assessment/internal/constants"
)

type ErrorBag struct {
	Code    string `json:"code"`
	Cause   error  `json:"cause"`
	Message string `json:"message"`
}

func NewErrorBag(code string, cause error, message string) *ErrorBag {
	if cause == nil {
		cause = errors.New(message)
	}
	return &ErrorBag{Code: code, Cause: cause, Message: message}
}

func (e ErrorBag) GetCode() string {
	return e.Code
}

func (e ErrorBag) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return e.Message
}

func (e ErrorBag) GetMessage() string {
	return e.Message
}

func (e ErrorBag) GetStatusCode() int {
	return constants.HTTPStatusByErrorCode(e.Code)
}
