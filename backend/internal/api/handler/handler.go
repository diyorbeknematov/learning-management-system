// Package handler holds the HTTP handlers. A handler reads the request, calls a
// service and writes the answer; the rules live in the services.
package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/diyorbeknematov/lms/internal/api/response"
	"github.com/diyorbeknematov/lms/internal/api/validation"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// maxBodySize is the largest request body the API reads. Files do not go
// through the API (they are uploaded to the storage), so it can be small.
const maxBodySize = 1 << 20

// bindJSON reads the body into dst and validates it. When the body is wrong it
// writes the 400 answer and returns false; the handler then just returns.
func bindJSON(c *gin.Context, dst any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodySize)

	return bindWith(c, dst, c.ShouldBindJSON)
}

// bindQuery reads the query parameters into dst and validates them.
func bindQuery(c *gin.Context, dst any) bool {
	return bindWith(c, dst, c.ShouldBindQuery)
}

func bindWith(c *gin.Context, dst any, bind func(any) error) bool {
	err := bind(dst)
	if err == nil {
		return true
	}

	if fields := validation.Fields(err); fields != nil {
		response.ValidationFailed(c, fields)

		return false
	}

	var tooLarge *http.MaxBytesError

	switch {
	case errors.As(err, &tooLarge):
		response.Error(c, http.StatusRequestEntityTooLarge, response.CodeInvalidInput, "the request body is too large")
	case errors.Is(err, io.EOF):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidInput, "the request body is empty")
	default:
		response.Error(c, http.StatusBadRequest, response.CodeInvalidInput, "the request is malformed")
	}

	return false
}

// pathID reads an id from the address (/courses/:courseId). When it is not a
// valid id it writes the 400 answer and returns false.
func pathID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		response.ValidationFailed(c, map[string]string{name: "must be a valid id"})

		return uuid.Nil, false
	}

	return id, true
}

// optionalID turns a query parameter that was already checked as a UUID into an
// id; a missing parameter stays nil.
func optionalID(value *string) *uuid.UUID {
	if value == nil || *value == "" {
		return nil
	}

	id, err := uuid.Parse(*value)
	if err != nil {
		return nil
	}

	return &id
}
