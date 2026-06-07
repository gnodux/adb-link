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

func setupClickHouse(t *testing.T) (*sql.DB, *services.QueryService, *services.SchemaService, *config.ConfigService) {
	t.Helper()

	// Set explicit password for ClickHouse container
	container := testutil.StartContainer(t, "docker.io/clickhouse/clickhouse-server:latest", 9000,
		[]string{
			"CLICKHOUSE_USER=default",
			"CLICKHOUSE_PASSWORD=testpass",
		}, nil)

	dsn := fmt.Sprintf("clickhouse://default:testpass@localhost:%s/default?dial_timeout=5s&read_timeout=60s", container.HostPort)
	testutil.WaitForSQL(t, "clickhouse", dsn, 90*time.Second)

	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	seedClickHouse(t, db)

	dir := t.TempDir()
	writeYAML(t, dir, "ds.yaml", fmt.Sprintf(`
kind: datasource
name: test-clickhouse
type: clickhouse
connection:
  host: localhost
  port: %s
  username: default
  password: testpass
  default_database: default
`, container.HostPort))

	cs := config.NewConfigService(&config.Settings{ConfigDir: dir})
	conn := services.NewConnectionService(cs)
	t.Cleanup(func() { conn.DisposeAll() })

	qs := services.NewQueryService(conn, cs, nil)
	ps := services.NewPermissionService(nil, nil)
	ss := services.NewSchemaService(cs, conn, services.NewMetadataService(nil), ps)

	return db, qs, ss, cs
}

func seedClickHouse(t *testing.T, db *sql.DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE users (id UInt32, name String, email String, created_at DateTime) ENGINE = MergeTree() ORDER BY id`,
		`CREATE TABLE orders (id UInt32, user_id UInt32, amount Float64, status String) ENGINE = MergeTree() ORDER BY id`,
		`INSERT INTO users VALUES (1, 'alice', 'alice@test.com', '2024-01-01 00:00:00')`,
		`INSERT INTO users VALUES (2, 'bob', 'bob@test.com', '2024-01-02 00:00:00')`,
		`INSERT INTO users VALUES (3, 'carol', 'carol@test.com', '2024-01-03 00:00:00')`,
		`INSERT INTO orders VALUES (1, 1, 100.50, 'completed')`,
		`INSERT INTO orders VALUES (2, 1, 200.00, 'completed')`,
		`INSERT INTO orders VALUES (3, 2, 50.25, 'pending')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed clickhouse: %v\nSQL: %s", err, s)
		}
	}
}

// --- Dialect tests ---

func TestCH_Dialect_GetDatabases(t *testing.T) {
	db, _, _, _ := setupClickHouse(t)
	dial := &dialects.ClickHouseDialect{}
	dbs, err := dial.GetDatabases(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range dbs {
		if d.Name == "default" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'default' in databases, got %v", dbs)
	}
}

func TestCH_Dialect_GetTableNames(t *testing.T) {
	db, _, _, _ := setupClickHouse(t)
	dial := &dialects.ClickHouseDialect{}
	tables, err := dial.GetTableNames(context.Background(), db, "default")
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

func TestCH_Dialect_GetTableInfo(t *testing.T) {
	db, _, _, _ := setupClickHouse(t)
	dial := &dialects.ClickHouseDialect{}
	info, err := dial.GetTableInfo(context.Background(), db, "default", "users")
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "users" {
		t.Errorf("name = %q", info.Name)
	}
	if len(info.Columns) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(info.Columns))
	}
}

// --- Service tests ---

func TestCH_QueryService_Execute_Select(t *testing.T) {
	_, qs, _, _ := setupClickHouse(t)
	result, err := qs.Execute(context.Background(), &models.QueryRequest{
		DatasourceName: "test-clickhouse",
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

func TestCH_QueryService_Explain(t *testing.T) {
	_, qs, _, _ := setupClickHouse(t)
	result, err := qs.Explain(context.Background(), &models.ExplainRequest{
		DatasourceName: "test-clickhouse",
		SQL:            "SELECT * FROM users WHERE id = 1",
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.DatabaseType != "clickhouse" {
		t.Errorf("database_type = %q", result.DatabaseType)
	}
	if len(result.Rows) == 0 {
		t.Error("expected explain output rows")
	}
}
