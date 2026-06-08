package services

import (
	"context"

	"github.com/gnodux/adb-link/internal/models"
)

// NonSQLClient is the interface for databases that do not use database/sql.
// Implementations provide schema discovery and query execution through native
// client libraries instead of the standard SQL interface.
type NonSQLClient interface {
	Ping(ctx context.Context) error
	Close() error
	GetDatabases(ctx context.Context) ([]models.ObjectName, error)
	GetTableNames(ctx context.Context, database string) ([]models.ObjectName, error)
	GetTableInfo(ctx context.Context, database, table string) (*models.TableInfo, error)
	Execute(ctx context.Context, database, query string, limit int) (*models.QueryResult, error)
	GetServerInfo(ctx context.Context) (*models.ServerInfo, error)
}

// IsNonSQLType returns true for database types that use a NonSQLClient
// instead of database/sql.
func IsNonSQLType(t models.DatabaseType) bool {
	switch t {
	case models.DatabaseTypeElasticsearch,
		models.DatabaseTypeRedis,
		models.DatabaseTypeMongoDB,
		models.DatabaseTypeMilvus:
		return true
	}
	return false
}
