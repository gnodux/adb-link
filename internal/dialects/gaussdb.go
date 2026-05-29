package dialects

import (
	"github.com/gnodux/adb-link/internal/models"
)

// GaussDBDialect implements SchemaDialect for Huawei GaussDB (PostgreSQL-compatible).
type GaussDBDialect struct {
	PostgreSQLDialect
}

// BuildDSN returns a PostgreSQL-compatible DSN for GaussDB.
func (d *GaussDBDialect) BuildDSN(config *models.DatasourceConfig, database string) string {
	return d.PostgreSQLDialect.BuildDSN(config, database)
}
