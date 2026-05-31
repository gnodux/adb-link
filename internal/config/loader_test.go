package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gnodux/adb-link/internal/models"
)

func writeYAML(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func newTestConfigService(t *testing.T, dir string) *ConfigService {
	t.Helper()
	cs := &ConfigService{configDir: dir}
	cs.snap.Store(newEmptySnapshot())
	if err := cs.Reload(); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	return cs
}

func TestNewConfigService_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	if got := len(cs.ListDatasources()); got != 0 {
		t.Errorf("expected 0 datasources, got %d", got)
	}
	if got := len(cs.AllTools()); got != 0 {
		t.Errorf("expected 0 tools, got %d", got)
	}
}

func TestReload_SingleDatasource(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "ds.yaml", `
kind: datasource
name: test-mysql
type: mysql
description: "test mysql"
connection:
  host: localhost
  port: 3306
  username: root
  password: pass
  default_database: mydb
`)
	cs := newTestConfigService(t, dir)
	dss := cs.ListDatasources()
	if len(dss) != 1 {
		t.Fatalf("expected 1 datasource, got %d", len(dss))
	}
	if dss[0].Name != "test-mysql" {
		t.Errorf("name = %q", dss[0].Name)
	}
	if dss[0].Type != models.DatabaseTypeMySQL {
		t.Errorf("type = %q", dss[0].Type)
	}
}

func TestReload_MultipleKinds(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "ds.yaml", `
kind: datasource
name: pg1
type: postgresql
connection:
  host: localhost
  port: 5432
`)
	writeYAML(t, dir, "tool.yaml", `
kind: tool
name: my-tool
description: "a tool"
datasource: pg1
template: "SELECT 1"
`)
	writeYAML(t, dir, "meta.yaml", `
kind: metadata
datasource: pg1
databases:
  mydb:
    comment: "main db"
`)
	cs := newTestConfigService(t, dir)
	if len(cs.ListDatasources()) != 1 {
		t.Errorf("datasources = %d", len(cs.ListDatasources()))
	}
	if len(cs.AllTools()) != 1 {
		t.Errorf("tools = %d", len(cs.AllTools()))
	}
	if len(cs.AllMetadata()) != 1 {
		t.Errorf("metadata = %d", len(cs.AllMetadata()))
	}
}

func TestReload_DisabledDatasource_Skipped(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "ds.yaml", `
kind: datasource
name: disabled-ds
type: mysql
enable: false
connection:
  host: localhost
`)
	cs := newTestConfigService(t, dir)
	if len(cs.ListDatasources()) != 0 {
		t.Errorf("disabled datasource should be skipped")
	}
}

func TestReload_DisabledTool_Skipped(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "tool.yaml", `
kind: tool
name: disabled-tool
description: "off"
enable: false
datasource: x
template: "SELECT 1"
`)
	cs := newTestConfigService(t, dir)
	if len(cs.AllTools()) != 0 {
		t.Errorf("disabled tool should be skipped")
	}
}

func TestReload_EnvVarInterpolation(t *testing.T) {
	t.Setenv("TEST_DB_HOST", "10.0.0.1")
	dir := t.TempDir()
	writeYAML(t, dir, "ds.yaml", `
kind: datasource
name: env-ds
type: mysql
connection:
  host: ${TEST_DB_HOST}
  port: 3306
`)
	cs := newTestConfigService(t, dir)
	cfg, err := cs.GetDatasource("env-ds")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Connection.Host != "10.0.0.1" {
		t.Errorf("host = %q, want %q", cfg.Connection.Host, "10.0.0.1")
	}
}

func TestReload_EnvVarUnset_KeepsPlaceholder(t *testing.T) {
	os.Unsetenv("NONEXISTENT_VAR_XYZ")
	dir := t.TempDir()
	writeYAML(t, dir, "ds.yaml", `
kind: datasource
name: placeholder-ds
type: mysql
connection:
  host: ${NONEXISTENT_VAR_XYZ}
`)
	cs := newTestConfigService(t, dir)
	cfg, err := cs.GetDatasource("placeholder-ds")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Connection.Host != "${NONEXISTENT_VAR_XYZ}" {
		t.Errorf("host = %q, want placeholder", cfg.Connection.Host)
	}
}

func TestReload_MultiDocumentYAML(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "multi.yaml", `
kind: datasource
name: ds-a
type: mysql
connection:
  host: a
---
kind: datasource
name: ds-b
type: postgresql
connection:
  host: b
`)
	cs := newTestConfigService(t, dir)
	if len(cs.ListDatasources()) != 2 {
		t.Errorf("expected 2 datasources from multi-doc, got %d", len(cs.ListDatasources()))
	}
}

func TestReload_UnknownKind_NoError(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "unknown.yaml", `
kind: something_else
name: foo
`)
	cs := newTestConfigService(t, dir)
	// Should not crash; unknown kind is silently skipped
	if len(cs.ListDatasources()) != 0 {
		t.Errorf("unexpected datasources")
	}
	_ = cs
}

