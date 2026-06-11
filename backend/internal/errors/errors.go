// Package errors provides standardized error types and handling
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError represents an application error with context
type AppError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	StatusCode int                    `json:"-"`
	Internal   error                  `json:"-"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Internal)
	}
	return e.Message
}

// Unwrap implements error unwrapping for error chains
func (e *AppError) Unwrap() error {
	return e.Internal
}

// Common error codes
const (
	ErrCodeInternal          = "INTERNAL_ERROR"
	ErrCodeBadRequest        = "BAD_REQUEST"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeForbidden         = "FORBIDDEN"
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeConflict          = "CONFLICT"
	ErrCodeValidation        = "VALIDATION_ERROR"
	ErrCodeRateLimit         = "RATE_LIMIT_EXCEEDED"
	ErrCodeDatabaseError     = "DATABASE_ERROR"
	ErrCodeExternalService   = "EXTERNAL_SERVICE_ERROR"
)

// New creates a new AppError
func New(code, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, code, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Internal:   err,
	}
}

// WithDetails adds details to an AppError
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	e.Details = details
	return e
}

// Predefined error constructors

// Internal creates an internal server error
func Internal(err error, message string) *AppError {
	if message == "" {
		message = "An internal error occurred"
	}
	return Wrap(err, ErrCodeInternal, message, http.StatusInternalServerError)
}

// BadRequest creates a bad request error
func BadRequest(message string) *AppError {
	if message == "" {
		message = "Invalid request"
	}
	return New(ErrCodeBadRequest, message, http.StatusBadRequest)
}

// Unauthorized creates an unauthorized error
func Unauthorized(message string) *AppError {
	if message == "" {
		message = "Authentication required"
	}
	return New(ErrCodeUnauthorized, message, http.StatusUnauthorized)
}

// Forbidden creates a forbidden error
func Forbidden(message string) *AppError {
	if message == "" {
		message = "Access forbidden"
	}
	return New(ErrCodeForbidden, message, http.StatusForbidden)
}

// NotFound creates a not found error
func NotFound(resource string) *AppError {
	message := "Resource not found"
	if resource != "" {
		message = fmt.Sprintf("%s not found", resource)
	}
	return New(ErrCodeNotFound, message, http.StatusNotFound)
}

// Conflict creates a conflict error
func Conflict(message string) *AppError {
	if message == "" {
		message = "Resource conflict"
	}
	return New(ErrCodeConflict, message, http.StatusConflict)
}

// Validation creates a validation error
func Validation(message string, details map[string]interface{}) *AppError {
	if message == "" {
		message = "Validation failed"
	}
	return New(ErrCodeValidation, message, http.StatusBadRequest).WithDetails(details)
}

// RateLimit creates a rate limit error
func RateLimit() *AppError {
	return New(ErrCodeRateLimit, "Rate limit exceeded", http.StatusTooManyRequests)
}

// Database creates a database error
func Database(err error, operation string) *AppError {
	message := "Database operation failed"
	if operation != "" {
		message = fmt.Sprintf("Database %s failed", operation)
	}
	return Wrap(err, ErrCodeDatabaseError, message, http.StatusInternalServerError)
}

// ExternalService creates an external service error
func ExternalService(err error, service string) *AppError {
	message := "External service error"
	if service != "" {
		message = fmt.Sprintf("%s service error", service)
	}
	return Wrap(err, ErrCodeExternalService, message, http.StatusBadGateway)
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}

// GetAppError extracts AppError from error chain
func GetAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}

// GetStatusCode extracts HTTP status code from error
func GetStatusCode(err error) int {
	if appErr := GetAppError(err); appErr != nil {
		return appErr.StatusCode
	}
	return http.StatusInternalServerError
}
