package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLIParseFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantJSON bool
		wantDB   string
		wantLim  int
		wantPos  int
	}{
		{
			name:     "no flags",
			args:     []string{"ds", "SELECT 1"},
			wantJSON: false, wantDB: "", wantLim: 1000, wantPos: 2,
		},
		{
			name:     "json flag",
			args:     []string{"--json", "ds", "SELECT 1"},
			wantJSON: true, wantDB: "", wantLim: 1000, wantPos: 2,
		},
		{
			name:     "database flag",
			args:     []string{"--database", "mydb", "ds", "SELECT 1"},
			wantJSON: false, wantDB: "mydb", wantLim: 1000, wantPos: 2,
		},
		{
			name:     "limit flag",
			args:     []string{"--limit", "50", "ds", "SELECT 1"},
			wantJSON: false, wantDB: "", wantLim: 50, wantPos: 2,
		},
		{
			name:     "all flags",
			args:     []string{"--json", "--database", "mydb", "--limit", "10", "--timeout", "60", "ds", "SELECT 1"},
			wantJSON: true, wantDB: "mydb", wantLim: 10, wantPos: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags, pos, err := parseFlags(tt.args)
			if err != nil {
				t.Fatalf("parseFlags() error = %v", err)
			}
			if flags.JSON != tt.wantJSON {
				t.Errorf("JSON = %v, want %v", flags.JSON, tt.wantJSON)
			}
			if flags.Database != tt.wantDB {
				t.Errorf("Database = %q, want %q", flags.Database, tt.wantDB)
			}
			if flags.Limit != tt.wantLim {
				t.Errorf("Limit = %d, want %d", flags.Limit, tt.wantLim)
			}
			if len(pos) != tt.wantPos {
				t.Errorf("positional args = %d, want %d", len(pos), tt.wantPos)
			}
		})
	}
}

func TestCLIUsage(t *testing.T) {
	var buf bytes.Buffer
	cliUsage(&buf)
	out := buf.String()
	if !strings.Contains(out, "list-datasources") {
		t.Error("usage should mention list-datasources")
	}
	if !strings.Contains(out, "query") {
		t.Error("usage should mention query")
	}
	if !strings.Contains(out, "ping") {
		t.Error("usage should mention ping")
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		input any
		want  string
	}{
		{nil, "NULL"},
		{"hello", "hello"},
		{int64(42), "42"},
		{float64(3.14), "3.14"},
		{float64(100), "100"},
		{[]byte("data"), "data"},
		{true, "true"},
	}
	for _, tt := range tests {
		got := formatValue(tt.input)
		if got != tt.want {
			t.Errorf("formatValue(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
