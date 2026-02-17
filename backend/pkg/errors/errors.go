package errors

import "fmt"

// ErrorCategory represents the category of a domain error.
type ErrorCategory string

const (
	CategoryNotFound     ErrorCategory = "not_found"
	CategoryValidation   ErrorCategory = "validation"
	CategoryConflict     ErrorCategory = "conflict"
	CategoryUnauthorized ErrorCategory = "unauthorized"
	CategoryForbidden    ErrorCategory = "forbidden"
	CategoryInternal     ErrorCategory = "internal"
	CategoryInvalidState ErrorCategory = "invalid_state"
)

// DomainError represents a structured error from the domain layer.
type DomainError struct {
	Category ErrorCategory
	Code     string
	Message  string
	Cause    error
}

func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error {
	return e.Cause
}

// NewNotFound creates a not-found domain error.
func NewNotFound(resource string, id interface{}) *DomainError {
	return &DomainError{
		Category: CategoryNotFound,
		Code:     fmt.Sprintf("%s_NOT_FOUND", toUpperSnake(resource)),
		Message:  fmt.Sprintf("%s with ID %v not found", resource, id),
	}
}

// NewValidation creates a validation domain error.
func NewValidation(field, reason string) *DomainError {
	return &DomainError{
		Category: CategoryValidation,
		Code:     "VALIDATION_ERROR",
		Message:  fmt.Sprintf("validation failed on field '%s': %s", field, reason),
	}
}

// NewConflict creates a conflict domain error.
func NewConflict(resource, value string) *DomainError {
	return &DomainError{
		Category: CategoryConflict,
		Code:     fmt.Sprintf("%s_ALREADY_EXISTS", toUpperSnake(resource)),
		Message:  fmt.Sprintf("%s '%s' already exists", resource, value),
	}
}

// NewForbidden creates a forbidden domain error.
func NewForbidden(message string) *DomainError {
	return &DomainError{
		Category: CategoryForbidden,
		Code:     "FORBIDDEN",
		Message:  message,
	}
}

// NewUnauthorized creates an unauthorized domain error.
func NewUnauthorized(message string) *DomainError {
	return &DomainError{
		Category: CategoryUnauthorized,
		Code:     "UNAUTHORIZED",
		Message:  message,
	}
}

// NewInvalidState creates an invalid state transition domain error.
func NewInvalidState(code, message string) *DomainError {
	return &DomainError{
		Category: CategoryInvalidState,
		Code:     code,
		Message:  message,
	}
}

// NewInternal creates an internal domain error.
func NewInternal(message string, cause error) *DomainError {
	return &DomainError{
		Category: CategoryInternal,
		Code:     "INTERNAL_ERROR",
		Message:  message,
		Cause:    cause,
	}
}

func toUpperSnake(s string) string {
	result := make([]byte, 0, len(s)+4)
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(c))
		} else if c >= 'a' && c <= 'z' {
			result = append(result, byte(c-32))
		} else {
			result = append(result, byte(c))
		}
	}
	return string(result)
}
