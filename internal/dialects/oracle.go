package dialects

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gnodux/adb-link/internal/models"
)

// OracleDialect implements SchemaDialect for Oracle Database.
type OracleDialect struct{}

// BuildDSN returns a go-ora compatible DSN.
func (d *OracleDialect) BuildDSN(config *models.DatasourceConfig, database string) string {
	conn := config.Connection
	host := conn.Host
	port := conn.Port
	if port == 0 {
		port = 1521
	}
	service := database
	if service == "" {
		service = conn.DefaultDatabase
	}
	if service == "" {
		service = "ORCL"
	}
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		conn.Username, conn.Password, host, port, service)
}

// GetDatabases returns Oracle schemas as virtual databases.
func (d *OracleDialect) GetDatabases(ctx context.Context, db *sql.DB) ([]models.ObjectName, error) {
	rows, err := db.QueryContext(ctx, "SELECT USERNAME FROM ALL_USERS ORDER BY USERNAME")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObjectNames(rows)
}

// GetTableNames returns table names for a given schema (owner).
func (d *OracleDialect) GetTableNames(ctx context.Context, db *sql.DB, database string) ([]models.ObjectName, error) {
	rows, err := db.QueryContext(ctx,
		"SELECT TABLE_NAME, '' FROM ALL_TABLES WHERE OWNER = :1 ORDER BY TABLE_NAME",
		database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObjectNames(rows)
}

// GetViewNames returns view names for a given schema (owner).
func (d *OracleDialect) GetViewNames(ctx context.Context, db *sql.DB, database string) ([]models.ObjectName, error) {
	rows, err := db.QueryContext(ctx,
		"SELECT VIEW_NAME, '' FROM ALL_VIEWS WHERE OWNER = :1 ORDER BY VIEW_NAME",
		database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObjectNames(rows)
}

// GetTableInfo returns column info for a table in a given schema.
func (d *OracleDialect) GetTableInfo(ctx context.Context, db *sql.DB, database, table string) (*models.TableInfo, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			c.COLUMN_NAME,
			c.DATA_TYPE ||
				CASE
					WHEN c.DATA_PRECISION IS NOT NULL THEN '(' || c.DATA_PRECISION || ',' || c.DATA_SCALE || ')'
					WHEN c.CHAR_LENGTH > 0 THEN '(' || c.CHAR_LENGTH || ')'
					ELSE ''
				END,
			CASE WHEN c.NULLABLE = 'Y' THEN 1 ELSE 0 END,
			c.DATA_DEFAULT,
			''
		FROM ALL_TAB_COLUMNS c
		WHERE c.OWNER = :1 AND c.TABLE_NAME = :2
		ORDER BY c.COLUMN_ID`,
		database, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []models.ColumnInfo
	for rows.Next() {
		var col models.ColumnInfo
		var name, typeName, comment string
		var nullable int
		var defaultVal sql.NullString
		if err := rows.Scan(&name, &typeName, &nullable, &defaultVal, &comment); err != nil {
			return nil, err
		}
		col.Name = name
		col.Type = typeName
		col.Nullable = nullable == 1
		if defaultVal.Valid && defaultVal.String != "" {
			col.Default = &defaultVal.String
		}
		columns = append(columns, col)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Detect primary key columns
	pkRows, err := db.QueryContext(ctx, `
		SELECT cc.COLUMN_NAME
		FROM ALL_CONSTRAINTS c
		JOIN ALL_CONS_COLUMNS cc ON c.CONSTRAINT_NAME = cc.CONSTRAINT_NAME AND c.OWNER = cc.OWNER
		WHERE c.OWNER = :1 AND c.TABLE_NAME = :2 AND c.CONSTRAINT_TYPE = 'P'`,
		database, table)
	if err == nil {
		pkSet := map[string]bool{}
		for pkRows.Next() {
			var col string
			if err := pkRows.Scan(&col); err == nil {
				pkSet[col] = true
			}
		}
		pkRows.Close()
		for i := range columns {
			if pkSet[columns[i].Name] {
				columns[i].IsPrimaryKey = true
			}
		}
	}

	return &models.TableInfo{Name: table, Columns: columns}, nil
}

// GetViewInfo returns column info for a view.
func (d *OracleDialect) GetViewInfo(ctx context.Context, db *sql.DB, database, view string) (*models.TableInfo, error) {
	return d.GetTableInfo(ctx, db, database, view)
}
