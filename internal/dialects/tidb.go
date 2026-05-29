package dialects

import (
	"fmt"

	"github.com/gnodux/adb-link/internal/models"
)

// TiDBDialect implements SchemaDialect for TiDB (MySQL-compatible).
type TiDBDialect struct {
	MySQLDialect
}

// BuildDSN returns a MySQL-compatible DSN for TiDB with default port 4000.
func (d *TiDBDialect) BuildDSN(config *models.DatasourceConfig, database string) string {
	port := config.Connection.Port
	if port == 0 {
		port = 4000
	}
	cfg := *config
	cfg.Connection.Port = port
	return d.MySQLDialect.BuildDSN(&cfg, database)
}
