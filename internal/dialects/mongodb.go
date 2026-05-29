package dialects

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gnodux/adb-link/internal/models"
)

// MongoDBDialect is a stub SchemaDialect for MongoDB.
// MongoDB does not use database/sql; all operations go through MongoClient.
type MongoDBDialect struct{}

func (d *MongoDBDialect) BuildDSN(config *models.DatasourceConfig, database string) string {
	conn := config.Connection
	port := conn.Port
	if port == 0 {
		port = 27017
	}
	return fmt.Sprintf("mongodb://%s:%d", conn.Host, port)
}

func (d *MongoDBDialect) GetDatabases(_ context.Context, _ *sql.DB) ([]models.ObjectName, error) {
	return nil, fmt.Errorf("mongodb does not use sql.DB; use MongoDB client")
}

func (d *MongoDBDialect) GetTableNames(_ context.Context, _ *sql.DB, _ string) ([]models.ObjectName, error) {
	return nil, fmt.Errorf("mongodb does not use sql.DB; use MongoDB client")
}

func (d *MongoDBDialect) GetViewNames(_ context.Context, _ *sql.DB, _ string) ([]models.ObjectName, error) {
	return nil, nil
}

func (d *MongoDBDialect) GetTableInfo(_ context.Context, _ *sql.DB, _, _ string) (*models.TableInfo, error) {
	return nil, fmt.Errorf("mongodb does not use sql.DB; use MongoDB client")
}

func (d *MongoDBDialect) GetViewInfo(_ context.Context, _ *sql.DB, _, _ string) (*models.TableInfo, error) {
	return &models.TableInfo{}, nil
}
