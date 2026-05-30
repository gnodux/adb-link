package integration_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/gnodux/adb-link/internal/config"
	"github.com/gnodux/adb-link/internal/dialects"
	"github.com/gnodux/adb-link/internal/models"
	"github.com/gnodux/adb-link/internal/services"
	_ "modernc.org/sqlite"
)

func setupSQLite(t *testing.T) (*sql.DB, *config.ConfigService, string) {
	t.Helper()

	// Use file-based SQLite so all connections share the same DB
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	seedSQLite(t, db)

	writeYAML(t, dir, "ds.yaml", `
kind: datasource
name: test-sqlite
type: sqlite
connection:
  path: `+dbPath+`
`)
	cs := config.NewConfigService(&config.Settings{ConfigDir: dir})
	return db, cs, dir
}

func seedSQLite(t *testing.T, db *sql.DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT, created_at TEXT)`,
		`CREATE TABLE orders (id INTEGER PRIMARY KEY, user_id INTEGER, amount REAL, status TEXT)`,
		`CREATE VIEW user_order_summary AS SELECT u.name, COUNT(o.id) as order_count, COALESCE(SUM(o.amount),0) as total FROM users u LEFT JOIN orders o ON u.id = o.user_id GROUP BY u.name`,
		`INSERT INTO users VALUES (1, 'alice', 'alice@test.com', '2024-01-01')`,
		`INSERT INTO users VALUES (2, 'bob', 'bob@test.com', '2024-01-02')`,
		`INSERT INTO users VALUES (3, 'carol', 'carol@test.com', '2024-01-03')`,
		`INSERT INTO orders VALUES (1, 1, 100.50, 'completed')`,
		`INSERT INTO orders VALUES (2, 1, 200.00, 'completed')`,
		`INSERT INTO orders VALUES (3, 2, 50.25, 'pending')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed: %v\nSQL: %s", err, s)
		}
	}
}

