package dialects

import (
	"strings"
	"testing"

	"github.com/gnodux/adb-link/internal/models"
)

func TestPostgreSQL_BuildDSN_DefaultPort(t *testing.T) {
	d := &PostgreSQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "localhost:5432") {
		t.Fatalf("expected default port 5432, got DSN: %s", dsn)
	}
}

func TestPostgreSQL_BuildDSN_CustomPort(t *testing.T) {
	d := &PostgreSQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Port:     5433,
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "localhost:5433") {
		t.Fatalf("expected custom port 5433, got DSN: %s", dsn)
	}
}

func TestPostgreSQL_BuildDSN_DefaultDatabase(t *testing.T) {
	d := &PostgreSQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "")

	if !strings.Contains(dsn, "/postgres?") {
		t.Fatalf("expected default database 'postgres', got DSN: %s", dsn)
	}
}

func TestPostgreSQL_BuildDSN_ExplicitDatabase(t *testing.T) {
	d := &PostgreSQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "mydb")

	if !strings.Contains(dsn, "/mydb?") {
		t.Fatalf("expected explicit database 'mydb', got DSN: %s", dsn)
	}
}

func TestPostgreSQL_BuildDSN_SSLModeDisableDefault(t *testing.T) {
	d := &PostgreSQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "sslmode=disable") {
		t.Fatalf("expected sslmode=disable by default, got DSN: %s", dsn)
	}
}

func TestPostgreSQL_BuildDSN_CustomSSLMode(t *testing.T) {
	d := &PostgreSQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{
			"sslmode": "require",
		},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "sslmode=require") {
		t.Fatalf("expected sslmode=require, got DSN: %s", dsn)
	}
}

func TestPostgreSQL_BuildDSN_DefaultConnectTimeout(t *testing.T) {
	d := &PostgreSQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "connect_timeout=5") {
		t.Fatalf("expected default connect_timeout=5, got DSN: %s", dsn)
	}
}

func TestPostgreSQL_BuildDSN_DefaultStatementTimeout(t *testing.T) {
	d := &PostgreSQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user",
			Password: "pass",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	if !strings.Contains(dsn, "statement_timeout=60000") {
		t.Fatalf("expected default statement_timeout=60000, got DSN: %s", dsn)
	}
}

func TestPostgreSQL_BuildDSN_URLEscapingOfCredentials(t *testing.T) {
	d := &PostgreSQLDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host:     "localhost",
			Username: "user name",
			Password: "p@ss/word",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "testdb")

	// url.PathEscape encodes space as %20 and / as %2F, but @ is valid in path segments
	if !strings.Contains(dsn, "user%20name") {
		t.Fatalf("expected URL-escaped username with %%20 for space, got DSN: %s", dsn)
	}
	if !strings.Contains(dsn, "p@ss%2Fword") {
		t.Fatalf("expected URL-escaped password with %%2F for slash, got DSN: %s", dsn)
	}
}
