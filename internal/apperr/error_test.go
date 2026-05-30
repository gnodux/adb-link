package apperr

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestNew(t *testing.T) {
	e := New(CodeBadRequest, http.StatusBadRequest, "invalid input")
	if e.Code != CodeBadRequest {
		t.Errorf("Code = %q, want %q", e.Code, CodeBadRequest)
	}
	if e.Status != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", e.Status, http.StatusBadRequest)
	}
	if e.Msg != "invalid input" {
		t.Errorf("Msg = %q, want %q", e.Msg, "invalid input")
	}
	if e.Cause != nil {
		t.Errorf("Cause = %v, want nil", e.Cause)
	}
}

func TestNewf(t *testing.T) {
	e := Newf(CodeNotFound, http.StatusNotFound, "user %s not found", "alice")
	if e.Msg != "user alice not found" {
		t.Errorf("Msg = %q", e.Msg)
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("connection refused")
	e := Wrap(CodeUnavailable, http.StatusServiceUnavailable, cause, "db down")
	if e.Cause != cause {
		t.Errorf("Cause = %v, want %v", e.Cause, cause)
	}
	if !errors.Is(e, cause) {
		t.Error("errors.Is should match cause")
	}
}

func TestWrap_Unwrap(t *testing.T) {
	cause := errors.New("root")
	e := Wrap(CodeInternal, 500, cause, "wrapped")
	if e.Unwrap() != cause {
		t.Error("Unwrap should return cause")
	}
}

func TestError_String_WithCause(t *testing.T) {
	cause := errors.New("timeout")
	e := Wrap(CodeInternal, 500, cause, "request failed")
	s := e.Error()
	if s != "internal: request failed: timeout" {
		t.Errorf("Error() = %q", s)
	}
}

func TestError_String_WithoutCause(t *testing.T) {
	e := New(CodeBadRequest, 400, "bad input")
	if got := e.Error(); got != "bad_request: bad input" {
		t.Errorf("Error() = %q", got)
	}
}

func TestError_NilReceiver(t *testing.T) {
	var e *Error
	if got := e.Error(); got != "" {
		t.Errorf("nil Error.Error() = %q, want empty", got)
	}
	if got := e.Unwrap(); got != nil {
		t.Errorf("nil Error.Unwrap() = %v, want nil", got)
	}
}

func TestUnauthorized(t *testing.T) {
	e := Unauthorized("no token")
	if e.Code != CodeUnauthorized || e.Status != 401 {
		t.Errorf("got Code=%s Status=%d", e.Code, e.Status)
	}
}

func TestForbidden(t *testing.T) {
	e := Forbidden("denied")
	if e.Code != CodeForbidden || e.Status != 403 {
		t.Errorf("got Code=%s Status=%d", e.Code, e.Status)
	}
}

func TestNotFound(t *testing.T) {
	e := NotFound("missing")
	if e.Code != CodeNotFound || e.Status != 404 {
		t.Errorf("got Code=%s Status=%d", e.Code, e.Status)
	}
}

func TestBadRequest(t *testing.T) {
	e := BadRequest("invalid")
	if e.Code != CodeBadRequest || e.Status != 400 {
		t.Errorf("got Code=%s Status=%d", e.Code, e.Status)
	}
}

func TestConflict(t *testing.T) {
	e := Conflict("exists")
	if e.Code != CodeConflict || e.Status != 409 {
		t.Errorf("got Code=%s Status=%d", e.Code, e.Status)
	}
}

func TestInternal(t *testing.T) {
	e := Internal("boom")
	if e.Code != CodeInternal || e.Status != 500 {
		t.Errorf("got Code=%s Status=%d", e.Code, e.Status)
	}
}

func TestUnavailable(t *testing.T) {
	e := Unavailable("down")
	if e.Code != CodeUnavailable || e.Status != 503 {
		t.Errorf("got Code=%s Status=%d", e.Code, e.Status)
	}
}

func TestAs_AppError(t *testing.T) {
	e := BadRequest("oops")
	got := As(e)
	if got == nil || got.Code != CodeBadRequest {
		t.Errorf("As returned %v", got)
	}
}

func TestAs_WrappedError(t *testing.T) {
	e := NotFound("gone")
	wrapped := fmt.Errorf("layer: %w", e)
	got := As(wrapped)
	if got == nil || got.Code != CodeNotFound {
		t.Errorf("As returned %v", got)
	}
}

func TestAs_StdError_ReturnsNil(t *testing.T) {
	err := errors.New("plain error")
	if got := As(err); got != nil {
		t.Errorf("As should return nil for std error, got %v", got)
	}
}
