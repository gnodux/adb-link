package dialects

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/gnodux/adb-link/internal/models"
)

// PostgreSQLDialect implements SchemaDialect for PostgreSQL.
type PostgreSQLDialect struct{}

func (d *PostgreSQLDialect) BuildDSN(config *models.DatasourceConfig, database string) string {
	conn := config.Connection
	port := conn.Port
	if port == 0 {
		port = 5432
	}
	db := database
	if db == "" {
		db = conn.DefaultDatabase
	}
	if db == "" {
		db = "postgres"
	}
	params := url.Values{}
	params.Set("sslmode", "disable")
	if v, ok := config.Options["sslmode"].(string); ok {
		params.Set("sslmode", v)
	}
	// Defaults: prevent indefinite hangs on unreachable hosts.
	// connect_timeout in seconds, statement_timeout in milliseconds.
	connectTimeout := "5"
	if v, ok := config.Options["connect_timeout"]; ok {
		connectTimeout = fmt.Sprintf("%v", v)
	}
	params.Set("connect_timeout", connectTimeout)
	if v, ok := config.Options["statement_timeout"]; ok {
		params.Set("statement_timeout", fmt.Sprintf("%v", v))
	} else {
		params.Set("statement_timeout", "60000")
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?%s",
		url.PathEscape(conn.Username), url.PathEscape(conn.Password),
		conn.Host, port, db, params.Encode())
}

func (d *PostgreSQLDialect) GetDatabases(ctx context.Context, db *sql.DB) ([]models.ObjectName, error) {
	query := `SELECT datname as name, COALESCE(shobj_description(oid, 'pg_database'), '') as comment 
		FROM pg_database 
		WHERE datistemplate = false AND datname NOT IN ('postgres') 
		ORDER BY datname`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObjectNames(rows)
}

func (d *PostgreSQLDialect) GetTableNames(ctx context.Context, db *sql.DB, database string) ([]models.ObjectName, error) {
	query := `SELECT c.relname as name, COALESCE(obj_description(c.oid), '') as comment 
		FROM pg_class c 
		JOIN pg_namespace n ON n.oid = c.relnamespace 
		WHERE c.relkind = 'r' AND n.nspname = 'public' 
		ORDER BY c.relname`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObjectNames(rows)
}

func (d *PostgreSQLDialect) GetViewNames(ctx context.Context, db *sql.DB, database string) ([]models.ObjectName, error) {
	query := `SELECT c.relname as name, COALESCE(obj_description(c.oid), '') as comment 
		FROM pg_class c 
		JOIN pg_namespace n ON n.oid = c.relnamespace 
		WHERE c.relkind = 'v' AND n.nspname = 'public' 
		ORDER BY c.relname`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObjectNames(rows)
}

func (d *PostgreSQLDialect) GetTableInfo(ctx context.Context, db *sql.DB, database, table string) (*models.TableInfo, error) {
	query := `SELECT 
		a.attname AS name,
		pg_catalog.format_type(a.atttypid, a.atttypmod) AS type,
		NOT a.attnotnull AS nullable,
		pg_get_expr(ad.adbin, ad.adrelid) AS default_val,
		COALESCE(col_description(a.attrelid, a.attnum), '') AS comment,
		COALESCE((SELECT TRUE FROM pg_index i WHERE i.indrelid = a.attrelid AND a.attnum = ANY(i.indkey) AND i.indisprimary), FALSE) AS is_pk
	FROM pg_attribute a
	LEFT JOIN pg_attrdef ad ON ad.adrelid = a.attrelid AND ad.adnum = a.attnum
	WHERE a.attrelid = $1::regclass AND a.attnum > 0 AND NOT a.attisdropped
	ORDER BY a.attnum`

	tableName := fmt.Sprintf("public.%s", table)
	rows, err := db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []models.ColumnInfo
	for rows.Next() {
		var name, colType string
		var nullable, isPK bool
		var defaultVal, comment sql.NullString
		if err := rows.Scan(&name, &colType, &nullable, &defaultVal, &comment, &isPK); err != nil {
			return nil, err
		}
		col := models.ColumnInfo{
			Name:         name,
			Type:         colType,
			Nullable:     nullable,
			IsPrimaryKey: isPK,
		}
		if defaultVal.Valid {
			col.Default = &defaultVal.String
		}
		if comment.Valid && comment.String != "" {
			col.Comment = &comment.String
		}
		columns = append(columns, col)
	}

	// Get table comment
	var tableComment *string
	commentQuery := `SELECT obj_description($1::regclass)`
	var tc sql.NullString
	if err := db.QueryRowContext(ctx, commentQuery, tableName).Scan(&tc); err == nil && tc.Valid && tc.String != "" {
		tableComment = &tc.String
	}

	return &models.TableInfo{
		Name:    table,
		Columns: columns,
		Comment: tableComment,
	}, nil
}

func (d *PostgreSQLDialect) GetViewInfo(ctx context.Context, db *sql.DB, database, view string) (*models.TableInfo, error) {
	return d.GetTableInfo(ctx, db, database, view)
}
