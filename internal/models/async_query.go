package models

import "time"

// QueryStatus represents the lifecycle status of an async query.
type QueryStatus string

const (
	QueryStatusPending   QueryStatus = "pending"
	QueryStatusRunning   QueryStatus = "running"
	QueryStatusSucceeded QueryStatus = "succeeded"
	QueryStatusFailed    QueryStatus = "failed"
	QueryStatusCancelled QueryStatus = "cancelled"
)

// AsyncQueryRequest is the request for submitting an async SQL query.
type AsyncQueryRequest struct {
	DatasourceName string `json:"datasource_name"`
	Database       string `json:"database"`
	SQL            string `json:"sql"`
	Limit          int    `json:"limit"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// AsyncToolRequest is the request for submitting an async tool execution.
type AsyncToolRequest struct {
	Parameters     map[string]any `json:"parameters"`
	TimeoutSeconds int            `json:"timeout_seconds"`
}

// QueryIDRequest contains a query ID for status/result/cancel operations.
type QueryIDRequest struct {
	QueryID string `json:"query_id"`
}

// AsyncQueryStatusResponse is the status snapshot of an async query.
type AsyncQueryStatusResponse struct {
	QueryID         string      `json:"query_id"`
	Status          QueryStatus `json:"status"`
	CreatedAt       time.Time   `json:"created_at"`
	StartedAt       *time.Time  `json:"started_at,omitempty"`
	CompletedAt     *time.Time  `json:"completed_at,omitempty"`
	ExecutionTimeMs *float64    `json:"execution_time_ms,omitempty"`
	ErrorMessage    *string     `json:"error_message,omitempty"`
	RowCount        *int        `json:"row_count,omitempty"`
	Truncated       *bool       `json:"truncated,omitempty"`
}

// AsyncQueryResult is the full result of a completed async query.
type AsyncQueryResult struct {
	QueryID         string            `json:"query_id"`
	Status          QueryStatus       `json:"status"`
	Columns         []QueryColumnMeta `json:"columns,omitempty"`
	Rows            [][]any           `json:"rows,omitempty"`
	RowCount        *int              `json:"row_count,omitempty"`
	ExecutionTimeMs *float64          `json:"execution_time_ms,omitempty"`
	ErrorMessage    *string           `json:"error_message,omitempty"`
}
