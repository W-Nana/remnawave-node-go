package api

import (
	"time"
)

type SuccessResponse struct {
	Response interface{} `json:"response"`
}

type ErrorResponse struct {
	Timestamp string `json:"timestamp"`
	Path      string `json:"path"`
	Message   string `json:"message"`
	ErrorCode string `json:"errorCode"`
}

type ValidationError struct {
	Path    []string `json:"path"`
	Message string   `json:"message"`
}

type ValidationErrorResponse struct {
	StatusCode int               `json:"statusCode"`
	Message    string            `json:"message"`
	Errors     []ValidationError `json:"errors"`
}

func NewSuccessResponse(data interface{}) SuccessResponse {
	return SuccessResponse{Response: data}
}

func NewErrorResponse(path, message, errorCode string) ErrorResponse {
	return ErrorResponse{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Path:      path,
		Message:   message,
		ErrorCode: errorCode,
	}
}

func NewValidationErrorResponse(errors []ValidationError) ValidationErrorResponse {
	return ValidationErrorResponse{
		StatusCode: 400,
		Message:    "Validation failed",
		Errors:     errors,
	}
}
