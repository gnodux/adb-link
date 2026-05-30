package models

import (
	"encoding/json"
	"testing"
)

func boolPtr(b bool) *bool { return &b }

func TestDatasourceConfig_IsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		enable *bool
		want   bool
	}{
		{"nil defaults to true", nil, true},
		{"explicit true", boolPtr(true), true},
		{"explicit false", boolPtr(false), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DatasourceConfig{Enable: tt.enable}
			if got := d.IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetadataConfig_IsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		enable *bool
		want   bool
	}{
		{"nil defaults to true", nil, true},
		{"explicit true", boolPtr(true), true},
		{"explicit false", boolPtr(false), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MetadataConfig{Enable: tt.enable}
			if got := m.IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToolConfig_IsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		enable *bool
		want   bool
	}{
		{"nil defaults to true", nil, true},
		{"explicit true", boolPtr(true), true},
		{"explicit false", boolPtr(false), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := &ToolConfig{Enable: tt.enable}
			if got := tc.IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToolsetConfig_IsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		enable *bool
		want   bool
	}{
		{"nil defaults to true", nil, true},
		{"explicit true", boolPtr(true), true},
		{"explicit false", boolPtr(false), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &ToolsetConfig{Enable: tt.enable}
			if got := ts.IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthUser_IsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		enable *bool
		want   bool
	}{
		{"nil defaults to true", nil, true},
		{"explicit true", boolPtr(true), true},
		{"explicit false", boolPtr(false), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AuthUser{Enable: tt.enable}
			if got := a.IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPermissionConfig_IsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		enable *bool
		want   bool
	}{
		{"nil defaults to true", nil, true},
		{"explicit true", boolPtr(true), true},
		{"explicit false", boolPtr(false), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PermissionConfig{Enable: tt.enable}
			if got := p.IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToolConfig_BuildInputSchema_RawJSON(t *testing.T) {
	raw := json.RawMessage(`{"type":"object","properties":{"q":{"type":"string"}},"required":["q"]}`)
	tc := &ToolConfig{InputSchema: raw}
	schema := tc.BuildInputSchema()
	if schema["type"] != "object" {
		t.Errorf("expected type=object, got %v", schema["type"])
	}
	props, _ := schema["properties"].(map[string]any)
	if props == nil || props["q"] == nil {
		t.Fatal("expected property 'q' in schema")
	}
	req, _ := schema["required"].([]any)
	if len(req) != 1 || req[0] != "q" {
		t.Errorf("expected required=[q], got %v", req)
	}
}

func TestToolConfig_BuildInputSchema_Parameters(t *testing.T) {
	tc := &ToolConfig{
		Parameters: []ToolParameter{
			{Name: "name", Type: "string", Description: "user name", Required: true},
			{Name: "age", Type: "integer", Description: "user age"},
			{Name: "active", Type: "boolean", Default: true},
		},
	}
	schema := tc.BuildInputSchema()
	if schema["type"] != "object" {
		t.Errorf("expected type=object, got %v", schema["type"])
	}
	props, _ := schema["properties"].(map[string]any)
	if props == nil {
		t.Fatal("expected properties in schema")
	}

	nameProp, _ := props["name"].(map[string]any)
	if nameProp["type"] != "string" {
		t.Errorf("name type = %v, want string", nameProp["type"])
	}
	if nameProp["description"] != "user name" {
		t.Errorf("name description = %v, want 'user name'", nameProp["description"])
	}

	ageProp, _ := props["age"].(map[string]any)
	if ageProp["type"] != "integer" {
		t.Errorf("age type = %v, want integer", ageProp["type"])
	}

	activeProp, _ := props["active"].(map[string]any)
	if activeProp["type"] != "boolean" {
		t.Errorf("active type = %v, want boolean", activeProp["type"])
	}
	if activeProp["default"] != true {
		t.Errorf("active default = %v, want true", activeProp["default"])
	}

	req, ok := schema["required"].([]string)
	if !ok || len(req) != 1 || req[0] != "name" {
		t.Errorf("required = %v, want [name]", schema["required"])
	}
}

func TestToolConfig_BuildInputSchema_Empty(t *testing.T) {
	tc := &ToolConfig{}
	schema := tc.BuildInputSchema()
	if schema["type"] != "object" {
		t.Errorf("expected type=object, got %v", schema["type"])
	}
	props, _ := schema["properties"].(map[string]any)
	if len(props) != 0 {
		t.Errorf("expected empty properties, got %v", props)
	}
}

func TestToolConfig_BuildInputSchema_DefaultTypeIsString(t *testing.T) {
	tc := &ToolConfig{
		Parameters: []ToolParameter{
			{Name: "q"}, // no type specified
		},
	}
	schema := tc.BuildInputSchema()
	props, _ := schema["properties"].(map[string]any)
	qProp, _ := props["q"].(map[string]any)
	if qProp["type"] != "string" {
		t.Errorf("default type = %v, want string", qProp["type"])
	}
}

func TestDialectInfoMap_AllTypes(t *testing.T) {
	allTypes := []DatabaseType{
		DatabaseTypeMySQL,
		DatabaseTypePostgreSQL,
		DatabaseTypeSQLite,
		DatabaseTypeClickHouse,
		DatabaseTypeMSSQL,
		DatabaseTypeElasticsearch,
		DatabaseTypeHive,
	}
	for _, dt := range allTypes {
		t.Run(string(dt), func(t *testing.T) {
			info, ok := DialectInfoMap[dt]
			if !ok {
				t.Fatalf("DialectInfoMap missing type %s", dt)
			}
			if info.SQLStyle == "" {
				t.Errorf("%s: SQLStyle is empty", dt)
			}
			if len(info.Features) == 0 {
				t.Errorf("%s: Features is empty", dt)
			}
		})
	}
}

func TestDialectInfoMap_Fields(t *testing.T) {
	for dt, info := range DialectInfoMap {
		t.Run(string(dt), func(t *testing.T) {
			if info.LimitSyntax == "" {
				t.Errorf("%s: LimitSyntax is empty", dt)
			}
			if info.Notes == "" {
				t.Errorf("%s: Notes is empty", dt)
			}
		})
	}
}
