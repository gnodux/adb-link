package services

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gnodux/adb-link/internal/config"
	"github.com/gnodux/adb-link/internal/models"
)

// UnsupportedOperationError is returned when an operation is not supported for the target datasource.
type UnsupportedOperationError struct {
	Msg string
}

func (e *UnsupportedOperationError) Error() string { return e.Msg }

var explainTemplates = map[models.DatabaseType]string{
	models.DatabaseTypeMySQL:      "EXPLAIN FORMAT=JSON %s",
	models.DatabaseTypePostgreSQL: "EXPLAIN (FORMAT JSON) %s",
	models.DatabaseTypeSQLite:     "EXPLAIN QUERY PLAN %s",
	models.DatabaseTypeClickHouse: "EXPLAIN %s",
	models.DatabaseTypeGaussDB:    "EXPLAIN (FORMAT JSON) %s",
	models.DatabaseTypeTiDB:       "EXPLAIN FORMAT='brief' %s",
}

func isExplainSupported(t models.DatabaseType) bool {
	if _, ok := explainTemplates[t]; ok {
		return true
	}
	return t == models.DatabaseTypeMSSQL
}

// QueryService executes SQL/DSL queries with auditing and retry support.
type QueryService struct {
	connectionService *ConnectionService
	configService     *config.ConfigService
	permissionService *PermissionService
}

// NewQueryService creates a new QueryService.
func NewQueryService(
	conn *ConnectionService,
	cfg *config.ConfigService,
	perm *PermissionService,
) *QueryService {
	return &QueryService{
		connectionService: conn,
		configService:     cfg,
		permissionService: perm,
	}
}

// Execute runs a query against the configured datasource.
func (qs *QueryService) Execute(ctx context.Context, req *models.QueryRequest, userName string) (*models.QueryResult, error) {
	if userName == "" {
		userName = "anonymous"
	}
	AuditLog().Printf("user=%s | action=execute_query | datasource=%s | database=%s | sql=%s",
		userName, req.DatasourceName, req.Database, truncate(req.SQL, 200))

	cfg, err := qs.configService.GetDatasource(req.DatasourceName)
	if err != nil {
		ErrorLog().Printf("user=%s | action=execute_query | datasource=%s | error=%s",
			userName, req.DatasourceName, err.Error())
		return nil, err
	}

	if cfg.Shadow {
		err := fmt.Errorf("数据源 '%s' 是 shadow 数据源，不允许直接查询。请通过已配置的工具(tool)来访问该数据源", req.DatasourceName)
		ErrorLog().Printf("user=%s | action=execute_query | datasource=%s | error=%s",
			userName, req.DatasourceName, err.Error())
		return nil, err
	}

	// Permission check
	if qs.permissionService != nil {
		if req.Database != "" {
			if !qs.permissionService.CheckDatabase(userName, req.DatasourceName, req.Database) {
				err := fmt.Errorf("access denied: user '%s' cannot access '%s/%s'", userName, req.DatasourceName, req.Database)
				ErrorLog().Printf("user=%s | action=execute_query | error=%s", userName, err.Error())
				return nil, err
			}
		} else {
			if !qs.permissionService.CheckDatasource(userName, req.DatasourceName) {
				err := fmt.Errorf("access denied: user '%s' cannot access '%s'", userName, req.DatasourceName)
				ErrorLog().Printf("user=%s | action=execute_query | error=%s", userName, err.Error())
				return nil, err
			}
		}
	}

	if req.Limit <= 0 {
		req.Limit = 1000
	}
	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 60
	}

	if IsNonSQLType(cfg.Type) {
		return qs.executeNonSQL(ctx, req)
	}
	return qs.executeSQL(ctx, req, cfg)
}

