package question

import (
	"context"
	"errors"
	"net/http"
)

var (
	ErrQuestionNotFound    = errors.New("Question not found")
	ErrInvalidRequestBody  = errors.New("invalid request body")
	ErrMultipleJSONValues  = errors.New("request body must contain one JSON object")
	ErrRequestBodyTooLarge = errors.New("request body must not exceed 1 MiB")
	ErrDatabaseUnavailable = errors.New("database temporarily unavailable")
	ErrInternalServer      = errors.New("internal server error")
)

// map for handleServiceError
var serviceErrorResponses = map[error]serviceErrorResponse{
	ErrQuestionNotFound:      {http.StatusNotFound, ErrQuestionNotFound.Error()},
	ErrInvalidRequestBody:    {http.StatusBadRequest, ErrInvalidRequestBody.Error()},
	ErrMultipleJSONValues:    {http.StatusBadRequest, ErrMultipleJSONValues.Error()},
	ErrRequestBodyTooLarge:   {http.StatusRequestEntityTooLarge, ErrRequestBodyTooLarge.Error()},
	ErrDatabaseUnavailable:   {http.StatusServiceUnavailable, ErrDatabaseUnavailable.Error()},
	ErrInternalServer:        {http.StatusInternalServerError, ErrInternalServer.Error()},
	context.Canceled:         {http.StatusRequestTimeout, "request canceled"},
	context.DeadlineExceeded: {http.StatusGatewayTimeout, "request timed out"},
}
