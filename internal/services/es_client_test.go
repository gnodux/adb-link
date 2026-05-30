package services

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/gnodux/adb-link/internal/models"
)

// ---------------------------------------------------------------------------
// Tests for collectFields (pure logic, no HTTP)
// ---------------------------------------------------------------------------

func TestCollectFields_FlatProperties(t *testing.T) {
	props := map[string]any{
		"name": map[string]any{"type": "text"},
		"age":  map[string]any{"type": "integer"},
	}

	var columns []models.ColumnInfo
	collectFields(&columns, "", props)

	if len(columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(columns))
	}

	// Sort for deterministic comparison (map iteration order is random).
	sort.Slice(columns, func(i, j int) bool { return columns[i].Name < columns[j].Name })

	if columns[0].Name != "age" || columns[0].Type != "integer" {
		t.Errorf("unexpected column[0]: %+v", columns[0])
	}
	if columns[1].Name != "name" || columns[1].Type != "text" {
		t.Errorf("unexpected column[1]: %+v", columns[1])
	}
	for _, c := range columns {
		if !c.Nullable {
			t.Errorf("expected column %q to be nullable", c.Name)
		}
	}
}

func TestCollectFields_NestedProperties(t *testing.T) {
	props := map[string]any{
		"address": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"city": map[string]any{"type": "text"},
				"zip":  map[string]any{"type": "keyword"},
			},
		},
	}

	var columns []models.ColumnInfo
	collectFields(&columns, "", props)

	if len(columns) != 3 {
		t.Fatalf("expected 3 columns (parent + 2 nested), got %d", len(columns))
	}

	sort.Slice(columns, func(i, j int) bool { return columns[i].Name < columns[j].Name })

	expected := []struct{ name, typ string }{
		{"address", "object"},
		{"address.city", "text"},
		{"address.zip", "keyword"},
	}
	for i, e := range expected {
		if columns[i].Name != e.name || columns[i].Type != e.typ {
			t.Errorf("column[%d]: expected (%s, %s), got (%s, %s)",
				i, e.name, e.typ, columns[i].Name, columns[i].Type)
		}
	}
}

func TestCollectFields_DeeplyNested(t *testing.T) {
	props := map[string]any{
		"a": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"b": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"c": map[string]any{"type": "keyword"},
					},
				},
			},
		},
	}

	var columns []models.ColumnInfo
	collectFields(&columns, "", props)

	sort.Slice(columns, func(i, j int) bool { return columns[i].Name < columns[j].Name })

	expected := []struct{ name, typ string }{
		{"a", "object"},
		{"a.b", "object"},
		{"a.b.c", "keyword"},
	}
	if len(columns) != len(expected) {
		t.Fatalf("expected %d columns, got %d: %+v", len(expected), len(columns), columns)
	}
	for i, e := range expected {
		if columns[i].Name != e.name || columns[i].Type != e.typ {
			t.Errorf("column[%d]: expected (%s, %s), got (%s, %s)",
				i, e.name, e.typ, columns[i].Name, columns[i].Type)
		}
	}
}

func TestCollectFields_EmptyProperties(t *testing.T) {
	props := map[string]any{}

	var columns []models.ColumnInfo
	collectFields(&columns, "", props)

	if len(columns) != 0 {
		t.Fatalf("expected 0 columns for empty properties, got %d", len(columns))
	}
}

func TestCollectFields_NoType_SkipButRecurse(t *testing.T) {
	// A field without "type" but with "properties" should NOT produce a column
	// for itself, but should recurse into sub-properties.
	props := map[string]any{
		"group": map[string]any{
			"properties": map[string]any{
				"sub": map[string]any{"type": "text"},
			},
		},
	}

	var columns []models.ColumnInfo
	collectFields(&columns, "", props)

	if len(columns) != 1 {
		t.Fatalf("expected 1 column (only the leaf), got %d: %+v", len(columns), columns)
	}
	if columns[0].Name != "group.sub" || columns[0].Type != "text" {
		t.Errorf("unexpected column: %+v", columns[0])
	}
}

// ---------------------------------------------------------------------------
// Helper: create an ESClient backed by an httptest.Server
// ---------------------------------------------------------------------------

func newTestESClient(t *testing.T, handler http.HandlerFunc) (*ESClient, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("failed to parse test server URL: %v", err)
	}
	host, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatalf("failed to split host/port: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}

	cfg := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{Host: host, Port: port},
	}
	return NewESClient(cfg), ts
}

// jsonResponse writes a JSON body with the given status code.
func jsonResponse(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// ---------------------------------------------------------------------------
// Tests for ESClient methods via httptest.Server
// ---------------------------------------------------------------------------

func TestESClient_Info(t *testing.T) {
	client, _ := newTestESClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Errorf("expected path /, got %s", r.URL.Path)
		}
		jsonResponse(w, http.StatusOK, map[string]any{
			"cluster_name": "test-cluster",
			"version":      map[string]any{"number": "8.0.0"},
		})
	})

	info, err := client.Info(context.Background())
	if err != nil {
		t.Fatalf("Info() error: %v", err)
	}
	if name, _ := info["cluster_name"].(string); name != "test-cluster" {
		t.Errorf("expected cluster_name=test-cluster, got %q", name)
	}
}

