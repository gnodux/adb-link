//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gnodux/adb-link/internal/models"
	"github.com/gnodux/adb-link/internal/services"
	"github.com/gnodux/adb-link/tests/testutil"
)

func setupElasticsearch(t *testing.T) (*services.ESClient, string) {
	t.Helper()

	container := testutil.StartContainer(t, "docker.elastic.co/elasticsearch/elasticsearch:8.14.0", 9200,
		[]string{
			"discovery.type=single-node",
			"xpack.security.enabled=false",
			"ES_JAVA_OPTS=-Xms256m -Xmx256m",
		}, nil)

	baseURL := fmt.Sprintf("http://localhost:%s", container.HostPort)
	healthURL := fmt.Sprintf("%s/_cluster/health?wait_for_status=yellow", baseURL)
	testutil.WaitForHTTP(t, healthURL, 120*time.Second)

	seedElasticsearch(t, baseURL)

	cfg := &models.DatasourceConfig{
		Connection: models.ConnectionConfig{
			Host: "localhost",
			Port: mustAtoi(container.HostPort),
		},
	}
	client := services.NewESClient(cfg)
	t.Cleanup(func() { client.Close() })

	return client, baseURL
}

func mustAtoi(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

func seedElasticsearch(t *testing.T, baseURL string) {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}

	// Create index with mappings
	indexBody := map[string]any{
		"settings": map[string]any{
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
		"mappings": map[string]any{
			"properties": map[string]any{
				"name":    map[string]any{"type": "text"},
				"email":   map[string]any{"type": "keyword"},
				"age":     map[string]any{"type": "integer"},
				"created": map[string]any{"type": "date"},
			},
		},
	}
	doESHTTPRequest(t, client, "PUT", baseURL+"/test-index", indexBody)

	// Index sample documents
	docs := []map[string]any{
		{"name": "alice", "email": "alice@test.com", "age": 30, "created": "2024-01-01"},
		{"name": "bob", "email": "bob@test.com", "age": 25, "created": "2024-01-02"},
		{"name": "carol", "email": "carol@test.com", "age": 35, "created": "2024-01-03"},
	}
	for _, doc := range docs {
		doESHTTPRequest(t, client, "POST", baseURL+"/test-index/_doc", doc)
	}

	// Refresh to make documents searchable immediately
	doESHTTPRequest(t, client, "POST", baseURL+"/test-index/_refresh", nil)
}

func doESHTTPRequest(t *testing.T, client *http.Client, method, url string, body any) map[string]any {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if resp.StatusCode >= 400 {
		t.Fatalf("ES request failed (%d): %s", resp.StatusCode, string(respBody))
	}
	var result map[string]any
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
	}
	return result
}

// --- Client tests ---

func TestES_Client_Info(t *testing.T) {
	client, _ := setupElasticsearch(t)
	info, err := client.Info(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if info["cluster_name"] == nil || info["cluster_name"] == "" {
		t.Error("expected cluster_name in info response")
	}
	if info["version"] == nil {
		t.Error("expected version in info response")
	}
}

func TestES_Client_GetDatabases(t *testing.T) {
	client, _ := setupElasticsearch(t)
	dbs, err := client.GetDatabases(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(dbs) == 0 {
		t.Fatal("expected at least one database (cluster)")
	}
	// The cluster name should be returned as a virtual database
	found := false
	for _, d := range dbs {
		if d.Name != "" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected non-empty cluster name, got %v", dbs)
	}
}

func TestES_Client_GetTableNames(t *testing.T) {
	client, _ := setupElasticsearch(t)
	tables, err := client.GetTableNames(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tbl := range tables {
		if tbl.Name == "test-index" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected test-index in tables, got %v", tables)
	}
}

func TestES_Client_GetTableInfo(t *testing.T) {
	client, _ := setupElasticsearch(t)
	info, err := client.GetTableInfo(context.Background(), "test-index")
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "test-index" {
		t.Errorf("name = %q", info.Name)
	}
	if len(info.Columns) < 4 {
		t.Fatalf("expected at least 4 columns (name, email, age, created), got %d", len(info.Columns))
	}
	fieldNames := map[string]bool{}
	for _, col := range info.Columns {
		fieldNames[col.Name] = true
	}
	for _, expected := range []string{"name", "email", "age", "created"} {
		if !fieldNames[expected] {
			t.Errorf("expected field %q in columns, got %v", expected, fieldNames)
		}
	}
}

func TestES_Client_Search(t *testing.T) {
	client, _ := setupElasticsearch(t)
	body := map[string]any{
		"query": map[string]any{
			"match": map[string]any{
				"name": "alice",
			},
		},
	}
	result, err := client.Search(context.Background(), "test-index", body, 10)
	if err != nil {
		t.Fatal(err)
	}
	hitsRoot, ok := result["hits"].(map[string]any)
	if !ok {
		t.Fatal("expected hits in search response")
	}
	hits, ok := hitsRoot["hits"].([]any)
	if !ok {
		t.Fatal("expected hits array in search response")
	}
	if len(hits) == 0 {
		t.Error("expected at least 1 search result for 'alice'")
	}
}