// executeSQL runs a standard SQL query through database/sql.
func (qs *QueryService) executeSQL(ctx context.Context, req *models.QueryRequest, cfg *models.DatasourceConfig) (*models.QueryResult, error) {
	db, _, err := qs.connectionService.GetSQLDB(req.DatasourceName, req.Database)
	if err != nil {
		return nil, err
	}

	sqlStr := strings.TrimRight(strings.TrimSpace(req.SQL), ";")
	fetchLimit := req.Limit + 1

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
	defer cancel()

	start := time.Now()
	rows, err := qs.queryWithRetry(timeoutCtx, db, sqlStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	colTypes, _ := rows.ColumnTypes()
	columns := make([]models.QueryColumnMeta, len(cols))
	for i, name := range cols {
		colType := "TEXT"
		if colTypes != nil && i < len(colTypes) {
			if t := colTypes[i].DatabaseTypeName(); t != "" {
				colType = t
			}
		}
		columns[i] = models.QueryColumnMeta{Name: name, Type: colType}
	}

	var rawRows [][]any
	for rows.Next() {
		if len(rawRows) >= fetchLimit {
			break
		}
		holders := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range holders {
			ptrs[i] = &holders[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		rawRows = append(rawRows, serializeRow(holders))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	truncated := len(rawRows) > req.Limit
	if truncated {
		rawRows = rawRows[:req.Limit]
	}

	elapsedMs := float64(time.Since(start).Microseconds()) / 1000.0
	return &models.QueryResult{
		Columns:         columns,
		Rows:            rawRows,
		RowCount:        len(rawRows),
		ExecutionTimeMs: roundFloat(elapsedMs, 2),
		Truncated:       truncated,
		Limit:           req.Limit,
	}, nil
}

// queryWithRetry retries once on transient connection loss.
func (qs *QueryService) queryWithRetry(ctx context.Context, db *sql.DB, sqlStr string) (*sql.Rows, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		rows, err := db.QueryContext(ctx, sqlStr)
		if err != nil {
			lastErr = err
			msg := err.Error()
			if attempt == 0 && (strings.Contains(msg, "Lost connection") || strings.Contains(msg, "MySQL server has gone away") || strings.Contains(msg, "broken pipe")) {
				continue
			}
			return nil, err
		}
		return rows, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("query failed after retries")
}

// executeNonSQL runs a query through a NonSQLClient (Elasticsearch, Redis, MongoDB, Milvus).
func (qs *QueryService) executeNonSQL(ctx context.Context, req *models.QueryRequest) (*models.QueryResult, error) {
	client, _, err := qs.connectionService.GetNonSQLClient(req.DatasourceName)
	if err != nil {
		return nil, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
	defer cancel()

	return client.Execute(timeoutCtx, req.Database, req.SQL, req.Limit)
}

// Explain runs an EXPLAIN-style query for the supported database types.
func (qs *QueryService) Explain(ctx context.Context, req *models.ExplainRequest, userName string) (*models.ExplainResult, error) {
	if userName == "" {
		userName = "anonymous"
	}
	AuditLog().Printf("user=%s | action=explain_query | datasource=%s | database=%s | sql=%s",
		userName, req.DatasourceName, req.Database, truncate(req.SQL, 200))

	cfg, err := qs.configService.GetDatasource(req.DatasourceName)
	if err != nil {
		return nil, err
	}
	if cfg.Shadow {
		return nil, fmt.Errorf("数据源 '%s' 是 shadow 数据源，不允许直接查询。请通过已配置的工具(tool)来访问该数据源", req.DatasourceName)
	}
	if qs.permissionService != nil {
		if req.Database != "" {
			if !qs.permissionService.CheckDatabase(userName, req.DatasourceName, req.Database) {
				return nil, fmt.Errorf("access denied: user '%s' cannot access '%s/%s'", userName, req.DatasourceName, req.Database)
			}
		} else {
			if !qs.permissionService.CheckDatasource(userName, req.DatasourceName) {
				return nil, fmt.Errorf("access denied: user '%s' cannot access '%s'", userName, req.DatasourceName)
			}
		}
	}
	if !isExplainSupported(cfg.Type) {
		return nil, &UnsupportedOperationError{Msg: fmt.Sprintf("explain_query is not supported for datasource type: %s", cfg.Type)}
	}

	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 60
	}

	if cfg.Type == models.DatabaseTypeMSSQL {
		return qs.explainMSSQL(ctx, req, cfg)
	}
	return qs.explainTemplate(ctx, req, cfg)
}

func (qs *QueryService) explainTemplate(ctx context.Context, req *models.ExplainRequest, cfg *models.DatasourceConfig) (*models.ExplainResult, error) {
	template := explainTemplates[cfg.Type]
	sqlStr := strings.TrimRight(strings.TrimSpace(req.SQL), ";")
	explainSQL := fmt.Sprintf(template, sqlStr)

	db, _, err := qs.connectionService.GetSQLDB(req.DatasourceName, req.Database)
	if err != nil {
		return nil, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
	defer cancel()

	start := time.Now()
	rows, err := db.QueryContext(timeoutCtx, explainSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, raw, err := scanAllRows(rows)
	if err != nil {
		return nil, err
	}
	elapsedMs := float64(time.Since(start).Microseconds()) / 1000.0
	return &models.ExplainResult{
		DatabaseType:    string(cfg.Type),
		Columns:         columns,
		Rows:            raw,
		ExecutionTimeMs: roundFloat(elapsedMs, 2),
	}, nil
}

func (qs *QueryService) explainMSSQL(ctx context.Context, req *models.ExplainRequest, cfg *models.DatasourceConfig) (*models.ExplainResult, error) {
	db, _, err := qs.connectionService.GetSQLDB(req.DatasourceName, req.Database)
	if err != nil {
		return nil, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
	defer cancel()

	conn, err := db.Conn(timeoutCtx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(timeoutCtx, "SET SHOWPLAN_XML ON"); err != nil {
		return nil, err
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), "SET SHOWPLAN_XML OFF")
	}()

	sqlStr := strings.TrimRight(strings.TrimSpace(req.SQL), ";")
	start := time.Now()
	rows, err := conn.QueryContext(timeoutCtx, sqlStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, raw, err := scanAllRows(rows)
	if err != nil {
		return nil, err
	}
	elapsedMs := float64(time.Since(start).Microseconds()) / 1000.0
	return &models.ExplainResult{
		DatabaseType:    string(cfg.Type),
		Columns:         columns,
		Rows:            raw,
		ExecutionTimeMs: roundFloat(elapsedMs, 2),
	}, nil
}

// paramRegex matches :param_name placeholders.
var paramRegex = regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)

// ExecuteTemplate runs a tool's template with the given parameters.
func (qs *QueryService) ExecuteTemplate(ctx context.Context, tool *models.ToolConfig, params map[string]any, userName string) (*models.QueryResult, error) {
	if userName == "" {
		userName = "anonymous"
	}
	paramsJSON, _ := json.Marshal(params)
	AuditLog().Printf("user=%s | action=execute_tool | tool=%s | datasource=%s | params=%s",
		userName, tool.Name, tool.Datasource, string(paramsJSON))

	if qs.permissionService != nil {
		if !qs.permissionService.CheckTool(userName, tool.Name) {
			err := fmt.Errorf("access denied: user '%s' cannot execute tool '%s'", userName, tool.Name)
			ErrorLog().Printf("user=%s | action=execute_tool | tool=%s | error=%s", userName, tool.Name, err.Error())
			return nil, err
		}
	}

	cfg, err := qs.configService.GetDatasource(tool.Datasource)
	if err != nil {
		return nil, err
	}

	if IsNonSQLType(cfg.Type) {
		body := tool.Template
		for k, v := range params {
			var replacement string
			switch val := v.(type) {
			case string:
				replacement = fmt.Sprintf("%q", val)
			default:
				b, _ := json.Marshal(val)
				replacement = string(b)
			}
			body = strings.ReplaceAll(body, ":"+k, replacement)
		}
		db := tool.Database
		if db == "" {
			db = "_all"
		}
		req := &models.QueryRequest{
			DatasourceName: tool.Datasource,
			Database:       db,
			SQL:            body,
			Limit:          1000,
			TimeoutSeconds: 30,
		}
		return qs.executeNonSQL(ctx, req)
	}

	if cfg.Type == models.DatabaseTypeHive {
		body := tool.Template
		for k, v := range params {
			body = strings.ReplaceAll(body, ":"+k, fmt.Sprintf("%v", v))
		}
		db := tool.Database
		if db == "" {
			db = "default"
		}
		req := &models.QueryRequest{
			DatasourceName: tool.Datasource,
			Database:       db,
			SQL:            body,
			Limit:          1000,
			TimeoutSeconds: 30,
		}
		return qs.executeSQL(ctx, req, cfg)
	}

	return qs.executeTemplateSQL(ctx, tool, params, cfg)
}

// executeTemplateSQL substitutes :param placeholders for SQL datasources by
// converting them to positional placeholders the driver supports.
func (qs *QueryService) executeTemplateSQL(ctx context.Context, tool *models.ToolConfig, params map[string]any, cfg *models.DatasourceConfig) (*models.QueryResult, error) {
	db, _, err := qs.connectionService.GetSQLDB(tool.Datasource, tool.Database)
	if err != nil {
		return nil, err
	}

	// Convert :param placeholders into driver-specific placeholders and an args slice.
	finalSQL, args := convertNamedParams(tool.Template, params, cfg.Type)

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	start := time.Now()
	rows, err := db.QueryContext(timeoutCtx, finalSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, raw, err := scanAllRows(rows)
	if err != nil {
		return nil, err
	}
	elapsedMs := float64(time.Since(start).Microseconds()) / 1000.0
	return &models.QueryResult{
		Columns:         columns,
		Rows:            raw,
		RowCount:        len(raw),
		ExecutionTimeMs: roundFloat(elapsedMs, 2),
		Truncated:       false,
		Limit:           len(raw),
	}, nil
}

// convertNamedParams converts :name placeholders into driver-specific placeholders.
func convertNamedParams(template string, params map[string]any, t models.DatabaseType) (string, []any) {
	var args []any
	idx := 0
	out := paramRegex.ReplaceAllStringFunc(template, func(match string) string {
		name := match[1:]
		v, ok := params[name]
		if !ok {
			v = nil
		}
		args = append(args, v)
		idx++
		switch t {
		case models.DatabaseTypePostgreSQL, models.DatabaseTypeGaussDB:
			return fmt.Sprintf("$%d", idx)
		case models.DatabaseTypeMSSQL:
			return fmt.Sprintf("@p%d", idx)
		default:
			return "?"
		}
	})
	return out, args
}

// scanAllRows pulls all rows + column metadata from a *sql.Rows.
func scanAllRows(rows *sql.Rows) ([]models.QueryColumnMeta, [][]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}
	colTypes, _ := rows.ColumnTypes()
	columns := make([]models.QueryColumnMeta, len(cols))
	for i, name := range cols {
		colType := "TEXT"
		if colTypes != nil && i < len(colTypes) {
			if t := colTypes[i].DatabaseTypeName(); t != "" {
				colType = t
			}
		}
		columns[i] = models.QueryColumnMeta{Name: name, Type: colType}
	}
	var raw [][]any
	for rows.Next() {
		holders := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range holders {
			ptrs[i] = &holders[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, nil, err
		}
		raw = append(raw, serializeRow(holders))
	}
	return columns, raw, rows.Err()
}

// serializeRow converts driver values into JSON-safe values.
func serializeRow(row []any) []any {
	out := make([]any, len(row))
	for i, v := range row {
		out[i] = serializeValue(v)
	}
	return out
}

func serializeValue(v any) any {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case []byte:
		// best-effort: try to keep printable strings as-is, otherwise hex.
		if isPrintable(x) {
			return string(x)
		}
		return hex.EncodeToString(x)
	case time.Time:
		return x.Format(time.RFC3339Nano)
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, string:
		return x
	case map[string]any, []any:
		return x
	default:
		return fmt.Sprintf("%v", x)
	}
}

func isPrintable(b []byte) bool {
	if len(b) == 0 {
		return true
	}
	for _, c := range b {
		if c < 0x20 && c != '\n' && c != '\r' && c != '\t' {
			return false
		}
	}
	return true
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func roundFloat(f float64, decimals int) float64 {
	mult := 1.0
	for i := 0; i < decimals; i++ {
		mult *= 10
	}
	return float64(int64(f*mult+0.5)) / mult
}
