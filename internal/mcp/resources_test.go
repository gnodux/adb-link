package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

func TestHandleRequest_ResourcesList_Empty(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	req := makeRequest("resources/list", 1, nil)

	resp := srv.HandleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", resp.Result)
	}
	resources, ok := result["resources"].([]Resource)
	if !ok {
		t.Fatalf("expected []Resource, got %T", result["resources"])
	}
	if len(resources) != 0 {
		t.Fatalf("expected 0 resources, got %d", len(resources))
	}
}

func TestHandleRequest_ResourcesList_WithResources(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	srv.RegisterResource(Resource{
		URI:      "datasource:///",
		Name:     "Datasources",
		MimeType: "application/json",
	}, func(ctx context.Context, uri string) ([]ResourceContent, error) {
		return []ResourceContent{{URI: uri, Text: "[]"}}, nil
	})

	req := makeRequest("resources/list", 1, nil)
	resp := srv.HandleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	resources := result["resources"].([]Resource)
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if resources[0].URI != "datasource:///" {
		t.Fatalf("expected URI %q, got %q", "datasource:///", resources[0].URI)
	}
}

func TestHandleRequest_ResourcesTemplatesList(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	srv.RegisterResourceTemplate(ResourceTemplate{
		URITemplate: "datasource:///{name}",
		Name:        "Datasource Detail",
		MimeType:    "application/json",
	}, func(ctx context.Context, uri string) ([]ResourceContent, error) {
		return []ResourceContent{{URI: uri, Text: "{}"}}, nil
	})
	srv.RegisterResourceTemplate(ResourceTemplate{
		URITemplate: "datasource:///{name}/{db}/schema",
		Name:        "Database Schema",
		MimeType:    "application/json",
	}, func(ctx context.Context, uri string) ([]ResourceContent, error) {
		return []ResourceContent{{URI: uri, Text: "{}"}}, nil
	})

	req := makeRequest("resources/templates/list", 1, nil)
	resp := srv.HandleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	templates := result["resourceTemplates"].([]ResourceTemplate)
	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}
}

func TestHandleRequest_ResourcesRead_StaticResource(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	srv.RegisterResource(Resource{
		URI:      "datasource:///",
		Name:     "Datasources",
		MimeType: "application/json",
	}, func(ctx context.Context, uri string) ([]ResourceContent, error) {
		return []ResourceContent{{URI: uri, MimeType: "application/json", Text: `[{"name":"test"}]`}}, nil
	})

	req := makeRequest("resources/read", 1, map[string]any{
		"uri": "datasource:///",
	})
	resp := srv.HandleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	contents := result["contents"].([]ResourceContent)
	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}
	if contents[0].Text != `[{"name":"test"}]` {
		t.Fatalf("unexpected text: %v", contents[0].Text)
	}
}

func TestHandleRequest_ResourcesRead_TemplateMatch(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	srv.RegisterResourceTemplate(ResourceTemplate{
		URITemplate: "datasource:///{name}/{db}/tables/{table}",
		Name:        "Table Schema",
		MimeType:    "application/json",
	}, func(ctx context.Context, uri string) ([]ResourceContent, error) {
		parsed, _ := parseDatasourceURI(uri)
		data := map[string]any{
			"name":     parsed.segments[2],
			"ds":       parsed.segments[0],
			"database": parsed.segments[1],
		}
		b, _ := json.Marshal(data)
		return []ResourceContent{{URI: uri, MimeType: "application/json", Text: string(b)}}, nil
	})

	req := makeRequest("resources/read", 1, map[string]any{
		"uri": "datasource:///mysql-prod/mydb/tables/users",
	})
	resp := srv.HandleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	contents := result["contents"].([]ResourceContent)
	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(contents[0].Text), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed["name"] != "users" {
		t.Fatalf("expected name %q, got %v", "users", parsed["name"])
	}
	if parsed["ds"] != "mysql-prod" {
		t.Fatalf("expected ds %q, got %v", "mysql-prod", parsed["ds"])
	}
}

func TestHandleRequest_ResourcesRead_NotFound(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	srv.RegisterResource(Resource{
		URI:  "datasource:///",
		Name: "Datasources",
	}, func(ctx context.Context, uri string) ([]ResourceContent, error) {
		return nil, nil
	})

	req := makeRequest("resources/read", 1, map[string]any{
		"uri": "datasource:///nonexistent/path",
	})
	resp := srv.HandleRequest(context.Background(), req)
	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if resp.Error.Code != ErrCodeResourceNotFound {
		t.Fatalf("expected error code %d, got %d", ErrCodeResourceNotFound, resp.Error.Code)
	}
}

func TestHandleRequest_ResourcesRead_InvalidParams(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	req := makeRequest("resources/read", 1, nil)
	req.Params = json.RawMessage(`{invalid json`)

	resp := srv.HandleRequest(context.Background(), req)
	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if resp.Error.Code != ErrCodeInvalidParams {
		t.Fatalf("expected error code %d, got %d", ErrCodeInvalidParams, resp.Error.Code)
	}
}

