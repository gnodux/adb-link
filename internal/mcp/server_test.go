package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

// makeRequest builds a Request struct for testing.
func makeRequest(method string, id int, params any) *Request {
	idJSON, _ := json.Marshal(id)
	var paramsJSON json.RawMessage
	if params != nil {
		paramsJSON, _ = json.Marshal(params)
	}
	return &Request{JSONRPC: "2.0", ID: idJSON, Method: method, Params: paramsJSON}
}

func TestNewServer(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	if srv == nil {
		t.Fatal("NewServer returned nil")
	}
	if len(srv.tools) != 0 {
		t.Fatalf("expected empty tools map, got %d entries", len(srv.tools))
	}
}

func TestHandleRequest_InvalidJSONRPCVersion(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	req := &Request{JSONRPC: "1.0", Method: "initialize"}
	req.ID, _ = json.Marshal(1)

	resp := srv.HandleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if resp.Error.Code != ErrCodeInvalidRequest {
		t.Fatalf("expected error code %d, got %d", ErrCodeInvalidRequest, resp.Error.Code)
	}
}

func TestHandleRequest_Initialize(t *testing.T) {
	srv := NewServer("myserver", "2.0.0")
	req := makeRequest("initialize", 1, nil)

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

	if result["protocolVersion"] != ProtocolVersion {
		t.Fatalf("expected protocolVersion %q, got %v", ProtocolVersion, result["protocolVersion"])
	}

	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatalf("expected serverInfo map, got %T", result["serverInfo"])
	}
	if serverInfo["name"] != "myserver" {
		t.Fatalf("expected serverInfo.name %q, got %v", "myserver", serverInfo["name"])
	}
	if serverInfo["version"] != "2.0.0" {
		t.Fatalf("expected serverInfo.version %q, got %v", "2.0.0", serverInfo["version"])
	}

	caps, ok := result["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("expected capabilities map, got %T", result["capabilities"])
	}
	toolsCap, ok := caps["tools"].(map[string]any)
	if !ok {
		t.Fatalf("expected tools capability map, got %T", caps["tools"])
	}
	if toolsCap["listChanged"] != true {
		t.Fatalf("expected listChanged=true, got %v", toolsCap["listChanged"])
	}
}

func TestHandleRequest_Ping(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	req := makeRequest("ping", 1, nil)

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
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %v", result)
	}
}

func TestHandleRequest_ToolsList_Empty(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	req := makeRequest("tools/list", 1, nil)

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
	tools, ok := result["tools"].([]Tool)
	if !ok {
		t.Fatalf("expected []Tool, got %T", result["tools"])
	}
	if len(tools) != 0 {
		t.Fatalf("expected 0 tools, got %d", len(tools))
	}
}

func TestHandleRequest_ToolsList_WithTools(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	srv.RegisterTool(Tool{Name: "tool_a", Description: "first"}, func(ctx context.Context, args map[string]any) (string, error) {
		return "a", nil
	})
	srv.RegisterTool(Tool{Name: "tool_b", Description: "second"}, func(ctx context.Context, args map[string]any) (string, error) {
		return "b", nil
	})

	req := makeRequest("tools/list", 1, nil)
	resp := srv.HandleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	tools := result["tools"].([]Tool)
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}
	if !names["tool_a"] || !names["tool_b"] {
		t.Fatalf("expected tool_a and tool_b, got %v", names)
	}
}

func TestHandleRequest_ToolsCall_Success(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	srv.RegisterTool(Tool{Name: "greet", Description: "greet"}, func(ctx context.Context, args map[string]any) (string, error) {
		return "hello", nil
	})

	req := makeRequest("tools/call", 1, map[string]any{
		"name":      "greet",
		"arguments": map[string]any{},
	})

	resp := srv.HandleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	content := result["content"].([]map[string]any)
	if len(content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(content))
	}
	if content[0]["text"] != "hello" {
		t.Fatalf("expected text %q, got %v", "hello", content[0]["text"])
	}
}

func TestHandleRequest_ToolsCall_ToolNotFound(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	req := makeRequest("tools/call", 1, map[string]any{
		"name":      "nonexistent",
		"arguments": map[string]any{},
	})

	resp := srv.HandleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if resp.Error.Code != ErrCodeMethodNotFound {
		t.Fatalf("expected error code %d, got %d", ErrCodeMethodNotFound, resp.Error.Code)
	}
}

func TestHandleRequest_ToolsCall_InvalidParams(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	// Set Params to invalid JSON directly
	req := makeRequest("tools/call", 1, nil)
	req.Params = json.RawMessage(`{invalid json`)

	resp := srv.HandleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if resp.Error.Code != ErrCodeInvalidParams {
		t.Fatalf("expected error code %d, got %d", ErrCodeInvalidParams, resp.Error.Code)
	}
}

