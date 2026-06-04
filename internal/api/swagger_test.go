package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOpenAPISpecValid(t *testing.T) {
	// Verify the embedded YAML is valid and parseable
	if openapiSpecYAML == "" {
		t.Fatal("embedded openapi spec is empty")
	}

	var spec map[string]interface{}
	if err := yaml.Unmarshal([]byte(openapiSpecYAML), &spec); err != nil {
		t.Fatalf("YAML parse error: %v", err)
	}

	// Check required top-level fields
	for _, key := range []string{"openapi", "info", "paths"} {
		if _, ok := spec[key]; !ok {
			t.Errorf("missing required top-level field: %s", key)
		}
	}

	// Verify openapi version
	version, _ := spec["openapi"].(string)
	if !strings.HasPrefix(version, "3.0") {
		t.Errorf("expected OpenAPI 3.0.x, got %s", version)
	}

	// Verify info has title and version
	info, _ := spec["info"].(map[string]interface{})
	if info == nil {
		t.Fatal("info field is not a map")
	}
	if _, ok := info["title"]; !ok {
		t.Error("info.title is missing")
	}
	if _, ok := info["version"]; !ok {
		t.Error("info.version is missing")
	}
}

func TestOpenAPISpecJSON(t *testing.T) {
	data, err := openAPISpecJSON()
	if err != nil {
		t.Fatalf("openAPISpecJSON() failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("JSON output is empty")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON output is invalid: %v", err)
	}

	if _, ok := parsed["openapi"]; !ok {
		t.Error("JSON output missing 'openapi' field")
	}
	if _, ok := parsed["paths"]; !ok {
		t.Error("JSON output missing 'paths' field")
	}
}

func TestOpenAPISpecCoversAllEndpoints(t *testing.T) {
	var spec map[string]interface{}
	if err := yaml.Unmarshal([]byte(openapiSpecYAML), &spec); err != nil {
		t.Fatalf("YAML parse error: %v", err)
	}

	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("paths field is not a map")
	}

	// All API paths from router.go that should be documented
	requiredPaths := []string{
		"/api/health",
		"/api/datasources/list",
		"/api/datasources/detail",
		"/api/datasources/test",
		"/api/datasources/register",
		"/api/datasources/unregister",
		"/api/databases/list",
		"/api/schema/get",
		"/api/schema/table",
		"/api/schema/view",
		"/api/query/execute",
		"/api/query/explain",
		"/api/async/query/submit",
		"/api/async/query/status",
		"/api/async/query/result",
		"/api/async/query/cancel",
		"/api/tools",
		"/api/tool/register",
		"/api/tool/unregister",
		"/api/tool/async/{tool_name}/submit",
		"/api/tool/async/{tool_name}/status",
		"/api/tool/async/{tool_name}/result",
		"/api/tool/async/{tool_name}/cancel",
		"/api/tool/{tool_name}",
	}

	for _, p := range requiredPaths {
		if _, ok := paths[p]; !ok {
			t.Errorf("endpoint %s is missing from OpenAPI spec", p)
		}
	}
}

func TestOpenAPISpecHasSecurityScheme(t *testing.T) {
	var spec map[string]interface{}
	if err := yaml.Unmarshal([]byte(openapiSpecYAML), &spec); err != nil {
		t.Fatalf("YAML parse error: %v", err)
	}

	components, ok := spec["components"].(map[string]interface{})
	if !ok {
		t.Fatal("components field is missing or not a map")
	}
	schemes, ok := components["securitySchemes"].(map[string]interface{})
	if !ok {
		t.Fatal("securitySchemes is missing or not a map")
	}
	bearer, ok := schemes["bearerAuth"].(map[string]interface{})
	if !ok {
		t.Fatal("bearerAuth scheme is missing")
	}
	if bearer["type"] != "http" {
		t.Errorf("bearerAuth.type should be 'http', got %v", bearer["type"])
	}
	if bearer["scheme"] != "bearer" {
		t.Errorf("bearerAuth.scheme should be 'bearer', got %v", bearer["scheme"])
	}
}

func TestSwaggerUIEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	registerSwaggerRoutes(mux)

	req := httptest.NewRequest("GET", "/api/swagger/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html Content-Type, got %s", ct)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "swagger-ui") {
		t.Error("response body does not contain swagger-ui reference")
	}
}

func TestSwaggerDocJSONEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	registerSwaggerRoutes(mux)

	req := httptest.NewRequest("GET", "/api/swagger/doc.json", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected application/json Content-Type, got %s", ct)
	}

	body, _ := io.ReadAll(resp.Body)
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("doc.json is not valid JSON: %v", err)
	}
	if _, ok := parsed["openapi"]; !ok {
		t.Error("doc.json missing 'openapi' field")
	}
}

func TestSwaggerRedirect(t *testing.T) {
	mux := http.NewServeMux()
	registerSwaggerRoutes(mux)

	req := httptest.NewRequest("GET", "/api/swagger", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMovedPermanently {
		t.Errorf("expected 301 redirect, got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if loc != "/api/swagger/" {
		t.Errorf("expected redirect to /api/swagger/, got %s", loc)
	}
}