func TestHandleRequest_ResourcesRead_HandlerError(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	srv.RegisterResource(Resource{
		URI:  "datasource:///",
		Name: "Datasources",
	}, func(ctx context.Context, uri string) ([]ResourceContent, error) {
		return nil, fmt.Errorf("something went wrong")
	})

	req := makeRequest("resources/read", 1, map[string]any{
		"uri": "datasource:///",
	})
	resp := srv.HandleRequest(context.Background(), req)
	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if resp.Error.Code != ErrCodeInternal {
		t.Fatalf("expected error code %d, got %d", ErrCodeInternal, resp.Error.Code)
	}
}

func TestRegisterResource_NotifyCalled(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	var calledMethod string
	srv.SetNotifyFn(func(method string, params any) {
		calledMethod = method
	})

	srv.RegisterResource(Resource{URI: "test:///", Name: "Test"}, func(ctx context.Context, uri string) ([]ResourceContent, error) {
		return nil, nil
	})

	if calledMethod != "notifications/resources/list_changed" {
		t.Fatalf("expected notification %q, got %q", "notifications/resources/list_changed", calledMethod)
	}
}

func TestRegisterResourceTemplate_NotifyCalled(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	var calledMethod string
	srv.SetNotifyFn(func(method string, params any) {
		calledMethod = method
	})

	srv.RegisterResourceTemplate(ResourceTemplate{URITemplate: "test:///{id}", Name: "Test"}, func(ctx context.Context, uri string) ([]ResourceContent, error) {
		return nil, nil
	})

	if calledMethod != "notifications/resources/list_changed" {
		t.Fatalf("expected notification %q, got %q", "notifications/resources/list_changed", calledMethod)
	}
}

func TestMatchTemplate(t *testing.T) {
	tests := []struct {
		uri      string
		tmpl     string
		wantOK   bool
		wantSegs []string
	}{
		{"datasource:///mysql-prod", "datasource:///{name}", true, []string{"mysql-prod"}},
		{"datasource:///mysql-prod/mydb/databases", "datasource:///{name}/{db}/databases", true, []string{"mysql-prod", "mydb"}},
		{"datasource:///mysql-prod/mydb/schema", "datasource:///{name}/{db}/schema", true, []string{"mysql-prod", "mydb"}},
		{"datasource:///mysql-prod/mydb/tables/users", "datasource:///{name}/{db}/tables/{table}", true, []string{"mysql-prod", "mydb", "users"}},
		{"datasource:///mysql-prod/mydb/views/v_active", "datasource:///{name}/{db}/views/{view}", true, []string{"mysql-prod", "mydb", "v_active"}},
		{"datasource:///", "datasource:///{name}", false, nil},
		{"datasource:///a/b/c/d", "datasource:///{name}", false, nil},
		{"datasource:///a/b/schema", "datasource:///{name}/{db}/databases", false, nil},
	}

	for _, tt := range tests {
		ok, segs := matchTemplate(tt.uri, tt.tmpl)
		if ok != tt.wantOK {
			t.Errorf("matchTemplate(%q, %q) = %v, want %v", tt.uri, tt.tmpl, ok, tt.wantOK)
			continue
		}
		if ok && len(segs) != len(tt.wantSegs) {
			t.Errorf("matchTemplate(%q, %q) segments = %v, want %v", tt.uri, tt.tmpl, segs, tt.wantSegs)
		}
	}
}

func TestParseDatasourceURI(t *testing.T) {
	tests := []struct {
		uri      string
		wantKind datasourceURIKind
		wantSegs []string
		wantErr  bool
	}{
		{"datasource:///", uriList, nil, false},
		{"datasource:///mysql-prod", uriDatasource, []string{"mysql-prod"}, false},
		{"datasource:///mysql-prod/mydb/databases", uriDatabases, []string{"mysql-prod", "mydb"}, false},
		{"datasource:///mysql-prod/mydb/schema", uriSchema, []string{"mysql-prod", "mydb"}, false},
		{"datasource:///mysql-prod/mydb/tables/users", uriTable, []string{"mysql-prod", "mydb", "users"}, false},
		{"datasource:///mysql-prod/mydb/views/v1", uriView, []string{"mysql-prod", "mydb", "v1"}, false},
		{"datasource:///a/b/unknown", "", nil, true},
		{"datasource:///a/b/c/d/e", "", nil, true},
	}

	for _, tt := range tests {
		parsed, err := parseDatasourceURI(tt.uri)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseDatasourceURI(%q) expected error, got nil", tt.uri)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseDatasourceURI(%q) unexpected error: %v", tt.uri, err)
			continue
		}
		if parsed.kind != tt.wantKind {
			t.Errorf("parseDatasourceURI(%q) kind = %q, want %q", tt.uri, parsed.kind, tt.wantKind)
		}
		if len(parsed.segments) != len(tt.wantSegs) {
			t.Errorf("parseDatasourceURI(%q) segments = %v, want %v", tt.uri, parsed.segments, tt.wantSegs)
		}
	}
}

func TestNewServer_ResourcesInitialized(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	if srv.resources == nil {
		t.Fatal("expected resources map to be initialized")
	}
	if len(srv.resources) != 0 {
		t.Fatalf("expected empty resources map, got %d entries", len(srv.resources))
	}
	if len(srv.templates) != 0 {
		t.Fatalf("expected empty templates slice, got %d entries", len(srv.templates))
	}
}