func TestReload_DefaultKind_IsDatasource(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "no-kind.yaml", `
name: default-kind-ds
type: sqlite
connection:
  path: ./test.db
`)
	cs := newTestConfigService(t, dir)
	if len(cs.ListDatasources()) != 1 {
		t.Errorf("expected 1 datasource (default kind), got %d", len(cs.ListDatasources()))
	}
}

func TestReload_InvalidDir_ReturnsError(t *testing.T) {
	cs := &ConfigService{configDir: "/nonexistent/dir/xyz"}
	cs.snap.Store(newEmptySnapshot())
	if err := cs.Reload(); err == nil {
		t.Error("expected error for invalid dir")
	}
}

func TestRegisterTool_AppearsInAllTools(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	tool := &models.ToolConfig{Name: "dynamic-tool", Description: "d", Datasource: "x", Template: "SELECT 1"}
	cs.RegisterTool(tool)
	tools := cs.AllTools()
	if len(tools) != 1 || tools[0].Name != "dynamic-tool" {
		t.Errorf("expected dynamic-tool, got %v", tools)
	}
}

func TestRegisterTool_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	cs.RegisterTool(&models.ToolConfig{Name: "t1", Description: "v1", Datasource: "x", Template: "1"})
	cs.RegisterTool(&models.ToolConfig{Name: "t1", Description: "v2", Datasource: "x", Template: "2"})
	tool, err := cs.GetTool("t1")
	if err != nil {
		t.Fatal(err)
	}
	if tool.Description != "v2" {
		t.Errorf("description = %q, want v2", tool.Description)
	}
}

func TestUnregisterTool_RemovesTool(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	cs.RegisterTool(&models.ToolConfig{Name: "rm-me", Datasource: "x", Template: "1"})
	removed := cs.UnregisterTool("rm-me")
	if removed == nil || removed.Name != "rm-me" {
		t.Errorf("expected removed tool, got %v", removed)
	}
	if len(cs.AllTools()) != 0 {
		t.Error("tool should be removed")
	}
}

func TestUnregisterTool_NonExistent_ReturnsNil(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	if got := cs.UnregisterTool("ghost"); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestPersistTool_WritesYAMLFile(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	tool := &models.ToolConfig{Name: "persist-me", Description: "desc", Datasource: "x", Template: "SELECT 1"}
	path, err := cs.PersistTool(tool)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file should exist: %v", err)
	}
}

func TestPersistTool_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	tool := &models.ToolConfig{Kind: "tool", Name: "rt-tool", Description: "round trip", Datasource: "x", Template: "SELECT :id"}
	if _, err := cs.PersistTool(tool); err != nil {
		t.Fatal(err)
	}
	// Reload should pick up the persisted file
	if err := cs.Reload(); err != nil {
		t.Fatal(err)
	}
	got, err := cs.GetTool("rt-tool")
	if err != nil {
		t.Fatal(err)
	}
	if got.Description != "round trip" {
		t.Errorf("description = %q", got.Description)
	}
}

func TestRemoveToolFile_DeletesFile(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	cs.PersistTool(&models.ToolConfig{Name: "del-tool", Datasource: "x", Template: "1"})
	if !cs.RemoveToolFile("del-tool") {
		t.Error("expected true")
	}
	path := filepath.Join(dir, "tool-del-tool.yaml")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}
}

func TestRemoveToolFile_NonExistent_ReturnsFalse(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	if cs.RemoveToolFile("nope") {
		t.Error("expected false for non-existent file")
	}
}

func TestAddReloadCallback_CalledOnReload(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	called := false
	cs.AddReloadCallback(func() { called = true })
	if err := cs.Reload(); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("callback should have been called")
	}
}

func TestGetDatasource_NotFound(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	_, err := cs.GetDatasource("nonexistent")
	if err == nil {
		t.Error("expected error")
	}
}

func TestGetTool_NotFound(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	_, err := cs.GetTool("nonexistent")
	if err == nil {
		t.Error("expected error")
	}
}

func TestListDatasources_ReturnsDialectInfo(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "ds.yaml", `
kind: datasource
name: my-pg
type: postgresql
connection:
  host: localhost
`)
	cs := newTestConfigService(t, dir)
	dss := cs.ListDatasources()
	if len(dss) != 1 {
		t.Fatal("expected 1")
	}
	if dss[0].Dialect.SQLStyle != "postgresql" {
		t.Errorf("sql_style = %q", dss[0].Dialect.SQLStyle)
	}
}

func TestAllAuthUsers_KeyedByAPIKey(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "auth.yaml", `
kind: users
users:
  - name: alice
    api_key: key-alice
  - name: bob
    api_key: key-bob
`)
	cs := newTestConfigService(t, dir)
	users := cs.AllAuthUsers()
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users["key-alice"].Name != "alice" {
		t.Errorf("alice not found by key")
	}
}

