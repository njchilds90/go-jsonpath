package jsonpath

import "fmt"

// ErrorCode identifies the category of a JSONPath error.
type ErrorCode int

const (
	// ErrInvalidPath indicates a malformed JSONPath expression.
	ErrInvalidPath ErrorCode = iota + 1
	// ErrInvalidJSON indicates the input JSON could not be parsed.
	ErrInvalidJSON
	// ErrInvalidFilter indicates a malformed filter expression.
	ErrInvalidFilter
	// ErrInvalidInput indicates invalid parameters (nil context, etc.).
	ErrInvalidInput
	// ErrKeyNotFound indicates a required key was not found (strict mode).
	ErrKeyNotFound
	// ErrIndexOutOfBounds indicates an array index was out of range (strict mode).
	ErrIndexOutOfBounds
	// ErrTypeMismatch indicates the node type did not match expectation (strict mode).
	ErrTypeMismatch
	// ErrMaxDepthExceeded indicates the recursive descent exceeded the configured depth limit.
	ErrMaxDepthExceeded
	// ErrCancelled indicates the context was cancelled.
	ErrCancelled
)

// Error is the structured error type returned by all jsonpath operations.
// Use Code to programmatically distinguish error categories.
type Error struct {
	// Code identifies the error category.
	Code ErrorCode
	// Message is a human-readable description.
	Message string
	// Cause is the underlying error, if any.
	Cause error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("jsonpath: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("jsonpath: %s", e.Message)
}

// Unwrap returns the underlying cause, supporting errors.Is and errors.As chains.
func (e *Error) Unwrap() error {
	return e.Cause
}

// IsPathError returns true if err is a jsonpath path syntax error.
func IsPathError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrInvalidPath
	}
	return false
}

// IsJSONError returns true if err is a JSON parsing error.
func IsJSONError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrInvalidJSON
	}
	return false
}

// IsFilterError returns true if err is a filter expression error.
func IsFilterError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrInvalidFilter
	}
	return false
}

// IsNotFound returns true if err indicates a missing key or out-of-bounds index (strict mode).
func IsNotFound(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrKeyNotFound || e.Code == ErrIndexOutOfBounds
	}
	return false
}

// IsCancelled returns true if err is a context cancellation error.
func IsCancelled(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrCancelled
	}
	return false
}
