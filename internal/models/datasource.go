package models

import "encoding/json"

// DatabaseType represents supported database types.
type DatabaseType string

const (
	DatabaseTypeMySQL         DatabaseType = "mysql"
	DatabaseTypePostgreSQL    DatabaseType = "postgresql"
	DatabaseTypeSQLite        DatabaseType = "sqlite"
	DatabaseTypeClickHouse    DatabaseType = "clickhouse"
	DatabaseTypeMSSQL         DatabaseType = "mssql"
	DatabaseTypeElasticsearch DatabaseType = "elasticsearch"
	DatabaseTypeHive          DatabaseType = "hive"
	DatabaseTypeGaussDB       DatabaseType = "gaussdb"
	DatabaseTypeRedis         DatabaseType = "redis"
	DatabaseTypeMongoDB       DatabaseType = "mongodb"
	DatabaseTypeMilvus        DatabaseType = "milvus"
	DatabaseTypeOracle        DatabaseType = "oracle"
	DatabaseTypeTiDB          DatabaseType = "tidb"
)

// DialectInfo provides dialect-specific metadata for LLM guidance.
type DialectInfo struct {
	SQLStyle        string   `json:"sql_style" yaml:"sql_style"`
	IdentifierQuote string   `json:"identifier_quote" yaml:"identifier_quote"`
	StringQuote     string   `json:"string_quote" yaml:"string_quote"`
	LimitSyntax     string   `json:"limit_syntax" yaml:"limit_syntax"`
	Features        []string `json:"features" yaml:"features"`
	Notes           string   `json:"notes" yaml:"notes"`
}