func TestAllPermissions(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "perm.yaml", `
kind: permission
users:
  - alice
rules:
  - datasource: "*"
    databases: ["*"]
    tables: ["*"]
`)
	cs := newTestConfigService(t, dir)
	perms := cs.AllPermissions()
	if len(perms) != 1 {
		t.Fatalf("expected 1 permission, got %d", len(perms))
	}
	if len(perms[0].Rules) != 1 {
		t.Errorf("expected 1 rule")
	}
}

func TestAllMetadata(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "meta.yaml", `
kind: metadata
datasource: pg1
databases:
  mydb:
    comment: "main"
`)
	cs := newTestConfigService(t, dir)
	meta := cs.AllMetadata()
	if len(meta) != 1 {
		t.Fatalf("expected 1 metadata, got %d", len(meta))
	}
	if meta[0].Datasource != "pg1" {
		t.Errorf("datasource = %q", meta[0].Datasource)
	}
}

func TestRegisterDatasource_AppearsInList(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	cfg := &models.DatasourceConfig{
		Name: "dynamic-ds",
		Type: models.DatabaseTypeSQLite,
		Connection: models.ConnectionConfig{Path: "./test.db"},
	}
	cs.RegisterDatasource(cfg)
	dss := cs.ListDatasources()
	if len(dss) != 1 || dss[0].Name != "dynamic-ds" {
		t.Errorf("expected dynamic-ds, got %v", dss)
	}
}

func TestRegisterDatasource_GetDatasource(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	cfg := &models.DatasourceConfig{
		Name:        "dyn-pg",
		Type:        models.DatabaseTypePostgreSQL,
		Description: "dynamic pg",
		Connection:  models.ConnectionConfig{Host: "10.0.0.1", Port: 5432},
	}
	cs.RegisterDatasource(cfg)
	got, err := cs.GetDatasource("dyn-pg")
	if err != nil {
		t.Fatal(err)
	}
	if got.Description != "dynamic pg" {
		t.Errorf("description = %q", got.Description)
	}
	if got.Connection.Host != "10.0.0.1" {
		t.Errorf("host = %q", got.Connection.Host)
	}
}

func TestUnregisterDatasource_RemovesDatasource(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	cs.RegisterDatasource(&models.DatasourceConfig{Name: "rm-ds", Type: models.DatabaseTypeMySQL})
	removed := cs.UnregisterDatasource("rm-ds")
	if removed == nil || removed.Name != "rm-ds" {
		t.Errorf("expected removed datasource, got %v", removed)
	}
	if len(cs.ListDatasources()) != 0 {
		t.Error("datasource should be removed")
	}
}

func TestUnregisterDatasource_NonExistent_ReturnsNil(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	if got := cs.UnregisterDatasource("ghost"); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestPersistDatasource_WritesYAMLFile(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	cfg := &models.DatasourceConfig{
		Name: "persist-ds",
		Type: models.DatabaseTypeMySQL,
		Connection: models.ConnectionConfig{Host: "localhost", Port: 3306},
	}
	path, err := cs.PersistDatasource(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file should exist: %v", err)
	}
}

func TestPersistDatasource_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	cfg := &models.DatasourceConfig{
		Name:        "rt-ds",
		Type:        models.DatabaseTypePostgreSQL,
		Description: "round trip ds",
		Connection:  models.ConnectionConfig{Host: "db.example.com", Port: 5432, Username: "user1"},
	}
	if _, err := cs.PersistDatasource(cfg); err != nil {
		t.Fatal(err)
	}
	// Reload should pick up the persisted file
	if err := cs.Reload(); err != nil {
		t.Fatal(err)
	}
	got, err := cs.GetDatasource("rt-ds")
	if err != nil {
		t.Fatal(err)
	}
	if got.Description != "round trip ds" {
		t.Errorf("description = %q", got.Description)
	}
	if got.Connection.Host != "db.example.com" {
		t.Errorf("host = %q", got.Connection.Host)
	}
	if got.Connection.Username != "user1" {
		t.Errorf("username = %q", got.Connection.Username)
	}
}

func TestRemoveDatasourceFile_DeletesFile(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	cs.PersistDatasource(&models.DatasourceConfig{Name: "del-ds", Type: models.DatabaseTypeMySQL})
	if !cs.RemoveDatasourceFile("del-ds") {
		t.Error("expected true")
	}
	path := filepath.Join(dir, "datasource-del-ds.yaml")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}
}

func TestRemoveDatasourceFile_NonExistent_ReturnsFalse(t *testing.T) {
	dir := t.TempDir()
	cs := newTestConfigService(t, dir)
	if cs.RemoveDatasourceFile("nope") {
		t.Error("expected false for non-existent file")
	}
}
