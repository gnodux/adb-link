package dialects

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gnodux/adb-link/internal/models"
)

// RedisDialect is a stub SchemaDialect for Redis.
// Redis does not use database/sql; all operations go through RedisClient.
type RedisDialect struct{}

func (d *RedisDialect) BuildDSN(config *models.DatasourceConfig, database string) string {
	conn := config.Connection
	port := conn.Port
	if port == 0 {
		port = 6379
	}
	return fmt.Sprintf("redis://%s:%d", conn.Host, port)
}

func (d *RedisDialect) GetDatabases(_ context.Context, _ *sql.DB) ([]models.ObjectName, error) {
	return nil, fmt.Errorf("redis does not use sql.DB; use Redis client")
}

func (d *RedisDialect) GetTableNames(_ context.Context, _ *sql.DB, _ string) ([]models.ObjectName, error) {
	return nil, fmt.Errorf("redis does not use sql.DB; use Redis client")
}

func (d *RedisDialect) GetViewNames(_ context.Context, _ *sql.DB, _ string) ([]models.ObjectName, error) {
	return nil, nil
}

func (d *RedisDialect) GetTableInfo(_ context.Context, _ *sql.DB, _, _ string) (*models.TableInfo, error) {
	return nil, fmt.Errorf("redis does not use sql.DB; use Redis client")
}

func (d *RedisDialect) GetViewInfo(_ context.Context, _ *sql.DB, _, _ string) (*models.TableInfo, error) {
	return &models.TableInfo{}, nil
}

func (d *RedisDialect) GetServerInfo(_ context.Context, _ *sql.DB) (*models.ServerInfo, error) {
	return nil, fmt.Errorf("redis does not use sql.DB; use Redis client")
}