// DialectInfoMap contains dialect information for each database type.
var DialectInfoMap = map[DatabaseType]DialectInfo{
	DatabaseTypeMySQL: {
		SQLStyle:        "mysql",
		IdentifierQuote: "`",
		StringQuote:     "'",
		LimitSyntax:     "LIMIT n",
		Features:        []string{"views", "explain", "stored_procedures", "json_functions", "window_functions"},
		Notes:           "MySQL兼容语法，使用反引号包裹标识符",
	},
	DatabaseTypePostgreSQL: {
		SQLStyle:        "postgresql",
		IdentifierQuote: `"`,
		StringQuote:     "'",
		LimitSyntax:     "LIMIT n",
		Features:        []string{"views", "explain", "cte", "json_functions", "window_functions", "materialized_views", "arrays"},
		Notes:           "PostgreSQL语法，支持丰富的数据类型和高级查询特性",
	},
	DatabaseTypeSQLite: {
		SQLStyle:        "sqlite",
		IdentifierQuote: `"`,
		StringQuote:     "'",
		LimitSyntax:     "LIMIT n",
		Features:        []string{"views", "explain", "cte", "json_functions", "window_functions"},
		Notes:           "SQLite语法，轻量级数据库，部分高级特性不支持",
	},
	DatabaseTypeClickHouse: {
		SQLStyle:        "clickhouse",
		IdentifierQuote: "`",
		StringQuote:     "'",
		LimitSyntax:     "LIMIT n",
		Features:        []string{"views", "explain", "materialized_views", "array_functions", "window_functions"},
		Notes:           "ClickHouse语法，面向OLAP分析，JOIN能力有限，擅长聚合查询",
	},
	DatabaseTypeMSSQL: {
		SQLStyle:        "tsql",
		IdentifierQuote: "[",
		StringQuote:     "'",
		LimitSyntax:     "TOP n 或 OFFSET...FETCH",
		Features:        []string{"views", "explain", "stored_procedures", "cte", "window_functions", "json_functions"},
		Notes:           "T-SQL语法，使用方括号包裹标识符，分页用OFFSET...FETCH NEXT",
	},
	DatabaseTypeElasticsearch: {
		SQLStyle:        "elasticsearch_dsl",
		IdentifierQuote: "",
		StringQuote:     "",
		LimitSyntax:     "size字段",
		Features:        []string{"full_text_search", "aggregations", "nested_queries"},
		Notes:           "不使用SQL，需传入JSON格式的Elasticsearch Query DSL查询体",
	},
	DatabaseTypeHive: {
		SQLStyle:        "hive_hql",
		IdentifierQuote: "`",
		StringQuote:     "",
		LimitSyntax:     "LIMIT n",
		Features:        []string{"partitions", "bucketing", "UDF", "lateral_view"},
		Notes:           "使用HiveQL语法，支持分区表、桶表、用户自定义函数。不支持UPDATE/DELETE（除ACID表外）。",
	},
	DatabaseTypeGaussDB: {
		SQLStyle:        "gaussdb",
		IdentifierQuote: `"`,
		StringQuote:     "'",
		LimitSyntax:     "LIMIT n",
		Features:        []string{"views", "explain", "cte", "json_functions", "window_functions", "materialized_views"},
		Notes:           "华为GaussDB，兼容PostgreSQL语法。系统目录可能与标准PostgreSQL略有差异。",
	},
	DatabaseTypeRedis: {
		SQLStyle:        "redis_commands",
		IdentifierQuote: "",
		StringQuote:     "",
		LimitSyntax:     "SCAN的COUNT参数",
		Features:        []string{"key_value", "hash", "list", "set", "sorted_set", "stream", "pubsub"},
		Notes:           "Redis使用原生命令，不使用SQL。查询时传入Redis命令字符串（如 'GET mykey', 'HGETALL myhash'）。Schema发现列出key模式分组。",
	},
	DatabaseTypeMongoDB: {
		SQLStyle:        "mongodb_query",
		IdentifierQuote: "",
		StringQuote:     "",
		LimitSyntax:     "查询选项中的limit字段",
		Features:        []string{"document_query", "aggregation_pipeline", "index_management", "change_streams"},
		Notes:           "MongoDB使用JSON过滤语法。查询格式: {\"collection\": \"mycoll\", \"filter\": {...}, \"projection\": {...}, \"sort\": {...}, \"limit\": N}。聚合: {\"collection\": \"mycoll\", \"pipeline\": [...]}。",
	},
	DatabaseTypeMilvus: {
		SQLStyle:        "milvus_query",
		IdentifierQuote: "",
		StringQuote:     "",
		LimitSyntax:     "limit参数",
		Features:        []string{"vector_search", "scalar_filter", "hybrid_search", "collection_schema"},
		Notes:           "Milvus向量数据库。标量查询: {\"collection\": \"mycoll\", \"filter\": \"id > 100\", \"output_fields\": [\"*\"]}。向量搜索: {\"collection\": \"mycoll\", \"data\": [[0.1, 0.2, ...]], \"anns_field\": \"embedding\", \"limit\": 10}。",
	},
	DatabaseTypeOracle: {
		SQLStyle:        "oracle",
		IdentifierQuote: `"`,
		StringQuote:     "'",
		LimitSyntax:     "FETCH FIRST n ROWS ONLY (12c+) 或 ROWNUM",
		Features:        []string{"views", "explain", "stored_procedures", "cte", "window_functions", "materialized_views", "json_functions"},
		Notes:           "Oracle SQL语法。使用双引号标识符。LIMIT通过FETCH FIRST (12c+)或ROWNUM子查询实现。",
	},
	DatabaseTypeTiDB: {
		SQLStyle:        "mysql",
		IdentifierQuote: "`",
		StringQuote:     "'",
		LimitSyntax:     "LIMIT n",
		Features:        []string{"views", "explain", "stored_procedures", "json_functions", "window_functions", "cte", "distributed_transactions"},
		Notes:           "TiDB兼容MySQL语法。使用反引号标识符。分布式SQL数据库，支持强一致性事务。默认端口4000。",
	},
}

