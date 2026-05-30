package mcp

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestSchemaObject_NilProps(t *testing.T) {
	result := schemaObject(nil, nil)

	if result["type"] != "object" {
		t.Fatalf("expected type %q, got %v", "object", result["type"])
	}
	props, ok := result["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties map, got %T", result["properties"])
	}
	if len(props) != 0 {
		t.Fatalf("expected empty properties, got %v", props)
	}
	if _, hasRequired := result["required"]; hasRequired {
		t.Fatal("expected no required field when nil")
	}
}

func TestSchemaObject_WithProps(t *testing.T) {
	props := map[string]any{
		"name": prop("string", "user name"),
		"age":  prop("integer", "user age"),
	}
	result := schemaObject(props, nil)

	if result["type"] != "object" {
		t.Fatalf("expected type %q, got %v", "object", result["type"])
	}
	rp, ok := result["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties map, got %T", result["properties"])
	}
	if len(rp) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(rp))
	}
	nameProp := rp["name"].(map[string]any)
	if nameProp["type"] != "string" {
		t.Fatalf("expected name type %q, got %v", "string", nameProp["type"])
	}
}

func TestSchemaObject_WithRequired(t *testing.T) {
	props := map[string]any{
		"name": prop("string", "user name"),
	}
	required := []string{"name"}
	result := schemaObject(props, required)

	reqArr, ok := result["required"].([]string)
	if !ok {
		t.Fatalf("expected required []string, got %T", result["required"])
	}
	if len(reqArr) != 1 || reqArr[0] != "name" {
		t.Fatalf("expected required=[%q], got %v", "name", reqArr)
	}
}

func TestProp(t *testing.T) {
	result := prop("string", "desc")

	if result["type"] != "string" {
		t.Fatalf("expected type %q, got %v", "string", result["type"])
	}
	if result["description"] != "desc" {
		t.Fatalf("expected description %q, got %v", "desc", result["description"])
	}
	if _, hasDefault := result["default"]; hasDefault {
		t.Fatal("expected no default field")
	}
}

func TestPropWithDefault(t *testing.T) {
	result := propWithDefault("integer", "desc", 42)

	if result["type"] != "integer" {
		t.Fatalf("expected type %q, got %v", "integer", result["type"])
	}
	if result["description"] != "desc" {
		t.Fatalf("expected description %q, got %v", "desc", result["description"])
	}
	if result["default"] != 42 {
		t.Fatalf("expected default %v, got %v", 42, result["default"])
	}
}

func TestStringArg_Present(t *testing.T) {
	args := map[string]any{"name": "alice"}
	result := stringArg(args, "name")
	if result != "alice" {
		t.Fatalf("expected %q, got %q", "alice", result)
	}
}

func TestStringArg_Missing(t *testing.T) {
	args := map[string]any{"other": "value"}
	result := stringArg(args, "name")
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestStringArg_NonString(t *testing.T) {
	args := map[string]any{"count": 42}
	result := stringArg(args, "count")
	expected := fmt.Sprintf("%v", 42)
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestIntArg_Float64(t *testing.T) {
	args := map[string]any{"limit": float64(100)}
	result := intArg(args, "limit", 10)
	if result != 100 {
		t.Fatalf("expected 100, got %d", result)
	}
}

func TestIntArg_Int(t *testing.T) {
	args := map[string]any{"limit": 50}
	result := intArg(args, "limit", 10)
	if result != 50 {
		t.Fatalf("expected 50, got %d", result)
	}
}

func TestIntArg_Int64(t *testing.T) {
	args := map[string]any{"limit": int64(75)}
	result := intArg(args, "limit", 10)
	if result != 75 {
		t.Fatalf("expected 75, got %d", result)
	}
}

func TestIntArg_Missing(t *testing.T) {
	args := map[string]any{"other": 5}
	result := intArg(args, "limit", 10)
	if result != 10 {
		t.Fatalf("expected default 10, got %d", result)
	}
}

func TestIntArg_InvalidType(t *testing.T) {
	args := map[string]any{"limit": "not_a_number"}
	result := intArg(args, "limit", 10)
	if result != 10 {
		t.Fatalf("expected default 10, got %d", result)
	}
}

func TestJsonString_Object(t *testing.T) {
	input := map[string]any{"key": "value"}
	result, err := jsonString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if parsed["key"] != "value" {
		t.Fatalf("expected key=%q, got %v", "value", parsed["key"])
	}
}

func TestJsonString_Slice(t *testing.T) {
	input := []string{"a", "b", "c"}
	result, err := jsonString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed []string
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if len(parsed) != 3 || parsed[0] != "a" || parsed[1] != "b" || parsed[2] != "c" {
		t.Fatalf("expected [a,b,c], got %v", parsed)
	}
}

func TestJsonString_Nil(t *testing.T) {
	result, err := jsonString(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "null" {
		t.Fatalf("expected %q, got %q", "null", result)
	}
}
