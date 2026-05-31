package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gnodux/adb-link/internal/config"
	"github.com/gnodux/adb-link/internal/models"
	"github.com/gnodux/adb-link/internal/services"
	_ "modernc.org/sqlite"
)

func newTestContainer(t *testing.T) *services.Container {
	t.Helper()
	dir := t.TempDir()
	settings := &config.Settings{ConfigDir: dir, LogDir: dir, AsyncQueryTTL: 3600}
	return services.NewContainer(settings)
}

func TestRegisterDatasource_EmptyName(t *testing.T) {
	c := newTestContainer(t)
	h := NewHandlers(c)

	body := `{"name":"","type":"sqlite","connection":{"path":"./test.db"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/datasources/register", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	h.RegisterDatasource(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestRegisterDatasource_Conflict(t *testing.T) {
	c := newTestContainer(t)
	h := NewHandlers(c)

	// Pre-register a datasource in the snapshot.
	c.ConfigService.RegisterDatasource(&models.DatasourceConfig{
		Name: "existing-ds",
		Type: models.DatabaseTypeSQLite,
	})

	body := `{"name":"existing-ds","type":"sqlite","connection":{"path":"./test.db"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/datasources/register", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	h.RegisterDatasource(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestUnregisterDatasource_NotFound(t *testing.T) {
	c := newTestContainer(t)
	h := NewHandlers(c)

	body := `{"name":"ghost"}`
	req := httptest.NewRequest(http.MethodPost, "/api/datasources/unregister", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	h.UnregisterDatasource(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestRegisterDatasource_SQLite_OK(t *testing.T) {
	c := newTestContainer(t)
	h := NewHandlers(c)

	// Create a real SQLite file for connection validation.
	dir := c.ConfigService.ConfigDir()
	dbPath := filepath.Join(dir, "test.db")
	if err := os.WriteFile(dbPath, nil, 0644); err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{
		"name":        "dyn-sqlite",
		"type":        "sqlite",
		"description": "dynamic sqlite",
		"connection":  map[string]any{"path": dbPath},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/datasources/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.RegisterDatasource(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp models.APIResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.Error)
	}

	// Verify datasource is in snapshot.
	cfg, err := c.ConfigService.GetDatasource("dyn-sqlite")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Description != "dynamic sqlite" {
		t.Errorf("description = %q", cfg.Description)
	}

	// Verify YAML file was persisted.
	yamlPath := filepath.Join(dir, "datasource-dyn-sqlite.yaml")
	if _, err := os.Stat(yamlPath); err != nil {
		t.Errorf("persisted file should exist: %v", err)
	}
}

func TestUnregisterDatasource_OK(t *testing.T) {
	c := newTestContainer(t)
	h := NewHandlers(c)

	// Pre-register a datasource.
	cfg := &models.DatasourceConfig{
		Name: "rm-ds",
		Type: models.DatabaseTypeSQLite,
	}
	c.ConfigService.RegisterDatasource(cfg)
	c.ConfigService.PersistDatasource(cfg)

	body := `{"name":"rm-ds"}`
	req := httptest.NewRequest(http.MethodPost, "/api/datasources/unregister", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	h.UnregisterDatasource(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify datasource is removed from snapshot.
	if _, err := c.ConfigService.GetDatasource("rm-ds"); err == nil {
		t.Error("datasource should be removed")
	}

	// Verify YAML file was deleted.
	yamlPath := filepath.Join(c.ConfigService.ConfigDir(), "datasource-rm-ds.yaml")
	if _, err := os.Stat(yamlPath); !os.IsNotExist(err) {
		t.Error("persisted file should be deleted")
	}
}

func TestRegisterDatasource_InvalidConnection(t *testing.T) {
	c := newTestContainer(t)
	h := NewHandlers(c)

	payload := map[string]any{
		"name":       "bad-sqlite",
		"type":       "sqlite",
		"connection": map[string]any{"path": "/nonexistent/path/to/db.sqlite"},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/datasources/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.RegisterDatasource(rec, req)

	// SQLite will create the file on open, so we test with a type that
	// definitely can't connect — mysql to a non-routable address.
	payload = map[string]any{
		"name":       "bad-mysql",
		"type":       "mysql",
		"connection": map[string]any{"host": "192.0.2.1", "port": 1, "username": "x"},
	}
	body, _ = json.Marshal(payload)
	req = httptest.NewRequest(http.MethodPost, "/api/datasources/register", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	h.RegisterDatasource(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid connection, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify rollback: datasource should NOT be in snapshot.
	if _, err := c.ConfigService.GetDatasource("bad-mysql"); err == nil {
		t.Error("datasource should be rolled back after validation failure")
	}
}
