package question

import "errors"

var (
	ErrQuestionNotFound    = errors.New("Question not found")
	ErrInvalidRequestBody  = errors.New("invalid request body")
	ErrMultipleJSONValues  = errors.New("request body must contain one JSON object")
	ErrRequestBodyTooLarge = errors.New("request body must not exceed 1 MiB")
	ErrDatabaseUnavailable = errors.New("database temporarily unavailable")
	ErrPublishAttempt      = errors.New("answer saved, but Kafka publication could not be confirmed")
	ErrInternalServer      = errors.New("internal server error")
)
