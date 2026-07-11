package domain

import "fmt"

// Error code constants for domain errors.
const (
	ErrCodeInvalidInput      = "INVALID_INPUT"
	ErrCodeFileNotFound      = "FILE_NOT_FOUND"
	ErrCodeParseError        = "PARSE_ERROR"
	ErrCodeAnalysisError     = "ANALYSIS_ERROR"
	ErrCodeConfigError       = "CONFIG_ERROR"
	ErrCodeOutputError       = "OUTPUT_ERROR"
	ErrCodeUnsupportedFormat = "UNSUPPORTED_FORMAT"
	ErrCodeValidation        = "VALIDATION_ERROR"
	ErrCodeTimeout           = "TIMEOUT"
	ErrCodeCancelled         = "CANCELLED"
	ErrCodeNotImplemented    = "NOT_IMPLEMENTED"
	ErrCodeInternal          = "INTERNAL_ERROR"
)

// DomainError represents a structured error with code, message, and optional cause.
type DomainError struct {
	Code    string
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause for errors.Is/As support.
func (e *DomainError) Unwrap() error {
	return e.Cause
}

// NewDomainError creates a new DomainError with the given code, message, and optional cause.
func NewDomainError(code, message string, cause error) *DomainError {
	return &DomainError{Code: code, Message: message, Cause: cause}
}

// NewInvalidInputError creates an error for invalid input.
func NewInvalidInputError(message string, cause error) *DomainError {
	return NewDomainError(ErrCodeInvalidInput, message, cause)
}

// NewFileNotFoundError creates an error for missing files.
func NewFileNotFoundError(message string, cause error) *DomainError {
	return NewDomainError(ErrCodeFileNotFound, message, cause)
}

// NewParseError creates an error for parse failures.
func NewParseError(message string, cause error) *DomainError {
	return NewDomainError(ErrCodeParseError, message, cause)
}

// NewAnalysisError creates an error for analysis failures.
func NewAnalysisError(message string, cause error) *DomainError {
	return NewDomainError(ErrCodeAnalysisError, message, cause)
}

// NewConfigError creates an error for configuration issues.
func NewConfigError(message string, cause error) *DomainError {
	return NewDomainError(ErrCodeConfigError, message, cause)
}

// NewOutputError creates an error for output failures.
func NewOutputError(message string, cause error) *DomainError {
	return NewDomainError(ErrCodeOutputError, message, cause)
}

// NewUnsupportedFormatError creates an error for unsupported formats.
func NewUnsupportedFormatError(message string, cause error) *DomainError {
	return NewDomainError(ErrCodeUnsupportedFormat, message, cause)
}

// NewValidationError creates an error for validation failures.
func NewValidationError(message string, cause error) *DomainError {
	return NewDomainError(ErrCodeValidation, message, cause)
}

// NewTimeoutError creates an error for timeout conditions.
func NewTimeoutError(message string, cause error) *DomainError {
	return NewDomainError(ErrCodeTimeout, message, cause)
}

// NewCancelledError creates an error for cancelled operations.
func NewCancelledError(message string, cause error) *DomainError {
	return NewDomainError(ErrCodeCancelled, message, cause)
}

// NewNotImplementedError creates an error for unimplemented features.
func NewNotImplementedError(message string, cause error) *DomainError {
	return NewDomainError(ErrCodeNotImplemented, message, cause)
}

// NewInternalError creates an error for internal failures.
func NewInternalError(message string, cause error) *DomainError {
	return NewDomainError(ErrCodeInternal, message, cause)
}
