package dialects

import (
	"testing"

	"github.com/gnodux/adb-link/internal/models"
)

func TestGetDialect_MySQL(t *testing.T) {
	d, err := GetDialect(models.DatabaseTypeMySQL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := d.(*MySQLDialect); !ok {
		t.Fatalf("expected *MySQLDialect, got %T", d)
	}
}

func TestGetDialect_PostgreSQL(t *testing.T) {
	d, err := GetDialect(models.DatabaseTypePostgreSQL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := d.(*PostgreSQLDialect); !ok {
		t.Fatalf("expected *PostgreSQLDialect, got %T", d)
	}
}

func TestGetDialect_SQLite(t *testing.T) {
	d, err := GetDialect(models.DatabaseTypeSQLite)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := d.(*SQLiteDialect); !ok {
		t.Fatalf("expected *SQLiteDialect, got %T", d)
	}
}

func TestGetDialect_ClickHouse(t *testing.T) {
	d, err := GetDialect(models.DatabaseTypeClickHouse)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := d.(*ClickHouseDialect); !ok {
		t.Fatalf("expected *ClickHouseDialect, got %T", d)
	}
}

func TestGetDialect_MSSQL(t *testing.T) {
	d, err := GetDialect(models.DatabaseTypeMSSQL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := d.(*MSSQLDialect); !ok {
		t.Fatalf("expected *MSSQLDialect, got %T", d)
	}
}

func TestGetDialect_Elasticsearch(t *testing.T) {
	d, err := GetDialect(models.DatabaseTypeElasticsearch)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := d.(*ElasticsearchDialect); !ok {
		t.Fatalf("expected *ElasticsearchDialect, got %T", d)
	}
}

func TestGetDialect_Hive(t *testing.T) {
	d, err := GetDialect(models.DatabaseTypeHive)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if _, ok := d.(*HiveDialect); !ok {
		t.Fatalf("expected *HiveDialect, got %T", d)
	}
}

func TestGetDialect_Unsupported(t *testing.T) {
	_, err := GetDialect(models.DatabaseType("db2"))
	if err == nil {
		t.Fatal("expected error for unsupported type, got nil")
	}
	expected := "unsupported database type: db2"
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err.Error())
	}
}