func TestESClient_GetDatabases(t *testing.T) {
	client, _ := newTestESClient(t, func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, http.StatusOK, map[string]any{
			"cluster_name": "my-cluster",
		})
	})

	dbs, err := client.GetDatabases(context.Background())
	if err != nil {
		t.Fatalf("GetDatabases() error: %v", err)
	}
	if len(dbs) != 1 {
		t.Fatalf("expected 1 database, got %d", len(dbs))
	}
	if dbs[0].Name != "my-cluster" {
		t.Errorf("expected database name my-cluster, got %q", dbs[0].Name)
	}
}

func TestESClient_GetTableNames(t *testing.T) {
	client, _ := newTestESClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/_all/_alias") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		jsonResponse(w, http.StatusOK, map[string]any{
			"idx-a":   map[string]any{},
			"idx-b":   map[string]any{},
			".system": map[string]any{},
		})
	})

	names, err := client.GetTableNames(context.Background(), "")
	if err != nil {
		t.Fatalf("GetTableNames() error: %v", err)
	}

	// .system should be filtered out
	if len(names) != 2 {
		t.Fatalf("expected 2 table names, got %d: %+v", len(names), names)
	}

	// Results are sorted by the implementation.
	if names[0].Name != "idx-a" || names[1].Name != "idx-b" {
		t.Errorf("unexpected names: %q, %q", names[0].Name, names[1].Name)
	}
}

func TestESClient_GetTableInfo(t *testing.T) {
	client, _ := newTestESClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/my-index/_mapping") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		jsonResponse(w, http.StatusOK, map[string]any{
			"my-index": map[string]any{
				"mappings": map[string]any{
					"properties": map[string]any{
						"title": map[string]any{"type": "text"},
						"count": map[string]any{"type": "integer"},
					},
				},
			},
		})
	})

	info, err := client.GetTableInfo(context.Background(), "", "my-index")
	if err != nil {
		t.Fatalf("GetTableInfo() error: %v", err)
	}
	if info.Name != "my-index" {
		t.Errorf("expected table name my-index, got %q", info.Name)
	}
	if len(info.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(info.Columns))
	}

	sort.Slice(info.Columns, func(i, j int) bool { return info.Columns[i].Name < info.Columns[j].Name })
	if info.Columns[0].Name != "count" || info.Columns[0].Type != "integer" {
		t.Errorf("unexpected column[0]: %+v", info.Columns[0])
	}
	if info.Columns[1].Name != "title" || info.Columns[1].Type != "text" {
		t.Errorf("unexpected column[1]: %+v", info.Columns[1])
	}
}

func TestESClient_Search(t *testing.T) {
	client, _ := newTestESClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/logs/_search") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("size") != "10" {
			t.Errorf("expected size=10, got %s", r.URL.Query().Get("size"))
		}
		jsonResponse(w, http.StatusOK, map[string]any{
			"hits": map[string]any{
				"total": map[string]any{"value": 1},
				"hits": []any{
					map[string]any{"_id": "1", "_source": map[string]any{"msg": "hello"}},
				},
			},
		})
	})

	query := map[string]any{
		"query": map[string]any{"match_all": map[string]any{}},
	}
	result, err := client.Search(context.Background(), "logs", query, 10)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	hits, ok := result["hits"].(map[string]any)
	if !ok {
		t.Fatalf("expected hits to be a map, got %T", result["hits"])
	}
	total, ok := hits["total"].(map[string]any)
	if !ok {
		t.Fatalf("expected total to be a map, got %T", hits["total"])
	}
	if v, _ := total["value"].(float64); v != 1 {
		t.Errorf("expected total.value=1, got %v", total["value"])
	}
}

func TestESClient_BasicAuth(t *testing.T) {
	client, _ := newTestESClient(t, func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Error("expected Authorization header, got none")
		} else if !strings.HasPrefix(auth, "Basic ") {
			t.Errorf("expected Basic auth, got %q", auth)
		}
		jsonResponse(w, http.StatusOK, map[string]any{"cluster_name": "auth-cluster"})
	})

	// Set credentials directly on the client (the test config has none).
	client.username = "admin"
	client.password = "secret"

	_, err := client.Info(context.Background())
	if err != nil {
		t.Fatalf("Info() error: %v", err)
	}
}

func TestESClient_ErrorHandling_4xx(t *testing.T) {
	client, _ := newTestESClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal server error"}`))
	})

	_, err := client.Info(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention status 500, got: %v", err)
	}
}

func TestESClient_ErrorHandling_InvalidJSON(t *testing.T) {
	client, _ := newTestESClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("this is not json"))
	})

	_, err := client.Info(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON response, got nil")
	}
}
