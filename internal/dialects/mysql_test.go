package dialects

import (
	"strings"
	"testing"

	"github.com/gnodux/adb-link/internal/models"
)

func TestMySQL_BuildDSN_DefaultPort(t *testing.T) {
	d := &MySQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "tcp(localhost:3306)") {
		t.Fatalf("expected default port 3306, got DSN: %s", dsn)
	}
}

func TestMySQL_BuildDSN_CustomPort(t *testing.T) {
	d := &MySQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Port:     3307,
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "tcp(localhost:3307)") {
		t.Fatalf("expected custom port 3307, got DSN: %s", dsn)
	}
}

func TestMySQL_BuildDSN_DefaultCharset(t *testing.T) {
	d := &MySQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "charset=utf8mb4") {
		t.Fatalf("expected default charset utf8mb4, got DSN: %s", dsn)
	}
}

func TestMySQL_BuildDSN_CustomCharset(t *testing.T) {
	d := &MySQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{
			"charset": "latin1",
		},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "charset=latin1") {
		t.Fatalf("expected custom charset latin1, got DSN: %s", dsn)
	}
}

func TestMySQL_BuildDSN_CustomTimeouts(t *testing.T) {
	d := &MySQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{
			"connect_timeout": 10,
			"read_timeout":    120,
			"write_timeout":   90,
		},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "timeout=10s") {
		t.Fatalf("expected connect_timeout 10s, got DSN: %s", dsn)
	}
	if !strings.Contains(dsn, "readTimeout=120s") {
		t.Fatalf("expected readTimeout 120s, got DSN: %s", dsn)
	}
	if !strings.Contains(dsn, "writeTimeout=90s") {
		t.Fatalf("expected writeTimeout 90s, got DSN: %s", dsn)
	}
}

func TestMySQL_BuildDSN_EmptyDatabaseFallsToDefault(t *testing.T) {
	d := &MySQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:            "localhost",
			Username:        "user",
			Password:        "pass",
			DefaultDatabase: "defaultdb",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "")

	if !strings.Contains(dsn, "/defaultdb?") {
		t.Fatalf("expected database to fall back to DefaultDatabase 'defaultdb', got DSN: %s", dsn)
	}
}

func TestMySQL_BuildDSN_ExplicitDatabaseOverridesDefault(t *testing.T) {
	d := &MySQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:            "localhost",
			Username:        "user",
			Password:        "pass",
			DefaultDatabase: "defaultdb",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "explicitdb")

	if !strings.Contains(dsn, "/explicitdb?") {
		t.Fatalf("expected explicit database 'explicitdb', got DSN: %s", dsn)
	}
	if strings.Contains(dsn, "/defaultdb?") {
		t.Fatalf("explicit database should override DefaultDatabase, got DSN: %s", dsn)
	}
}

func TestMySQL_BuildDSN_ContainsParseTime(t *testing.T) {
	d := &MySQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "parseTime=true") {
		t.Fatalf("expected parseTime=true in DSN, got: %s", dsn)
	}
}
