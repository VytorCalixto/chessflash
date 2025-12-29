package errors

import "fmt"

// Error codes
const (
	ErrCodeNotFound      = "NOT_FOUND"
	ErrCodeValidation    = "VALIDATION_ERROR"
	ErrCodeInternal      = "INTERNAL_ERROR"
	ErrCodeBadRequest    = "BAD_REQUEST"
)

// AppError represents an application error with HTTP status code and error code
type AppError struct {
	Code    string // Error code (e.g., "NOT_FOUND", "VALIDATION_ERROR")
	Message string // Human-readable error message
	Status  int    // HTTP status code
	Err     error  // Wrapped underlying error (optional)
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for error wrapping support
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewNotFoundError creates a new NOT_FOUND error
func NewNotFoundError(resource string, id interface{}) *AppError {
	return &AppError{
		Code:    ErrCodeNotFound,
		Message: fmt.Sprintf("%s not found: %v", resource, id),
		Status:  404,
	}
}

// NewValidationError creates a new VALIDATION_ERROR
func NewValidationError(field string, reason string) *AppError {
	return &AppError{
		Code:    ErrCodeValidation,
		Message: fmt.Sprintf("validation failed for %s: %s", field, reason),
		Status:  400,
	}
}

// NewInternalError creates a new INTERNAL_ERROR
func NewInternalError(err error) *AppError {
	return &AppError{
		Code:    ErrCodeInternal,
		Message: "internal server error",
		Status:  500,
		Err:     err,
	}
}

// NewBadRequestError creates a new BAD_REQUEST error
func NewBadRequestError(message string) *AppError {
	return &AppError{
		Code:    ErrCodeBadRequest,
		Message: message,
		Status:  400,
	}
}
