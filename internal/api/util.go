package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gnodux/adb-link/internal/apperr"
	"github.com/gnodux/adb-link/internal/models"
)

// UserFromRequest returns the AuthUser stored in the request context, or nil.
func UserFromRequest(r *http.Request) *models.AuthUser {
	return models.AuthUserFromContext(r.Context())
}

// UserNameFromRequest returns the authenticated user's name, or "anonymous".
func UserNameFromRequest(r *http.Request) string {
	if name := models.AuthUserNameFromContext(r.Context()); name != "" {
		return name
	}
	return "anonymous"
}

// WithUser returns a new context that contains the given user.
func WithUser(ctx context.Context, u *models.AuthUser) context.Context {
	return models.WithAuthUser(ctx, u)
}

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// WriteOK writes the standard success envelope: {success:true, data}.
func WriteOK(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusOK, models.APIResponse{Success: true, Data: data})
}

// WriteError maps any error to an HTTP response. Typed apperr.Error values
// produce their declared status; unrecognized errors default to
// 400 Bad Request, matching the legacy "input/validation failure" semantics
// that the previous WriteError signalled (it now returns a real 4xx instead
// of 200/success:false).
func WriteError(w http.ResponseWriter, msg string) {
	WriteJSON(w, http.StatusBadRequest, models.APIResponse{Success: false, Error: msg})
}

// WriteErrorStatus writes an HTTP error with the given status code and message.
func WriteErrorStatus(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, models.APIResponse{Success: false, Error: msg})
}

// WriteAppError handles a typed apperr.Error or falls back to 500 for
// arbitrary errors.
func WriteAppError(w http.ResponseWriter, err error) {
	if err == nil {
		WriteErrorStatus(w, http.StatusInternalServerError, "unknown error")
		return
	}
	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		payload := map[string]any{
			"success": false,
			"error":   appErr.Msg,
			"code":    string(appErr.Code),
		}
		WriteJSON(w, appErr.Status, payload)
		return
	}
	WriteErrorStatus(w, http.StatusInternalServerError, err.Error())
}

// DecodeJSON decodes a JSON body into the given pointer.
func DecodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