// ConnectionConfig holds database connection parameters.
type ConnectionConfig struct {
	Host            string `json:"host" yaml:"host"`
	Port            int    `json:"port,omitempty" yaml:"port,omitempty"`
	Username        string `json:"username,omitempty" yaml:"username,omitempty"`
	Password        string `json:"password,omitempty" yaml:"password,omitempty"`
	DefaultDatabase string `json:"default_database,omitempty" yaml:"default_database,omitempty"`
	Path            string `json:"path,omitempty" yaml:"path,omitempty"` // SQLite only
}

// DatasourceConfig represents a datasource definition from YAML.
type DatasourceConfig struct {
	Kind        string           `json:"kind" yaml:"kind"`
	Name        string           `json:"name" yaml:"name"`
	Type        DatabaseType     `json:"type" yaml:"type"`
	Description string           `json:"description,omitempty" yaml:"description,omitempty"`
	Shadow      bool             `json:"shadow,omitempty" yaml:"shadow,omitempty"`
	Enable      *bool            `json:"enable,omitempty" yaml:"enable,omitempty"`
	Connection  ConnectionConfig `json:"connection" yaml:"connection"`
	Options     map[string]any   `json:"options,omitempty" yaml:"options,omitempty"`
}

// IsEnabled returns whether the datasource is enabled (defaults to true).
func (d *DatasourceConfig) IsEnabled() bool {
	if d.Enable == nil {
		return true
	}
	return *d.Enable
}

// DatasourceInfo is the public-facing datasource summary.
type DatasourceInfo struct {
	Name        string       `json:"name"`
	Type        DatabaseType `json:"type"`
	Description string       `json:"description"`
	Shadow      bool         `json:"shadow"`
	Dialect     DialectInfo  `json:"dialect"`
	ServerInfo  *ServerInfo  `json:"server_info,omitempty"`
}

// ColumnMeta is metadata override for a single column.
type ColumnMeta struct {
	Comment string `json:"comment,omitempty" yaml:"comment,omitempty"`
}

// TableMeta is metadata override for a table or view.
type TableMeta struct {
	Comment string                `json:"comment,omitempty" yaml:"comment,omitempty"`
	Columns map[string]ColumnMeta `json:"columns,omitempty" yaml:"columns,omitempty"`
}

// DatabaseMeta is metadata override for a database.
type DatabaseMeta struct {
	Comment string `json:"comment,omitempty" yaml:"comment,omitempty"`
}

// MetadataConfig is YAML metadata config that enriches schema objects.
type MetadataConfig struct {
	Kind       string                  `json:"kind" yaml:"kind"`
	Enable     *bool                   `json:"enable,omitempty" yaml:"enable,omitempty"`
	Datasource string                  `json:"datasource" yaml:"datasource"`
	Databases  map[string]DatabaseMeta `json:"databases,omitempty" yaml:"databases,omitempty"`
	Tables     map[string]TableMeta    `json:"tables,omitempty" yaml:"tables,omitempty"`
	Views      map[string]TableMeta    `json:"views,omitempty" yaml:"views,omitempty"`
}

// IsEnabled returns whether the metadata config is enabled.
func (m *MetadataConfig) IsEnabled() bool {
	if m.Enable == nil {
		return true
	}
	return *m.Enable
}

