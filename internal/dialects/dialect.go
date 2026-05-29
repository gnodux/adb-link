package dialects

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gnodux/adb-link/internal/models"
)

// SchemaDialect is the interface for database-specific schema introspection.
type SchemaDialect interface {
	// BuildDSN returns a Go database/sql compatible DSN string.
	BuildDSN(config *models.DatasourceConfig, database string) string
	// GetDatabases returns the list of databases.
	GetDatabases(ctx context.Context, db *sql.DB) ([]models.ObjectName, error)
	// GetTableNames returns table names in a database.
	GetTableNames(ctx context.Context, db *sql.DB, database string) ([]models.ObjectName, error)
	// GetViewNames returns view names in a database.
	GetViewNames(ctx context.Context, db *sql.DB, database string) ([]models.ObjectName, error)
	// GetTableInfo returns detailed column info for a table.
	GetTableInfo(ctx context.Context, db *sql.DB, database, table string) (*models.TableInfo, error)
	// GetViewInfo returns detailed column info for a view.
	GetViewInfo(ctx context.Context, db *sql.DB, database, view string) (*models.TableInfo, error)
}

// GetDialect returns the appropriate dialect for a database type.
func GetDialect(dbType models.DatabaseType) (SchemaDialect, error) {
	switch dbType {
	case models.DatabaseTypeMySQL:
		return &MySQLDialect{}, nil
	case models.DatabaseTypePostgreSQL:
		return &PostgreSQLDialect{}, nil
	case models.DatabaseTypeSQLite:
		return &SQLiteDialect{}, nil
	case models.DatabaseTypeClickHouse:
		return &ClickHouseDialect{}, nil
	case models.DatabaseTypeMSSQL:
		return &MSSQLDialect{}, nil
	case models.DatabaseTypeElasticsearch:
		return &ElasticsearchDialect{}, nil
	case models.DatabaseTypeHive:
		return &HiveDialect{}, nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}
