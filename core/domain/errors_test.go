package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestDomainError_Error(t *testing.T) {
	err := NewDomainError(ErrCodeInvalidInput, "bad input", nil)
	got := err.Error()
	if !strings.Contains(got, ErrCodeInvalidInput) {
		t.Errorf("expected error code %q in %q", ErrCodeInvalidInput, got)
	}
	if !strings.Contains(got, "bad input") {
		t.Errorf("expected message in error string")
	}
}

func TestDomainError_ErrorWithCause(t *testing.T) {
	cause := errors.New("underlying")
	err := NewDomainError(ErrCodeParseError, "parse failed", cause)
	got := err.Error()
	if !strings.Contains(got, "underlying") {
		t.Errorf("expected cause in error string: %q", got)
	}
}

func TestDomainError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := NewDomainError(ErrCodeInternal, "internal", cause)
	if !errors.Is(err, cause) {
		t.Errorf("Unwrap should return the cause")
	}
}

func TestDomainError_UnwrapNil(t *testing.T) {
	err := NewDomainError(ErrCodeInternal, "internal", nil)
	if err.Unwrap() != nil {
		t.Errorf("Unwrap should return nil when no cause")
	}
}

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(string, error) *DomainError
		wantCode string
	}{
		{"InvalidInput", NewInvalidInputError, ErrCodeInvalidInput},
		{"FileNotFound", NewFileNotFoundError, ErrCodeFileNotFound},
		{"ParseError", NewParseError, ErrCodeParseError},
		{"AnalysisError", NewAnalysisError, ErrCodeAnalysisError},
		{"ConfigError", NewConfigError, ErrCodeConfigError},
		{"OutputError", NewOutputError, ErrCodeOutputError},
		{"UnsupportedFormat", NewUnsupportedFormatError, ErrCodeUnsupportedFormat},
		{"Validation", NewValidationError, ErrCodeValidation},
		{"Timeout", NewTimeoutError, ErrCodeTimeout},
		{"Cancelled", NewCancelledError, ErrCodeCancelled},
		{"NotImplemented", NewNotImplementedError, ErrCodeNotImplemented},
		{"Internal", NewInternalError, ErrCodeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn("test message", nil)
			if err.Code != tt.wantCode {
				t.Errorf("got code %q, want %q", err.Code, tt.wantCode)
			}
			if err.Message != "test message" {
				t.Errorf("got message %q, want %q", err.Message, "test message")
			}
		})
	}
}
