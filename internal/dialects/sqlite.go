package dialects

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gnodux/adb-link/internal/models"
)

// SQLiteDialect implements SchemaDialect for SQLite.
type SQLiteDialect struct{}

func (d *SQLiteDialect) BuildDSN(config *models.DatasourceConfig, database string) string {
	path := config.Connection.Path
	if path == "" {
		path = ":memory:"
	}
	return path
}

func (d *SQLiteDialect) GetDatabases(ctx context.Context, db *sql.DB) ([]models.ObjectName, error) {
	return []models.ObjectName{{Name: "main", Comment: ""}}, nil
}

func (d *SQLiteDialect) GetTableNames(ctx context.Context, db *sql.DB, database string) ([]models.ObjectName, error) {
	query := `SELECT name, '' as comment FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObjectNames(rows)
}

func (d *SQLiteDialect) GetViewNames(ctx context.Context, db *sql.DB, database string) ([]models.ObjectName, error) {
	query := `SELECT name, '' as comment FROM sqlite_master WHERE type='view' ORDER BY name`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanObjectNames(rows)
}

func (d *SQLiteDialect) GetTableInfo(ctx context.Context, db *sql.DB, database, table string) (*models.TableInfo, error) {
	query := fmt.Sprintf(`PRAGMA table_info("%s")`, table)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []models.ColumnInfo
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var defaultVal sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultVal, &pk); err != nil {
			return nil, err
		}
		col := models.ColumnInfo{
			Name:         name,
			Type:         colType,
			Nullable:     notNull == 0,
			IsPrimaryKey: pk > 0,
		}
		if defaultVal.Valid {
			col.Default = &defaultVal.String
		}
		columns = append(columns, col)
	}

	return &models.TableInfo{
		Name:    table,
		Columns: columns,
	}, nil
}

func (d *SQLiteDialect) GetViewInfo(ctx context.Context, db *sql.DB, database, view string) (*models.TableInfo, error) {
	return d.GetTableInfo(ctx, db, database, view)
}

func (d *SQLiteDialect) GetServerInfo(ctx context.Context, db *sql.DB) (*models.ServerInfo, error) {
	info := &models.ServerInfo{}

	// Version
	var version string
	if err := db.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&version); err == nil {
		info.Version = version
	}

	return info, nil
}
