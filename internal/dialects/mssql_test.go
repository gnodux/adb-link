package dialects

import (
	"strings"
	"testing"

	"github.com/gnodux/adb-link/internal/models"
)

func TestMSSQL_BuildDSN_DefaultPort(t *testing.T) {
	d := &MSSQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "localhost:1433") {
		t.Fatalf("expected default port 1433, got DSN: %s", dsn)
	}
}

func TestMSSQL_BuildDSN_DatabaseInQueryParams(t *testing.T) {
	d := &MSSQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "database=testdb") {
		t.Fatalf("expected database in query params, got DSN: %s", dsn)
	}
	// The path portion should be empty (no /db in path)
	if !strings.Contains(dsn, "@localhost:1433?") {
		t.Fatalf("expected no database in URL path, got DSN: %s", dsn)
	}
}

func TestMSSQL_BuildDSN_TrustServerCertificateDefault(t *testing.T) {
	d := &MSSQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "trustservercertificate=true") {
		t.Fatalf("expected trustservercertificate=true by default, got DSN: %s", dsn)
	}
}

func TestMSSQL_BuildDSN_ExplicitTrustServerCertificateFalse(t *testing.T) {
	d := &MSSQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{
			"trust_server_certificate": false,
		},
	}
	dsn := d.BuildDSN(config, "testdb")

	if strings.Contains(dsn, "trustservercertificate") {
		t.Fatalf("expected trustservercertificate to be absent when set to false, got DSN: %s", dsn)
	}
}

func TestMSSQL_BuildDSN_DefaultDialTimeout(t *testing.T) {
	d := &MSSQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	// url.Values.Encode will encode the space as +
	if !strings.Contains(dsn, "dial+timeout=5") {
		t.Fatalf("expected default dial timeout=5, got DSN: %s", dsn)
	}
}

func TestMSSQL_BuildDSN_DefaultConnectionTimeout(t *testing.T) {
	d := &MSSQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "connection+timeout=60") {
		t.Fatalf("expected default connection timeout=60, got DSN: %s", dsn)
	}
}
