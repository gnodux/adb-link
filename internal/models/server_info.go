package models

// ServerInfo represents runtime metadata from a database server.
type ServerInfo struct {
	Version    string   `json:"version"`
	SQLMode    string   `json:"sql_mode,omitempty"`
	Timezone   string   `json:"timezone,omitempty"`
	Extensions []string `json:"extensions,omitempty"`
}
