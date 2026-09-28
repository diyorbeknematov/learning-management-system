package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

type ErrorCode string

const (
	CodeNotFound     ErrorCode = "NOT_FOUND"
	CodeInvalidInput ErrorCode = "INVALID_INPUT"
	CodeUnauthorized ErrorCode = "UNAUTHORIZED"
	CodeForbidden    ErrorCode = "FORBIDDEN"
	CodeConflict     ErrorCode = "CONFLICT"
	CodeInternal     ErrorCode = "INTERNAL"
)

type AppError struct {
	Code    ErrorCode
	Layer   string
	Op      string
	Message string
	Err     error
}

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrUnauthorized = errors.New("unauthorized")
	ErrUserNotFound = errors.New("user not found")
	ErrEmailExists  = errors.New("email already exists")
)

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.Layer, e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Layer, e.Op, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) HTTPStatus() int {
	switch e.Code {
	case CodeNotFound:
		return http.StatusNotFound
	case CodeInvalidInput:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}


func Wrap(code ErrorCode, layer, op, message string, err error) error {
	return &AppError{
		Code:    code,
		Layer:   layer,
		Op:      op,
		Message: message,
		Err:     err,
	}
}

func As(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}