package models

// APIResponse is the unified JSON response envelope for all API endpoints.
type APIResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data"`
	Error   string `json:"error,omitempty"`
}
