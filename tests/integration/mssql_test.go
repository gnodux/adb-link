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

type mssqlSetup struct {
	masterDB *sql.DB
	testDB   *sql.DB
	qs       *services.QueryService
	ss       *services.SchemaService
	cs       *config.ConfigService
}

func setupMSSQL(t *testing.T) *mssqlSetup {
	t.Helper()

	container := testutil.StartContainer(t, "mcr.microsoft.com/mssql/server:2022-latest", 1433,
		[]string{
			"ACCEPT_EULA=Y",
			"MSSQL_SA_PASSWORD=YourStrong!Passw0rd",
		}, nil)

	masterDSN := fmt.Sprintf("sqlserver://sa:YourStrong!Passw0rd@localhost:%s?database=master&trustservercertificate=true&dial+timeout=5&connection+timeout=60", container.HostPort)
	testutil.WaitForSQL(t, "sqlserver", masterDSN, 120*time.Second)

	masterDB, err := sql.Open("sqlserver", masterDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { masterDB.Close() })

	// Open a separate connection to testdb for dialect tests
	testdbDSN := fmt.Sprintf("sqlserver://sa:YourStrong!Passw0rd@localhost:%s?database=testdb&trustservercertificate=true&dial+timeout=5&connection+timeout=60", container.HostPort)

	seedMSSQL(t, masterDB, testdbDSN)

	testDB, err := sql.Open("sqlserver", testdbDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testDB.Close() })

	dir := t.TempDir()
	writeYAML(t, dir, "ds.yaml", fmt.Sprintf(`
kind: datasource
name: test-mssql
type: mssql
connection:
  host: localhost
  port: %s
  username: sa
  password: "YourStrong!Passw0rd"
  default_database: testdb
`, container.HostPort))

	cs := config.NewConfigService(&config.Settings{ConfigDir: dir})
	conn := services.NewConnectionService(cs)
	t.Cleanup(func() { conn.DisposeAll() })

	qs := services.NewQueryService(conn, cs, nil)
	ps := services.NewPermissionService(nil, nil)
	ss := services.NewSchemaService(cs, conn, services.NewMetadataService(nil), ps)

	return &mssqlSetup{
		masterDB: masterDB,
		testDB:   testDB,
		qs:       qs,
		ss:       ss,
		cs:       cs,
	}
}

func seedMSSQL(t *testing.T, masterDB *sql.DB, testdbDSN string) {
	t.Helper()

	// Create testdb
	if _, err := masterDB.Exec(`IF NOT EXISTS (SELECT * FROM sys.databases WHERE name = 'testdb') CREATE DATABASE testdb`); err != nil {
		t.Fatalf("seed mssql create database: %v", err)
	}

	// Open a dedicated connection to testdb for table creation
	testDB, err := sql.Open("sqlserver", testdbDSN)
	if err != nil {
		t.Fatalf("seed mssql open testdb: %v", err)
	}
	defer testDB.Close()

	stmts := []string{
		`CREATE TABLE users (id INT IDENTITY PRIMARY KEY, name NVARCHAR(100) NOT NULL, email NVARCHAR(200), created_at DATETIME2 DEFAULT GETDATE())`,
		`CREATE TABLE orders (id INT IDENTITY PRIMARY KEY, user_id INTEGER, amount DECIMAL(10,2), status NVARCHAR(50))`,
		`INSERT INTO users (name, email) VALUES ('alice', 'alice@test.com')`,
		`INSERT INTO users (name, email) VALUES ('bob', 'bob@test.com')`,
		`INSERT INTO users (name, email) VALUES ('carol', 'carol@test.com')`,
		`INSERT INTO orders (user_id, amount, status) VALUES (1, 100.50, 'completed')`,
		`INSERT INTO orders (user_id, amount, status) VALUES (1, 200.00, 'completed')`,
		`INSERT INTO orders (user_id, amount, status) VALUES (2, 50.25, 'pending')`,
	}
	for _, s := range stmts {
		if _, err := testDB.Exec(s); err != nil {
			t.Fatalf("seed mssql: %v\nSQL: %s", err, s)
		}
	}
}

// --- Dialect tests ---

func TestMSSQL_Dialect_GetDatabases(t *testing.T) {
	s := setupMSSQL(t)
	dial := &dialects.MSSQLDialect{}
	dbs, err := dial.GetDatabases(context.Background(), s.masterDB)
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

func TestMSSQL_Dialect_GetTableNames(t *testing.T) {
	s := setupMSSQL(t)
	dial := &dialects.MSSQLDialect{}
	tables, err := dial.GetTableNames(context.Background(), s.testDB, "testdb")
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

func TestMSSQL_Dialect_GetTableInfo(t *testing.T) {
	s := setupMSSQL(t)
	dial := &dialects.MSSQLDialect{}
	info, err := dial.GetTableInfo(context.Background(), s.testDB, "testdb", "users")
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

// --- Service tests ---

func TestMSSQL_QueryService_Execute_Select(t *testing.T) {
	s := setupMSSQL(t)
	result, err := s.qs.Execute(context.Background(), &models.QueryRequest{
		DatasourceName: "test-mssql",
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

func TestMSSQL_QueryService_Explain_SHOWPLAN(t *testing.T) {
	s := setupMSSQL(t)
	result, err := s.qs.Explain(context.Background(), &models.ExplainRequest{
		DatasourceName: "test-mssql",
		SQL:            "SELECT * FROM users WHERE id = 1",
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.DatabaseType != "mssql" {
		t.Errorf("database_type = %q", result.DatabaseType)
	}
	if len(result.Rows) == 0 {
		t.Error("expected SHOWPLAN_XML output rows")
	}
}
