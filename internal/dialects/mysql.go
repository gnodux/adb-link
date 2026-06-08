package dialects

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/gnodux/adb-link/internal/models"
)

// MySQLDialect implements SchemaDialect for MySQL.
type MySQLDialect struct{}

func (d *MySQLDialect) BuildDSN(config *models.DatasourceConfig, database string) string {
	conn := config.Connection
	password := conn.Password
	username := conn.Username
	port := conn.Port
	if port == 0 {
		port = 3306
	}
	db := database
	if db == "" {
		db = conn.DefaultDatabase
	}
	charset := "utf8mb4"
	if v, ok := config.Options["charset"].(string); ok && v != "" {
		charset = v
	}
	// Go MySQL DSN format: user:password@tcp(host:port)/dbname?params
	params := url.Values{}
	params.Set("charset", charset)
	params.Set("parseTime", "true")
	params.Set("loc", "Local")
	// Defaults: prevent indefinite hangs on unreachable hosts
	connectTimeout := "5s"
	readTimeout := "60s"
	writeTimeout := "60s"
	if v, ok := config.Options["connect_timeout"]; ok {
		connectTimeout = fmt.Sprintf("%vs", v)
	}
	if v, ok := config.Options["read_timeout"]; ok {
		readTimeout = fmt.Sprintf("%vs", v)
	}
	if v, ok := config.Options["write_timeout"]; ok {
		writeTimeout = fmt.Sprintf("%vs", v)
	}
	params.Set("timeout", connectTimeout)
	params.Set("readTimeout", readTimeout)
	params.Set("writeTimeout", writeTimeout)
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		username, password, conn.Host, port, db, params.Encode())
}

func (d *MySQLDialect) GetDatabases(ctx context.Context, db *sql.DB) ([]models.ObjectName, error) {
	query := `SELECT SCHEMA_NAME as name, '' as comment 
		FROM information_schema.SCHEMATA 
		WHERE SCHEMA_NAME NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys') 
		ORDER BY SCHEMA_NAME`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObjectNames(rows)
}

func (d *MySQLDialect) GetTableNames(ctx context.Context, db *sql.DB, database string) ([]models.ObjectName, error) {
	query := `SELECT TABLE_NAME as name, IFNULL(TABLE_COMMENT, '') as comment 
		FROM information_schema.TABLES 
		WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE' 
		ORDER BY TABLE_NAME`
	rows, err := db.QueryContext(ctx, query, database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObjectNames(rows)
}

func (d *MySQLDialect) GetViewNames(ctx context.Context, db *sql.DB, database string) ([]models.ObjectName, error) {
	query := `SELECT TABLE_NAME as name, IFNULL(TABLE_COMMENT, '') as comment 
		FROM information_schema.TABLES 
		WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'VIEW' 
		ORDER BY TABLE_NAME`
	rows, err := db.QueryContext(ctx, query, database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObjectNames(rows)
}

func (d *MySQLDialect) GetTableInfo(ctx context.Context, db *sql.DB, database, table string) (*models.TableInfo, error) {
	query := `SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_DEFAULT, COLUMN_COMMENT, COLUMN_KEY 
		FROM information_schema.COLUMNS 
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? 
		ORDER BY ORDINAL_POSITION`
	rows, err := db.QueryContext(ctx, query, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []models.ColumnInfo
	for rows.Next() {
		var name, colType, nullable, colKey string
		var defaultVal, comment sql.NullString
		if err := rows.Scan(&name, &colType, &nullable, &defaultVal, &comment, &colKey); err != nil {
			return nil, err
		}
		col := models.ColumnInfo{
			Name:         name,
			Type:         colType,
			Nullable:     nullable == "YES",
			IsPrimaryKey: colKey == "PRI",
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
	commentQuery := `SELECT TABLE_COMMENT FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?`
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

func (d *MySQLDialect) GetViewInfo(ctx context.Context, db *sql.DB, database, view string) (*models.TableInfo, error) {
	return d.GetTableInfo(ctx, db, database, view)
}

func (d *MySQLDialect) GetServerInfo(ctx context.Context, db *sql.DB) (*models.ServerInfo, error) {
	info := &models.ServerInfo{}

	// Version
	var version string
	if err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version); err == nil {
		info.Version = version
	}

	// SQL Mode
	var sqlMode string
	if err := db.QueryRowContext(ctx, "SELECT @@sql_mode").Scan(&sqlMode); err == nil {
		info.SQLMode = sqlMode
	}

	// Timezone
	var timezone string
	if err := db.QueryRowContext(ctx, "SELECT @@system_time_zone").Scan(&timezone); err == nil {
		info.Timezone = timezone
	}

	return info, nil
}

func scanObjectNames(rows *sql.Rows) ([]models.ObjectName, error) {
	var result []models.ObjectName
	for rows.Next() {
		var name, comment string
		if err := rows.Scan(&name, &comment); err != nil {
			return nil, err
		}
		result = append(result, models.ObjectName{Name: name, Comment: comment})
	}
	return result, rows.Err()
}
