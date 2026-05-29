package models

// QueryRequest is the request payload for executing a SQL query.
type QueryRequest struct {
	DatasourceName string `json:"datasource_name"`
	Database       string `json:"database"`
	SQL            string `json:"sql"`
	Limit          int    `json:"limit"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// QueryColumnMeta describes name and data type of a column.
type QueryColumnMeta struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// QueryResult is the structured result of a SQL query execution.
type QueryResult struct {
	Columns         []QueryColumnMeta `json:"columns"`
	Rows            [][]any           `json:"rows"`
	RowCount        int               `json:"row_count"`
	ExecutionTimeMs float64           `json:"execution_time_ms"`
	Truncated       bool              `json:"truncated"`
	Limit           int               `json:"limit"`
}

// ExplainRequest is the request payload for an execution plan.
type ExplainRequest struct {
	DatasourceName string `json:"datasource_name"`
	Database       string `json:"database"`
	SQL            string `json:"sql"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// ExplainResult is the result of an EXPLAIN query.
type ExplainResult struct {
	DatabaseType    string            `json:"database_type"`
	Columns         []QueryColumnMeta `json:"columns"`
	Rows            [][]any           `json:"rows"`
	ExecutionTimeMs float64           `json:"execution_time_ms"`
}
