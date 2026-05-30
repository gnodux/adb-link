package dialects

import (
	"testing"

	"github.com/gnodux/adb-link/internal/models"
)

func TestSQLite_BuildDSN_PathSet(t *testing.T) {
	d := &SQLiteDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Path: "/data/mydb.sqlite3",
		},
	}
	dsn := d.BuildDSN(config, "")

	expected := "/data/mydb.sqlite3"
	if dsn != expected {
		t.Fatalf("expected DSN %q, got %q", expected, dsn)
	}
}

func TestSQLite_BuildDSN_EmptyPath(t *testing.T) {
	d := &SQLiteDialect{}
	config := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{},
	}
	dsn := d.BuildDSN(config, "")

	expected := ":memory:"
	if dsn != expected {
		t.Fatalf("expected DSN %q for empty path, got %q", expected, dsn)
	}
}
