package errors

import "net/http"

// AppError represents a structured application error with an HTTP status code,
// a machine-readable code, and a safe human-readable message.
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func NewAppError(status int, code, message string) *AppError {
	return &AppError{Code: code, Message: message, Status: status}
}

func (e *AppError) Error() string {
	return e.Message
}

// Common error codes reused across modules.
const (
	CodeNotFound         = "NOT_FOUND"
	CodeValidation       = "VALIDATION_ERROR"
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeConflict         = "CONFLICT"
	CodeRateLimited      = "RATE_LIMITED"
	CodeInternal         = "INTERNAL_ERROR"
)

func NotFound(message string) *AppError {
	return NewAppError(http.StatusNotFound, CodeNotFound, message)
}

func Validation(message string) *AppError {
	return NewAppError(http.StatusBadRequest, CodeValidation, message)
}

func Unauthorized(message string) *AppError {
	return NewAppError(http.StatusUnauthorized, CodeUnauthorized, message)
}

func Conflict(message string) *AppError {
	return NewAppError(http.StatusConflict, CodeConflict, message)
}

func Internal(message string) *AppError {
	return NewAppError(http.StatusInternalServerError, CodeInternal, message)
}
