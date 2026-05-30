package dialects

import (
	"strings"
	"testing"

	"github.com/gnodux/adb-link/internal/models"
)

func TestElasticsearch_BuildDSN_DefaultSchemeHTTP(t *testing.T) {
	d := &ElasticsearchDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host: "localhost",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "")

	if !strings.HasPrefix(dsn, "http://") {
		t.Fatalf("expected default scheme http, got DSN: %s", dsn)
	}
}

func TestElasticsearch_BuildDSN_SSLSchemeHTTPS(t *testing.T) {
	d := &ElasticsearchDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host: "localhost",
		},
		Options: map[string]any{
			"use_ssl": true,
		},
	}
	dsn := d.BuildDSN(config, "")

	if !strings.HasPrefix(dsn, "https://") {
		t.Fatalf("expected https scheme when use_ssl=true, got DSN: %s", dsn)
	}
}

func TestElasticsearch_BuildDSN_DefaultPort(t *testing.T) {
	d := &ElasticsearchDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host: "localhost",
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "")

	expected := "http://localhost:9200"
	if dsn != expected {
		t.Fatalf("expected DSN %q, got %q", expected, dsn)
	}
}

func TestElasticsearch_BuildDSN_CustomPort(t *testing.T) {
	d := &ElasticsearchDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host: "localhost",
			Port: 9201,
		},
		Options: map[string]any{},
	}
	dsn := d.BuildDSN(config, "")

	expected := "http://localhost:9201"
	if dsn != expected {
		t.Fatalf("expected DSN %q, got %q", expected, dsn)
	}
}
