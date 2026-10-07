// Package response writes every HTTP answer of the API in one format:
//
//	success: {"success": true, "data": ...}
//	failure: {"success": false, "error": {"code": "NOT_FOUND", "message": "..."}}
package response

import (
	"log/slog"
	"net/http"

	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/gin-gonic/gin"
)

const (
	CodeInvalidInput    = "INVALID_INPUT"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeForbidden       = "FORBIDDEN"
	CodeNotFound        = "NOT_FOUND"
	CodeTooManyRequests = "TOO_MANY_REQUESTS"
	CodeInternal        = "INTERNAL"
)

// Success is the body of a successful answer; the type of Data depends on the
// route (the API documentation shows it).
type Success struct {
	Success bool `json:"success"`
	Data    any  `json:"data,omitempty"`
}

// Failure is the body of a failed answer.
type Failure struct {
	Success bool      `json:"success"`
	Error   ErrorBody `json:"error"`
}

// ErrorBody is the "error" part of a failed answer. Fields tells which input
// field is wrong and why.
type ErrorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// OK answers 200 with data.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Success{Success: true, Data: data})
}

// Created answers 201 with the new resource.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Success{Success: true, Data: data})
}

// MessageData is the data of an answer that only has a short text.
type MessageData struct {
	Message string `json:"message"`
}

// Message answers 200 with a short text, for actions that return nothing else.
func Message(c *gin.Context, message string) {
	OK(c, MessageData{Message: message})
}

// NoContent answers 204 without a body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Error answers with a failure and stops the handlers that are left.
func Error(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, Failure{Success: false, Error: ErrorBody{Code: code, Message: message}})
}

// ValidationFailed answers 400 with the wrong fields.
func ValidationFailed(c *gin.Context, fields map[string]string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, Failure{
		Success: false,
		Error:   ErrorBody{Code: CodeInvalidInput, Message: "validation failed", Fields: fields},
	})
}

// Fail answers for an error that came from a service. The code and the message
// of an application error are for the client; an error that is not the
// client's fault (500) is logged with everything we know and the client gets
// only a general message.
func Fail(c *gin.Context, err error) {
	appErr, ok := apperror.As(err)
	if !ok {
		logInternal(c, err)
		Error(c, http.StatusInternalServerError, CodeInternal, "internal server error")

		return
	}

	status := appErr.HTTPStatus()

	if status >= http.StatusInternalServerError {
		logInternal(c, err)
		Error(c, status, string(appErr.Code), "internal server error")

		return
	}

	Error(c, status, string(appErr.Code), appErr.Message)
}

func logInternal(c *gin.Context, err error) {
	slog.ErrorContext(
		c.Request.Context(),
		"request failed",
		"method", c.Request.Method,
		"path", c.FullPath(),
		"request_id", c.GetString("request_id"),
		"error", err,
	)
}
