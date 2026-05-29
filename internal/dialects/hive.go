package dialects

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/gnodux/adb-link/internal/models"
)

// HiveDialect implements SchemaDialect for Apache Hive via HiveServer2.
// Uses sqlx-hive driver via database/sql.
type HiveDialect struct{}

func (d *HiveDialect) BuildDSN(config *models.DatasourceConfig, database string) string {
	conn := config.Connection
	host := conn.Host
	if host == "" {
		host = "localhost"
	}
	port := conn.Port
	if port == 0 {
		port = 10000
	}
	db := database
	if db == "" {
		db = conn.DefaultDatabase
	}
	if db == "" {
		db = "default"
	}
	auth := "NONE"
	if v, ok := config.Options["auth"].(string); ok && v != "" {
		auth = v
	}
	// Format: user:password@host:port/database?auth=NONE
	return fmt.Sprintf("%s:%s@%s:%d/%s?auth=%s",
		conn.Username, conn.Password, host, port, db, auth)
}

func (d *HiveDialect) GetDatabases(ctx context.Context, db *sql.DB) ([]models.ObjectName, error) {
	rows, err := db.QueryContext(ctx, "SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.ObjectName
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		result = append(result, models.ObjectName{Name: name})
	}
	return result, nil
}

func (d *HiveDialect) GetTableNames(ctx context.Context, db *sql.DB, database string) ([]models.ObjectName, error) {
	query := fmt.Sprintf("SHOW TABLES IN `%s`", database)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.ObjectName
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		result = append(result, models.ObjectName{Name: name})
	}
	return result, nil
}

func (d *HiveDialect) GetViewNames(ctx context.Context, db *sql.DB, database string) ([]models.ObjectName, error) {
	query := fmt.Sprintf("SHOW VIEWS IN `%s`", database)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		// SHOW VIEWS may not exist in older Hive
		return []models.ObjectName{}, nil
	}
	defer rows.Close()
	var result []models.ObjectName
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		result = append(result, models.ObjectName{Name: name})
	}
	return result, nil
}

func (d *HiveDialect) GetTableInfo(ctx context.Context, db *sql.DB, database, table string) (*models.TableInfo, error) {
	query := fmt.Sprintf("DESCRIBE `%s`.`%s`", database, table)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []models.ColumnInfo
	for rows.Next() {
		var colName, dataType, comment sql.NullString
		if err := rows.Scan(&colName, &dataType, &comment); err != nil {
			return nil, err
		}
		name := strings.TrimSpace(colName.String)
		// Stop at empty rows or section markers
		if name == "" || strings.HasPrefix(name, "#") {
			break
		}
		col := models.ColumnInfo{
			Name:     name,
			Type:     strings.TrimSpace(dataType.String),
			Nullable: true,
		}
		if comment.Valid {
			c := strings.TrimSpace(comment.String)
			if c != "" {
				col.Comment = &c
			}
		}
		columns = append(columns, col)
	}

	return &models.TableInfo{Name: table, Columns: columns}, nil
}

func (d *HiveDialect) GetViewInfo(ctx context.Context, db *sql.DB, database, view string) (*models.TableInfo, error) {
	return d.GetTableInfo(ctx, db, database, view)
}
