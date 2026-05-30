package dialects

import (
	"strings"
	"testing"

	"github.com/gnodux/adb-link/internal/models"
)

func TestHive_BuildDSN_DefaultHost(t *testing.T) {
	d := &HiveDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "@localhost:") {
		t.Fatalf("expected default host 'localhost', got DSN: %s", dsn)
	}
}

func TestHive_BuildDSN_DefaultPort(t *testing.T) {
	d := &HiveDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "myhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "myhost:10000") {
		t.Fatalf("expected default port 10000, got DSN: %s", dsn)
	}
}

func TestHive_BuildDSN_DefaultDatabase(t *testing.T) {
	d := &HiveDialect{}
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

func TestHive_BuildDSN_DefaultAuth(t *testing.T) {
	d := &HiveDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "auth=NONE") {
		t.Fatalf("expected default auth=NONE, got DSN: %s", dsn)
	}
}

func TestHive_BuildDSN_CustomAuth(t *testing.T) {
	d := &HiveDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{
			"auth": "KERBEROS",
		},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "auth=KERBEROS") {
		t.Fatalf("expected auth=KERBEROS, got DSN: %s", dsn)
	}
}
