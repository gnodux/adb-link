package dialects

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gnodux/adb-link/internal/models"
)

// MilvusDialect is a stub SchemaDialect for Milvus.
// Milvus does not use database/sql; all operations go through MilvusClient.
type MilvusDialect struct{}

func (d *MilvusDialect) BuildDSN(config *models.DatasourceConfig, database string) string {
	conn := config.Connection
	port := conn.Port
	if port == 0 {
		port = 19530
	}
	return fmt.Sprintf("%s:%d", conn.Host, port)
}

func (d *MilvusDialect) GetDatabases(_ context.Context, _ *sql.DB) ([]models.ObjectName, error) {
	return nil, fmt.Errorf("milvus does not use sql.DB; use Milvus client")
}

func (d *MilvusDialect) GetTableNames(_ context.Context, _ *sql.DB, _ string) ([]models.ObjectName, error) {
	return nil, fmt.Errorf("milvus does not use sql.DB; use Milvus client")
}

func (d *MilvusDialect) GetViewNames(_ context.Context, _ *sql.DB, _ string) ([]models.ObjectName, error) {
	return nil, nil
}

func (d *MilvusDialect) GetTableInfo(_ context.Context, _ *sql.DB, _, _ string) (*models.TableInfo, error) {
	return nil, fmt.Errorf("milvus does not use sql.DB; use Milvus client")
}

func (d *MilvusDialect) GetViewInfo(_ context.Context, _ *sql.DB, _, _ string) (*models.TableInfo, error) {
	return &models.TableInfo{}, nil
}
