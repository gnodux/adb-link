package dialects

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gnodux/adb-link/internal/models"
)

// ElasticsearchDialect implements SchemaDialect for Elasticsearch.
// Note: ES does not use database/sql. The DSN format is the base URL,
// and schema methods should be called via the ESClient adapter (see services).
type ElasticsearchDialect struct{}

func (d *ElasticsearchDialect) BuildDSN(config *models.DatasourceConfig, database string) string {
	conn := config.Connection
	scheme := "http"
	if v, ok := config.Options["use_ssl"].(bool); ok && v {
		scheme = "https"
	}
	port := conn.Port
	if port == 0 {
		port = 9200
	}
	return fmt.Sprintf("%s://%s:%d", scheme, conn.Host, port)
}

// ES dialect doesn't use *sql.DB; these methods exist to satisfy the interface
// but actual work is done via the dedicated ES client in services layer.
func (d *ElasticsearchDialect) GetDatabases(ctx context.Context, db *sql.DB) ([]models.ObjectName, error) {
	return nil, fmt.Errorf("elasticsearch does not use sql.DB; use ES client")
}

func (d *ElasticsearchDialect) GetTableNames(ctx context.Context, db *sql.DB, database string) ([]models.ObjectName, error) {
	return nil, fmt.Errorf("elasticsearch does not use sql.DB; use ES client")
}

func (d *ElasticsearchDialect) GetViewNames(ctx context.Context, db *sql.DB, database string) ([]models.ObjectName, error) {
	return nil, nil
}

func (d *ElasticsearchDialect) GetTableInfo(ctx context.Context, db *sql.DB, database, table string) (*models.TableInfo, error) {
	return nil, fmt.Errorf("elasticsearch does not use sql.DB; use ES client")
}

func (d *ElasticsearchDialect) GetViewInfo(ctx context.Context, db *sql.DB, database, view string) (*models.TableInfo, error) {
	return &models.TableInfo{Name: view, Columns: nil}, nil
}

func (d *ElasticsearchDialect) GetServerInfo(ctx context.Context, db *sql.DB) (*models.ServerInfo, error) {
	return nil, fmt.Errorf("elasticsearch does not use sql.DB; use ES client")
}
