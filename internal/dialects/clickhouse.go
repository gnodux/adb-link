package dialects

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/gnodux/adb-link/internal/models"
)

// ClickHouseDialect implements SchemaDialect for ClickHouse.
type ClickHouseDialect struct{}

func (d *ClickHouseDialect) BuildDSN(config *models.DatasourceConfig, database string) string {
	conn := config.Connection
	port := conn.Port
	if port == 0 {
		port = 9000
	}
	db := database
	if db == "" {
		db = conn.DefaultDatabase
	}
	if db == "" {
		db = "default"
	}
	username := conn.Username
	if username == "" {
		username = "default"
	}
	// clickhouse DSN: clickhouse://user:password@host:port/database?params
	params := url.Values{}
	dialTimeout := "5s"
	if v, ok := config.Options["connect_timeout"]; ok {
		dialTimeout = fmt.Sprintf("%vs", v)
	}
	params.Set("dial_timeout", dialTimeout)
	if v, ok := config.Options["read_timeout"]; ok {
		params.Set("read_timeout", fmt.Sprintf("%vs", v))
	} else {
		params.Set("read_timeout", "60s")
	}
	return fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s?%s",
		url.PathEscape(username), url.PathEscape(conn.Password),
		conn.Host, port, db, params.Encode())
}

func (d *ClickHouseDialect) GetDatabases(ctx context.Context, db *sql.DB) ([]models.ObjectName, error) {
	query := `SELECT name, '' as comment FROM system.databases ORDER BY name`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObjectNames(rows)
}

func (d *ClickHouseDialect) GetTableNames(ctx context.Context, db *sql.DB, database string) ([]models.ObjectName, error) {
	query := `SELECT name, comment FROM system.tables WHERE database = ? AND engine NOT LIKE '%View%' ORDER BY name`
	rows, err := db.QueryContext(ctx, query, database)
	if err != nil {
		// Fallback without comment column
		query = `SELECT name, '' as comment FROM system.tables WHERE database = ? AND engine NOT LIKE '%View%' ORDER BY name`
		rows, err = db.QueryContext(ctx, query, database)
		if err != nil {
			return nil, err
		}
	}
	defer rows.Close()
	return scanObjectNames(rows)
}

func (d *ClickHouseDialect) GetViewNames(ctx context.Context, db *sql.DB, database string) ([]models.ObjectName, error) {
	query := `SELECT name, comment FROM system.tables WHERE database = ? AND engine LIKE '%View%' ORDER BY name`
	rows, err := db.QueryContext(ctx, query, database)
	if err != nil {
		query = `SELECT name, '' as comment FROM system.tables WHERE database = ? AND engine LIKE '%View%' ORDER BY name`
		rows, err = db.QueryContext(ctx, query, database)
		if err != nil {
			return nil, err
		}
	}
	defer rows.Close()
	return scanObjectNames(rows)
}

func (d *ClickHouseDialect) GetTableInfo(ctx context.Context, db *sql.DB, database, table string) (*models.TableInfo, error) {
	query := `SELECT name, type, comment, is_in_primary_key, default_expression 
		FROM system.columns 
		WHERE database = ? AND table = ? 
		ORDER BY position`
	rows, err := db.QueryContext(ctx, query, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []models.ColumnInfo
	for rows.Next() {
		var name, colType string
		var comment sql.NullString
		var isPK bool
		var defaultExpr sql.NullString
		if err := rows.Scan(&name, &colType, &comment, &isPK, &defaultExpr); err != nil {
			return nil, err
		}
		col := models.ColumnInfo{
			Name:         name,
			Type:         colType,
			Nullable:     len(colType) > 9 && colType[:9] == "Nullable(",
			IsPrimaryKey: isPK,
		}
		if defaultExpr.Valid && defaultExpr.String != "" {
			col.Default = &defaultExpr.String
		}
		if comment.Valid && comment.String != "" {
			col.Comment = &comment.String
		}
		columns = append(columns, col)
	}

	// Get table comment
	var tableComment *string
	commentQuery := `SELECT comment FROM system.tables WHERE database = ? AND name = ?`
	var tc sql.NullString
	if err := db.QueryRowContext(ctx, commentQuery, database, table).Scan(&tc); err == nil && tc.Valid && tc.String != "" {
		tableComment = &tc.String
	}

	return &models.TableInfo{
		Name:    table,
		Columns: columns,
		Comment: tableComment,
	}, nil
}

func (d *ClickHouseDialect) GetViewInfo(ctx context.Context, db *sql.DB, database, view string) (*models.TableInfo, error) {
	return d.GetTableInfo(ctx, db, database, view)
}
