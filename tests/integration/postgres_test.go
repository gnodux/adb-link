//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/gnodux/adb-link/internal/config"
	"github.com/gnodux/adb-link/internal/dialects"
	"github.com/gnodux/adb-link/internal/models"
	"github.com/gnodux/adb-link/internal/services"
	"github.com/gnodux/adb-link/tests/testutil"
)

func setupPostgres(t *testing.T) (*sql.DB, *services.QueryService, *services.SchemaService, *config.ConfigService) {
	t.Helper()

	container := testutil.StartContainer(t, "postgres:16-alpine", 5432,
		[]string{
			"POSTGRES_USER=test",
			"POSTGRES_PASSWORD=test",
			"POSTGRES_DB=testdb",
		}, nil)

	dsn := fmt.Sprintf("postgres://test:test@localhost:%s/testdb?sslmode=disable", container.HostPort)
	testutil.WaitForSQL(t, "postgres", dsn, 90*time.Second)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	seedPostgres(t, db)

	dir := t.TempDir()
	writeYAML(t, dir, "ds.yaml", fmt.Sprintf(`
kind: datasource
name: test-postgres
type: postgresql
connection:
  host: localhost
  port: %s
  username: test
  password: test
  default_database: testdb
`, container.HostPort))

	cs := config.NewConfigService(&config.Settings{ConfigDir: dir})
	conn := services.NewConnectionService(cs)
	t.Cleanup(func() { conn.DisposeAll() })

	qs := services.NewQueryService(conn, cs, nil)
	ps := services.NewPermissionService(nil, nil)
	ss := services.NewSchemaService(cs, conn, services.NewMetadataService(nil), ps)

	return db, qs, ss, cs
}

func seedPostgres(t *testing.T, db *sql.DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR NOT NULL, email VARCHAR, created_at TIMESTAMP DEFAULT NOW())`,
		`CREATE TABLE orders (id SERIAL PRIMARY KEY, user_id INTEGER REFERENCES users(id), amount NUMERIC, status VARCHAR)`,
		`CREATE VIEW user_order_summary AS SELECT u.name, COUNT(o.id) as order_count, COALESCE(SUM(o.amount),0) as total FROM users u LEFT JOIN orders o ON u.id = o.user_id GROUP BY u.name`,
		`INSERT INTO users (name, email) VALUES ('alice', 'alice@test.com')`,
		`INSERT INTO users (name, email) VALUES ('bob', 'bob@test.com')`,
		`INSERT INTO users (name, email) VALUES ('carol', 'carol@test.com')`,
		`INSERT INTO orders (user_id, amount, status) VALUES (1, 100.50, 'completed')`,
		`INSERT INTO orders (user_id, amount, status) VALUES (1, 200.00, 'completed')`,
		`INSERT INTO orders (user_id, amount, status) VALUES (2, 50.25, 'pending')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed postgres: %v\nSQL: %s", err, s)
		}
	}
}

// --- Dialect tests ---

func TestPG_Dialect_GetDatabases(t *testing.T) {
	db, _, _, _ := setupPostgres(t)
	dial := &dialects.PostgreSQLDialect{}
	dbs, err := dial.GetDatabases(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range dbs {
		if d.Name == "testdb" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected testdb in databases, got %v", dbs)
	}
}

func TestPG_Dialect_GetTableNames(t *testing.T) {
	db, _, _, _ := setupPostgres(t)
	dial := &dialects.PostgreSQLDialect{}
	tables, err := dial.GetTableNames(context.Background(), db, "testdb")
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

func TestPG_Dialect_GetViewNames(t *testing.T) {
	db, _, _, _ := setupPostgres(t)
	dial := &dialects.PostgreSQLDialect{}
	views, err := dial.GetViewNames(context.Background(), db, "testdb")
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 1 || views[0].Name != "user_order_summary" {
		t.Errorf("expected [user_order_summary], got %v", views)
	}
}

func TestPG_Dialect_GetTableInfo(t *testing.T) {
	db, _, _, _ := setupPostgres(t)
	dial := &dialects.PostgreSQLDialect{}
	info, err := dial.GetTableInfo(context.Background(), db, "testdb", "users")
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

func TestPG_Dialect_GetViewInfo(t *testing.T) {
	db, _, _, _ := setupPostgres(t)
	dial := &dialects.PostgreSQLDialect{}
	info, err := dial.GetViewInfo(context.Background(), db, "testdb", "user_order_summary")
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

func TestPG_QueryService_Execute_Select(t *testing.T) {
	_, qs, _, _ := setupPostgres(t)
	result, err := qs.Execute(context.Background(), &models.QueryRequest{
		DatasourceName: "test-postgres",
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

func TestPG_QueryService_Explain(t *testing.T) {
	_, qs, _, _ := setupPostgres(t)
	result, err := qs.Explain(context.Background(), &models.ExplainRequest{
		DatasourceName: "test-postgres",
		SQL:            "SELECT * FROM users WHERE id = 1",
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.DatabaseType != "postgresql" {
		t.Errorf("database_type = %q", result.DatabaseType)
	}
	if len(result.Rows) == 0 {
		t.Error("expected explain output rows")
	}
}

func TestPG_SchemaService_GetDatabases(t *testing.T) {
	_, _, ss, _ := setupPostgres(t)
	dbs, err := ss.GetDatabases(context.Background(), "test-postgres", "")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range dbs {
		if d.Name == "testdb" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected testdb in databases, got %v", dbs)
	}
}

func TestPG_SchemaService_GetSchema(t *testing.T) {
	_, _, ss, _ := setupPostgres(t)
	schema, err := ss.GetSchema(context.Background(), "test-postgres", "testdb", "")
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
