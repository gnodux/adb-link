package dialects

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/gnodux/adb-link/internal/models"
)

// MSSQLDialect implements SchemaDialect for SQL Server.
type MSSQLDialect struct{}

func (d *MSSQLDialect) BuildDSN(config *models.DatasourceConfig, database string) string {
	conn := config.Connection
	port := conn.Port
	if port == 0 {
		port = 1433
	}
	db := database
	if db == "" {
		db = conn.DefaultDatabase
	}
	params := url.Values{}
	params.Set("database", db)
	if v, ok := config.Options["trust_server_certificate"]; ok {
		if b, ok := v.(bool); ok && b {
			params.Set("trustservercertificate", "true")
		}
	} else {
		params.Set("trustservercertificate", "true")
	}
	// Defaults: prevent indefinite hangs.
	// go-mssqldb: 'dial timeout' (TCP), 'connection timeout' (login).
	dialTimeout := "5"
	if v, ok := config.Options["connect_timeout"]; ok {
		dialTimeout = fmt.Sprintf("%v", v)
	}
	params.Set("dial timeout", dialTimeout)
	if v, ok := config.Options["connection_timeout"]; ok {
		params.Set("connection timeout", fmt.Sprintf("%v", v))
	} else {
		params.Set("connection timeout", "60")
	}
	return fmt.Sprintf("sqlserver://%s:%s@%s:%d?%s",
		url.PathEscape(conn.Username), url.PathEscape(conn.Password),
		conn.Host, port, params.Encode())
}

func (d *MSSQLDialect) GetDatabases(ctx context.Context, db *sql.DB) ([]models.ObjectName, error) {
	query := `SELECT name, '' as comment FROM sys.databases WHERE state = 0 ORDER BY name`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObjectNames(rows)
}

func (d *MSSQLDialect) GetTableNames(ctx context.Context, db *sql.DB, database string) ([]models.ObjectName, error) {
	query := `SELECT t.name, ISNULL(CAST(ep.value AS NVARCHAR(MAX)), '') as comment 
		FROM sys.tables t 
		LEFT JOIN sys.extended_properties ep 
		  ON ep.major_id = t.object_id AND ep.minor_id = 0 AND ep.name = 'MS_Description' 
		ORDER BY t.name`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObjectNames(rows)
}

func (d *MSSQLDialect) GetViewNames(ctx context.Context, db *sql.DB, database string) ([]models.ObjectName, error) {
	query := `SELECT v.name, ISNULL(CAST(ep.value AS NVARCHAR(MAX)), '') as comment 
		FROM sys.views v 
		LEFT JOIN sys.extended_properties ep 
		  ON ep.major_id = v.object_id AND ep.minor_id = 0 AND ep.name = 'MS_Description' 
		ORDER BY v.name`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObjectNames(rows)
}

func (d *MSSQLDialect) GetTableInfo(ctx context.Context, db *sql.DB, database, table string) (*models.TableInfo, error) {
	query := `SELECT c.COLUMN_NAME, c.DATA_TYPE, c.IS_NULLABLE, c.COLUMN_DEFAULT, 
		CAST(ep.value AS NVARCHAR(MAX)) AS column_comment, 
		CASE WHEN pk.COLUMN_NAME IS NOT NULL THEN 1 ELSE 0 END AS is_pk 
		FROM INFORMATION_SCHEMA.COLUMNS c 
		LEFT JOIN sys.columns sc 
		  ON sc.name = c.COLUMN_NAME 
		  AND sc.object_id = OBJECT_ID(QUOTENAME(c.TABLE_SCHEMA) + '.' + QUOTENAME(c.TABLE_NAME)) 
		LEFT JOIN sys.extended_properties ep 
		  ON ep.major_id = sc.object_id AND ep.minor_id = sc.column_id 
		  AND ep.name = 'MS_Description' 
		LEFT JOIN (
		  SELECT ku.TABLE_CATALOG, ku.TABLE_SCHEMA, ku.TABLE_NAME, ku.COLUMN_NAME 
		  FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc 
		  JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE ku 
		    ON tc.CONSTRAINT_NAME = ku.CONSTRAINT_NAME 
		  WHERE tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
		) pk ON pk.TABLE_CATALOG = c.TABLE_CATALOG 
		  AND pk.TABLE_SCHEMA = c.TABLE_SCHEMA 
		  AND pk.TABLE_NAME = c.TABLE_NAME 
		  AND pk.COLUMN_NAME = c.COLUMN_NAME 
		WHERE c.TABLE_CATALOG = @p1 AND c.TABLE_NAME = @p2 
		ORDER BY c.ORDINAL_POSITION`
	rows, err := db.QueryContext(ctx, query, database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []models.ColumnInfo
	for rows.Next() {
		var name, colType, nullable string
		var defaultVal, comment sql.NullString
		var isPK int
		if err := rows.Scan(&name, &colType, &nullable, &defaultVal, &comment, &isPK); err != nil {
			return nil, err
		}
		col := models.ColumnInfo{
			Name:         name,
			Type:         colType,
			Nullable:     nullable == "YES",
			IsPrimaryKey: isPK == 1,
		}
		if defaultVal.Valid {
			col.Default = &defaultVal.String
		}
		if comment.Valid && comment.String != "" {
			col.Comment = &comment.String
		}
		columns = append(columns, col)
	}

	var tableComment *string
	commentQuery := `SELECT CAST(ep.value AS NVARCHAR(MAX)) 
		FROM sys.extended_properties ep 
		JOIN sys.tables t ON ep.major_id = t.object_id 
		WHERE ep.minor_id = 0 AND ep.name = 'MS_Description' AND t.name = @p1`
	var tc sql.NullString
	if err := db.QueryRowContext(ctx, commentQuery, table).Scan(&tc); err == nil && tc.Valid && tc.String != "" {
		tableComment = &tc.String
	}

	return &models.TableInfo{Name: table, Columns: columns, Comment: tableComment}, nil
}

func (d *MSSQLDialect) GetViewInfo(ctx context.Context, db *sql.DB, database, view string) (*models.TableInfo, error) {
	return d.GetTableInfo(ctx, db, database, view)
}

func (d *MSSQLDialect) GetServerInfo(ctx context.Context, db *sql.DB) (*models.ServerInfo, error) {
	info := &models.ServerInfo{}

	// Version
	var version string
	if err := db.QueryRowContext(ctx, "SELECT @@VERSION").Scan(&version); err == nil {
		info.Version = version
	}

	return info, nil
}
