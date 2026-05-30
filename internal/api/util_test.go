package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gnodux/adb-link/internal/apperr"
	"github.com/gnodux/adb-link/internal/models"
)

// ---------------------------------------------------------------------------
// WriteJSON
// ---------------------------------------------------------------------------

func TestWriteJSON_SetsContentType(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusOK, map[string]string{"ok": "true"})

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Fatalf("expected Content-Type application/json; charset=utf-8, got %q", ct)
	}
}

func TestWriteJSON_SetsStatusCode(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusCreated, map[string]string{"id": "1"})

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
}

func TestWriteJSON_EncodesPayload(t *testing.T) {
	payload := map[string]any{"name": "alice", "age": float64(30)}
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusOK, payload)

	var got map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got["name"] != "alice" {
		t.Fatalf("expected name=alice, got %v", got["name"])
	}
	if got["age"] != float64(30) {
		t.Fatalf("expected age=30, got %v", got["age"])
	}
}

// ---------------------------------------------------------------------------
// WriteOK
// ---------------------------------------------------------------------------

func TestWriteOK_SuccessEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteOK(rec, map[string]string{"result": "ok"})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body models.APIResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !body.Success {
		t.Fatal("expected success=true")
	}
	if body.Error != "" {
		t.Fatalf("expected empty error field, got %q", body.Error)
	}
	// body.Data is any — re-marshal to check.
	data, _ := json.Marshal(body.Data)
	if !strings.Contains(string(data), `"result"`) {
		t.Fatalf("expected data to contain result key, got %s", data)
	}
}

// ---------------------------------------------------------------------------
// WriteError
// ---------------------------------------------------------------------------

func TestWriteError_BadRequestEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, "bad input")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	var body models.APIResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Success {
		t.Fatal("expected success=false")
	}
	if body.Error != "bad input" {
		t.Fatalf("expected error 'bad input', got %q", body.Error)
	}
}

// ---------------------------------------------------------------------------
// WriteErrorStatus
// ---------------------------------------------------------------------------

func TestWriteErrorStatus_CustomStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteErrorStatus(rec, http.StatusConflict, "already exists")

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
	var body models.APIResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Success {
		t.Fatal("expected success=false")
	}
	if body.Error != "already exists" {
		t.Fatalf("expected error 'already exists', got %q", body.Error)
	}
}

// ---------------------------------------------------------------------------
// WriteAppError
// ---------------------------------------------------------------------------

func TestWriteAppError_AppError(t *testing.T) {
	appErr := apperr.NotFound("item missing")
	rec := httptest.NewRecorder()
	WriteAppError(rec, appErr)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["success"] != false {
		t.Fatalf("expected success=false, got %v", body["success"])
	}
	if body["error"] != "item missing" {
		t.Fatalf("expected error 'item missing', got %v", body["error"])
	}
	if body["code"] != string(apperr.CodeNotFound) {
		t.Fatalf("expected code 'not_found', got %v", body["code"])
	}
}

func TestWriteAppError_StdError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteAppError(rec, errors.New("disk full"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	var body models.APIResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Success {
		t.Fatal("expected success=false")
	}
	if body.Error != "disk full" {
		t.Fatalf("expected error 'disk full', got %q", body.Error)
	}
}

func TestWriteAppError_NilError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteAppError(rec, nil)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	var body models.APIResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Success {
		t.Fatal("expected success=false")
	}
	if body.Error != "unknown error" {
		t.Fatalf("expected error 'unknown error', got %q", body.Error)
	}
}

// ---------------------------------------------------------------------------
// DecodeJSON
// ---------------------------------------------------------------------------

func TestDecodeJSON_ValidBody(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	body := `{"name":"bob","age":25}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	var p payload
	if err := DecodeJSON(req, &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "bob" {
		t.Fatalf("expected name 'bob', got %q", p.Name)
	}
	if p.Age != 25 {
		t.Fatalf("expected age 25, got %d", p.Age)
	}
}

func TestDecodeJSON_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{invalid"))
	var p struct{ Name string }
	err := DecodeJSON(req, &p)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// ---------------------------------------------------------------------------
// UserFromRequest / UserNameFromRequest
// ---------------------------------------------------------------------------

func TestUserFromRequest_WithUser(t *testing.T) {
	user := &models.AuthUser{Name: "carol", APIKey: "key-carol"}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithUser(req.Context(), user))

	got := UserFromRequest(req)
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.Name != "carol" {
		t.Fatalf("expected name 'carol', got %q", got.Name)
	}
}

func TestUserFromRequest_NoUser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	got := UserFromRequest(req)
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestUserNameFromRequest_WithUser(t *testing.T) {
	user := &models.AuthUser{Name: "dave", APIKey: "key-dave"}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithUser(req.Context(), user))

	name := UserNameFromRequest(req)
	if name != "dave" {
		t.Fatalf("expected 'dave', got %q", name)
	}
}

func TestUserNameFromRequest_NoUser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	name := UserNameFromRequest(req)
	if name != "anonymous" {
		t.Fatalf("expected 'anonymous', got %q", name)
	}
}