// ToolParameter defines a parameter for a configured tool.
type ToolParameter struct {
	Name        string `json:"name" yaml:"name"`
	Type        string `json:"type,omitempty" yaml:"type,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
	Default     any    `json:"default,omitempty" yaml:"default,omitempty"`
}

// ToolConfig is a configured query tool with a SQL/DSL template.
type ToolConfig struct {
	Kind        string          `json:"kind" yaml:"kind"`
	Name        string          `json:"name" yaml:"name"`
	Description string          `json:"description" yaml:"description"`
	Enable      *bool           `json:"enable,omitempty" yaml:"enable,omitempty"`
	Datasource  string          `json:"datasource" yaml:"datasource"`
	Database    string          `json:"database,omitempty" yaml:"database,omitempty"`
	Template    string          `json:"template" yaml:"template"`
	Parameters  []ToolParameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	// InputSchema holds a raw JSON Schema object for the tool's input.
	// When non-nil, it takes precedence over the legacy Parameters array.
	InputSchema json.RawMessage `json:"input_schema,omitempty" yaml:"input_schema,omitempty"`
}

// BuildInputSchema returns the JSON Schema for this tool's input.
// If InputSchema is set, it is returned directly. Otherwise, a schema
// is derived from the legacy Parameters list.
func (t *ToolConfig) BuildInputSchema() map[string]any {
	if len(t.InputSchema) > 0 {
		var schema map[string]any
		if err := json.Unmarshal(t.InputSchema, &schema); err == nil {
			return schema
		}
	}
	// Derive from legacy Parameters
	props := map[string]any{}
	required := []string{}
	for _, p := range t.Parameters {
		typ := p.Type
		if typ == "" {
			typ = "string"
		}
		switch typ {
		case "number", "integer", "boolean":
			// keep as-is
		default:
			typ = "string"
		}
		entry := map[string]any{"type": typ}
		if p.Description != "" {
			entry["description"] = p.Description
		}
		if p.Required {
			required = append(required, p.Name)
		} else if p.Default != nil {
			entry["default"] = p.Default
		}
		props[p.Name] = entry
	}
	schema := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// IsEnabled returns whether the tool is enabled.
func (t *ToolConfig) IsEnabled() bool {
	if t.Enable == nil {
		return true
	}
	return *t.Enable
}

// ToolsetConfig is a named group of tools.
type ToolsetConfig struct {
	Kind        string   `json:"kind" yaml:"kind"`
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Enable      *bool    `json:"enable,omitempty" yaml:"enable,omitempty"`
	Tools       []string `json:"tools" yaml:"tools"`
}

// IsEnabled returns whether the toolset is enabled.
func (t *ToolsetConfig) IsEnabled() bool {
	if t.Enable == nil {
		return true
	}
	return *t.Enable
}

// AuthUser is an API user authorized via Bearer token.
type AuthUser struct {
	Name        string `json:"name" yaml:"name"`
	APIKey      string `json:"api_key" yaml:"api_key"`
	Email       string `json:"email,omitempty" yaml:"email,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Group       string `json:"group,omitempty" yaml:"group,omitempty"`
	Enable      *bool  `json:"enable,omitempty" yaml:"enable,omitempty"`
}

// IsEnabled returns whether the auth user is enabled.
func (a *AuthUser) IsEnabled() bool {
	if a.Enable == nil {
		return true
	}
	return *a.Enable
}

// AuthAPIConfig holds a list of authorized API users.
type AuthAPIConfig struct {
	Kind  string     `json:"kind" yaml:"kind"`
	Users []AuthUser `json:"users" yaml:"users"`
}

// PermissionRule defines access to a datasource with glob pattern support.
type PermissionRule struct {
	Datasource string   `json:"datasource" yaml:"datasource"`
	Databases  []string `json:"databases,omitempty" yaml:"databases,omitempty"`
	Tables     []string `json:"tables,omitempty" yaml:"tables,omitempty"`
	Fields     []string `json:"fields,omitempty" yaml:"fields,omitempty"`
}

// PermissionConfig defines permissions for users/groups.
type PermissionConfig struct {
	Kind   string           `json:"kind" yaml:"kind"`
	Users  []string         `json:"users,omitempty" yaml:"users,omitempty"`
	Groups []string         `json:"groups,omitempty" yaml:"groups,omitempty"`
	Enable *bool            `json:"enable,omitempty" yaml:"enable,omitempty"`
	Rules  []PermissionRule `json:"rules,omitempty" yaml:"rules,omitempty"`
	Tools  []string         `json:"tools,omitempty" yaml:"tools,omitempty"`
}

// IsEnabled returns whether the permission config is enabled.
func (p *PermissionConfig) IsEnabled() bool {
	if p.Enable == nil {
		return true
	}
	return *p.Enable
}
