package app

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func New(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func IsAppError(err error) (*AppError, bool) {
	e, ok := err.(*AppError)
	return e, ok
}

func NotFound(msg string) *AppError {
	return New(http.StatusNotFound, msg, nil)
}

func Internal(msg string, err error) *AppError {
	return New(http.StatusInternalServerError, msg, err)
}

func BadRequest(msg string) *AppError {
	return New(http.StatusBadRequest, msg, nil)
}