func TestHandleRequest_ToolsCall_HandlerError(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	srv.RegisterTool(Tool{Name: "fail", Description: "fails"}, func(ctx context.Context, args map[string]any) (string, error) {
		return "", fmt.Errorf("something went wrong")
	})

	req := makeRequest("tools/call", 1, map[string]any{
		"name":      "fail",
		"arguments": map[string]any{},
	})

	resp := srv.HandleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	if result["isError"] != true {
		t.Fatalf("expected isError=true, got %v", result["isError"])
	}
	content := result["content"].([]map[string]any)
	if content[0]["text"] != "Error: something went wrong" {
		t.Fatalf("unexpected text: %v", content[0]["text"])
	}
}

func TestHandleRequest_Initialized_NoResponse(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	req := makeRequest("initialized", 1, nil)

	resp := srv.HandleRequest(context.Background(), req)
	if resp != nil {
		t.Fatalf("expected nil response for initialized notification, got %v", resp)
	}
}

func TestHandleRequest_UnknownMethod_WithID(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	req := makeRequest("unknown/method", 1, nil)

	resp := srv.HandleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}
	if resp.Error.Code != ErrCodeMethodNotFound {
		t.Fatalf("expected error code %d, got %d", ErrCodeMethodNotFound, resp.Error.Code)
	}
}

func TestHandleRequest_UnknownNotification_NoResponse(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	// No ID means it's a notification
	req := &Request{JSONRPC: "2.0", Method: "unknown/notification"}

	resp := srv.HandleRequest(context.Background(), req)
	if resp != nil {
		t.Fatalf("expected nil response for unknown notification, got %v", resp)
	}
}

func TestRegisterTool_AppearsInList(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	srv.RegisterTool(Tool{Name: "mytool", Description: "desc"}, func(ctx context.Context, args map[string]any) (string, error) {
		return "ok", nil
	})

	req := makeRequest("tools/list", 1, nil)
	resp := srv.HandleRequest(context.Background(), req)

	result := resp.Result.(map[string]any)
	tools := result["tools"].([]Tool)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "mytool" {
		t.Fatalf("expected tool name %q, got %q", "mytool", tools[0].Name)
	}
}

func TestRegisterTool_ReplaceExisting(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	srv.RegisterTool(Tool{Name: "mytool", Description: "first"}, func(ctx context.Context, args map[string]any) (string, error) {
		return "first", nil
	})
	srv.RegisterTool(Tool{Name: "mytool", Description: "second"}, func(ctx context.Context, args map[string]any) (string, error) {
		return "second", nil
	})

	// Call the tool to verify second handler is used
	req := makeRequest("tools/call", 1, map[string]any{
		"name":      "mytool",
		"arguments": map[string]any{},
	})
	resp := srv.HandleRequest(context.Background(), req)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result := resp.Result.(map[string]any)
	content := result["content"].([]map[string]any)
	if content[0]["text"] != "second" {
		t.Fatalf("expected %q, got %v", "second", content[0]["text"])
	}
}

func TestUnregisterTool_RemovedFromList(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	srv.RegisterTool(Tool{Name: "mytool", Description: "desc"}, func(ctx context.Context, args map[string]any) (string, error) {
		return "ok", nil
	})

	removed := srv.UnregisterTool("mytool")
	if !removed {
		t.Fatal("expected UnregisterTool to return true")
	}

	req := makeRequest("tools/list", 1, nil)
	resp := srv.HandleRequest(context.Background(), req)

	result := resp.Result.(map[string]any)
	tools := result["tools"].([]Tool)
	if len(tools) != 0 {
		t.Fatalf("expected 0 tools after unregister, got %d", len(tools))
	}
}

func TestUnregisterTool_NonExistent_ReturnsFalse(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	removed := srv.UnregisterTool("nonexistent")
	if removed {
		t.Fatal("expected UnregisterTool to return false for non-existent tool")
	}
}

func TestSetNotifyFn_CalledOnRegister(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	var calledMethod string
	srv.SetNotifyFn(func(method string, params any) {
		calledMethod = method
	})

	srv.RegisterTool(Tool{Name: "t", Description: "d"}, func(ctx context.Context, args map[string]any) (string, error) {
		return "", nil
	})

	if calledMethod != "notifications/tools/list_changed" {
		t.Fatalf("expected notification method %q, got %q", "notifications/tools/list_changed", calledMethod)
	}
}

func TestSetNotifyFn_CalledOnUnregister(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	srv.RegisterTool(Tool{Name: "t", Description: "d"}, func(ctx context.Context, args map[string]any) (string, error) {
		return "", nil
	})

	var calledMethod string
	srv.SetNotifyFn(func(method string, params any) {
		calledMethod = method
	})

	srv.UnregisterTool("t")

	if calledMethod != "notifications/tools/list_changed" {
		t.Fatalf("expected notification method %q, got %q", "notifications/tools/list_changed", calledMethod)
	}
}

func TestSetNotifyFn_NotCalledWhenUnregisterMiss(t *testing.T) {
	srv := NewServer("test", "1.0.0")
	called := false
	srv.SetNotifyFn(func(method string, params any) {
		called = true
	})

	srv.UnregisterTool("nonexistent")

	if called {
		t.Fatal("expected notifyFn not to be called when unregistering non-existent tool")
	}
}