func writeYAML(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// --- Dialect tests ---

func TestSQLite_Dialect_GetDatabases(t *testing.T) {
	db, _, _ := setupSQLite(t)
	d := &dialects.SQLiteDialect{}
	dbs, err := d.GetDatabases(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if len(dbs) != 1 || dbs[0].Name != "main" {
		t.Errorf("expected [main], got %v", dbs)
	}
}

func TestSQLite_Dialect_GetTableNames(t *testing.T) {
	db, _, _ := setupSQLite(t)
	d := &dialects.SQLiteDialect{}
	tables, err := d.GetTableNames(context.Background(), db, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 2 {
		t.Fatalf("expected 2 tables, got %d: %v", len(tables), tables)
	}
	names := map[string]bool{}
	for _, tbl := range tables {
		names[tbl.Name] = true
	}
	if !names["users"] || !names["orders"] {
		t.Errorf("expected users and orders, got %v", names)
	}
}

func TestSQLite_Dialect_GetViewNames(t *testing.T) {
	db, _, _ := setupSQLite(t)
	d := &dialects.SQLiteDialect{}
	views, err := d.GetViewNames(context.Background(), db, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 1 || views[0].Name != "user_order_summary" {
		t.Errorf("expected [user_order_summary], got %v", views)
	}
}

func TestSQLite_Dialect_GetTableInfo(t *testing.T) {
	db, _, _ := setupSQLite(t)
	d := &dialects.SQLiteDialect{}
	info, err := d.GetTableInfo(context.Background(), db, "main", "users")
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "users" {
		t.Errorf("name = %q", info.Name)
	}
	if len(info.Columns) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(info.Columns))
	}
	foundPK := false
	for _, col := range info.Columns {
		if col.Name == "id" && col.IsPrimaryKey {
			foundPK = true
		}
	}
	if !foundPK {
		t.Error("id should be primary key")
	}
}

func TestSQLite_Dialect_GetViewInfo(t *testing.T) {
	db, _, _ := setupSQLite(t)
	d := &dialects.SQLiteDialect{}
	info, err := d.GetViewInfo(context.Background(), db, "main", "user_order_summary")
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "user_order_summary" {
		t.Errorf("name = %q", info.Name)
	}
	if len(info.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(info.Columns))
	}
}

// --- Service tests ---

func newSQLiteServices(t *testing.T) (*services.QueryService, *services.SchemaService, *config.ConfigService) {
	t.Helper()
	_, cs, _ := setupSQLite(t)
	conn := services.NewConnectionService(cs)
	t.Cleanup(func() { conn.DisposeAll() })
	// nil permission service = bypass all checks
	qs := services.NewQueryService(conn, cs, nil)
	ss := services.NewSchemaService(cs, conn, services.NewMetadataService(nil), nil)
	return qs, ss, cs
}

func TestSQLite_QueryService_Execute_Select(t *testing.T) {
	qs, _, _ := newSQLiteServices(t)
	result, err := qs.Execute(context.Background(), &models.QueryRequest{
		DatasourceName: "test-sqlite",
		SQL:            "SELECT id, name, email FROM users ORDER BY id",
		Limit:          100,
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.RowCount != 3 {
		t.Errorf("expected 3 rows, got %d", result.RowCount)
	}
	if len(result.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(result.Columns))
	}
}

func TestSQLite_QueryService_Execute_CreateInsertSelect(t *testing.T) {
	qs, _, _ := newSQLiteServices(t)
	_, err := qs.Execute(context.Background(), &models.QueryRequest{
		DatasourceName: "test-sqlite",
		SQL:            "CREATE TABLE test_tmp (id INTEGER PRIMARY KEY, val TEXT)",
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = qs.Execute(context.Background(), &models.QueryRequest{
		DatasourceName: "test-sqlite",
		SQL:            "INSERT INTO test_tmp VALUES (1, 'hello')",
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	result, err := qs.Execute(context.Background(), &models.QueryRequest{
		DatasourceName: "test-sqlite",
		SQL:            "SELECT * FROM test_tmp",
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.RowCount != 1 {
		t.Errorf("expected 1 row, got %d", result.RowCount)
	}
}

func TestSQLite_QueryService_Explain(t *testing.T) {
	qs, _, _ := newSQLiteServices(t)
	result, err := qs.Explain(context.Background(), &models.ExplainRequest{
		DatasourceName: "test-sqlite",
		SQL:            "SELECT * FROM users WHERE id = 1",
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.DatabaseType != "sqlite" {
		t.Errorf("database_type = %q", result.DatabaseType)
	}
	if len(result.Rows) == 0 {
		t.Error("expected explain output rows")
	}
}

func TestSQLite_QueryService_Limit(t *testing.T) {
	qs, _, _ := newSQLiteServices(t)
	result, err := qs.Execute(context.Background(), &models.QueryRequest{
		DatasourceName: "test-sqlite",
		SQL:            "SELECT * FROM users",
		Limit:          2,
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.RowCount != 2 {
		t.Errorf("expected 2 rows (limit), got %d", result.RowCount)
	}
	if !result.Truncated {
		t.Error("expected truncated=true")
	}
}

func TestSQLite_SchemaService_GetDatabases(t *testing.T) {
	_, ss, _ := newSQLiteServices(t)
	dbs, err := ss.GetDatabases(context.Background(), "test-sqlite", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(dbs) != 1 || dbs[0].Name != "main" {
		t.Errorf("expected [main], got %v", dbs)
	}
}

func TestSQLite_SchemaService_GetSchema(t *testing.T) {
	_, ss, _ := newSQLiteServices(t)
	schema, err := ss.GetSchema(context.Background(), "test-sqlite", "main", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(schema.Tables) < 2 {
		t.Errorf("expected at least 2 tables, got %d", len(schema.Tables))
	}
	if len(schema.Views) < 1 {
		t.Errorf("expected at least 1 view, got %d", len(schema.Views))
	}
}

func TestSQLite_SchemaService_GetTableInfo(t *testing.T) {
	_, ss, _ := newSQLiteServices(t)
	info, err := ss.GetTableInfo(context.Background(), "test-sqlite", "main", "users", "")
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "users" {
		t.Errorf("name = %q", info.Name)
	}
	if len(info.Columns) != 4 {
		t.Errorf("expected 4 columns, got %d", len(info.Columns))
	}
}
