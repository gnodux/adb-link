package dialects

import (
	"strings"
	"testing"

	"github.com/gnodux/adb-link/internal/models"
)

func TestClickHouse_BuildDSN_DefaultPort(t *testing.T) {
	d := &ClickHouseDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "localhost:9000") {
		t.Fatalf("expected default port 9000, got DSN: %s", dsn)
	}
}

func TestClickHouse_BuildDSN_DefaultUsername(t *testing.T) {
	d := &ClickHouseDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.HasPrefix(dsn, "clickhouse://default:") {
		t.Fatalf("expected default username 'default', got DSN: %s", dsn)
	}
}

func TestClickHouse_BuildDSN_DefaultDatabase(t *testing.T) {
	d := &ClickHouseDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "")

	if !strings.Contains(dsn, "/default?") {
		t.Fatalf("expected default database 'default', got DSN: %s", dsn)
	}
}

func TestClickHouse_BuildDSN_CustomTimeouts(t *testing.T) {
	d := &ClickHouseDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{
			"connect_timeout": 10,
			"read_timeout":    120,
		},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "dial_timeout=10s") {
		t.Fatalf("expected dial_timeout=10s, got DSN: %s", dsn)
	}
	if !strings.Contains(dsn, "read_timeout=120s") {
		t.Fatalf("expected read_timeout=120s, got DSN: %s", dsn)
	}
}

func TestClickHouse_BuildDSN_DefaultReadTimeout(t *testing.T) {
	d := &ClickHouseDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "read_timeout=60s") {
		t.Fatalf("expected default read_timeout=60s, got DSN: %s", dsn)
	}
}
