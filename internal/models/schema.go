package models

// ObjectName is a named database object with an optional comment.
type ObjectName struct {
	Name    string `json:"name"`
	Comment string `json:"comment,omitempty"`
}

// ColumnInfo contains detailed information about a table/view column.
type ColumnInfo struct {
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Nullable     bool    `json:"nullable"`
	Default      *string `json:"default,omitempty"`
	Comment      *string `json:"comment,omitempty"`
	IsPrimaryKey bool    `json:"is_primary_key"`
}

// TableInfo is schema information for a single table or view.
type TableInfo struct {
	Name       string       `json:"name"`
	SchemaName *string      `json:"schema_name,omitempty"`
	Columns    []ColumnInfo `json:"columns"`
	Comment    *string      `json:"comment,omitempty"`
}

// DatabaseSchema is the complete schema of a database.
type DatabaseSchema struct {
	DatabaseName string      `json:"database_name"`
	Tables       []TableInfo `json:"tables"`
	Views        []TableInfo `json:"views"`
}
