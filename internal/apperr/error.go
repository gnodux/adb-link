// Package apperr provides typed errors carrying HTTP status codes plus
// helpers to extract them at the API edge. Domain code (services, MCP) raises
// apperr.Error values; HTTP handlers translate them into real 4xx/5xx
// responses instead of the legacy 200-with-success:false envelope.
package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// Code is a stable machine-readable identifier for an error class.
type Code string

const (
	CodeUnauthorized Code = "unauthorized"
	CodeForbidden    Code = "forbidden"
	CodeNotFound     Code = "not_found"
	CodeBadRequest   Code = "bad_request"
	CodeConflict     Code = "conflict"
	CodeInternal     Code = "internal"
	CodeUnavailable  Code = "unavailable"
)

// Error pairs a stable code, an HTTP status, and a human message. The
// underlying cause (if any) is preserved for logging via errors.Unwrap.
type Error struct {
	Code   Code
	Status int
	Msg    string
	Cause  error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Msg, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Msg)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// New builds an Error with the given code/status/message.
func New(code Code, status int, msg string) *Error {
	return &Error{Code: code, Status: status, Msg: msg}
}

// Newf is like New with formatting.
func Newf(code Code, status int, format string, args ...any) *Error {
	return &Error{Code: code, Status: status, Msg: fmt.Sprintf(format, args...)}
}

// Wrap attaches a cause to a new Error.
func Wrap(code Code, status int, cause error, msg string) *Error {
	return &Error{Code: code, Status: status, Msg: msg, Cause: cause}
}

// Convenience constructors for the common categories.
func Unauthorized(msg string) *Error { return New(CodeUnauthorized, http.StatusUnauthorized, msg) }
func Forbidden(msg string) *Error    { return New(CodeForbidden, http.StatusForbidden, msg) }
func NotFound(msg string) *Error     { return New(CodeNotFound, http.StatusNotFound, msg) }
func BadRequest(msg string) *Error   { return New(CodeBadRequest, http.StatusBadRequest, msg) }
func Conflict(msg string) *Error     { return New(CodeConflict, http.StatusConflict, msg) }
func Internal(msg string) *Error     { return New(CodeInternal, http.StatusInternalServerError, msg) }
func Unavailable(msg string) *Error  { return New(CodeUnavailable, http.StatusServiceUnavailable, msg) }

// As is a convenience over errors.As that returns the typed *Error or nil.
func As(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}
