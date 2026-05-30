package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/gnodux/adb-link/internal/models"
)

// ---------------------------------------------------------------------------
// convertNamedParams
// ---------------------------------------------------------------------------

func TestConvertNamedParams_MySQL(t *testing.T) {
	tmpl := "SELECT * FROM users WHERE name = :name AND age = :age"
	params := map[string]any{"name": "alice", "age": 30}

	sql, args := convertNamedParams(tmpl, params, models.DatabaseTypeMySQL)

	if sql != "SELECT * FROM users WHERE name = ? AND age = ?" {
		t.Fatalf("unexpected SQL: %s", sql)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[0] != "alice" {
		t.Errorf("args[0] = %v, want alice", args[0])
	}
	if args[1] != 30 {
		t.Errorf("args[1] = %v, want 30", args[1])
	}
}

func TestConvertNamedParams_PostgreSQL(t *testing.T) {
	tmpl := "SELECT * FROM users WHERE name = :name AND age = :age"
	params := map[string]any{"name": "bob", "age": 25}

	sql, args := convertNamedParams(tmpl, params, models.DatabaseTypePostgreSQL)

	if sql != "SELECT * FROM users WHERE name = $1 AND age = $2" {
		t.Fatalf("unexpected SQL: %s", sql)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[0] != "bob" {
		t.Errorf("args[0] = %v, want bob", args[0])
	}
	if args[1] != 25 {
		t.Errorf("args[1] = %v, want 25", args[1])
	}
}

func TestConvertNamedParams_MSSQL(t *testing.T) {
	tmpl := "SELECT * FROM users WHERE name = :name AND age = :age"
	params := map[string]any{"name": "carol", "age": 40}

	sql, args := convertNamedParams(tmpl, params, models.DatabaseTypeMSSQL)

	if sql != "SELECT * FROM users WHERE name = @p1 AND age = @p2" {
		t.Fatalf("unexpected SQL: %s", sql)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[0] != "carol" {
		t.Errorf("args[0] = %v, want carol", args[0])
	}
	if args[1] != 40 {
		t.Errorf("args[1] = %v, want 40", args[1])
	}
}

func TestConvertNamedParams_MissingParam(t *testing.T) {
	tmpl := "SELECT * FROM users WHERE name = :name AND age = :age"
	params := map[string]any{"name": "dave"}

	sql, args := convertNamedParams(tmpl, params, models.DatabaseTypeMySQL)

	if sql != "SELECT * FROM users WHERE name = ? AND age = ?" {
		t.Fatalf("unexpected SQL: %s", sql)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[0] != "dave" {
		t.Errorf("args[0] = %v, want dave", args[0])
	}
	if args[1] != nil {
		t.Errorf("args[1] = %v, want nil", args[1])
	}
}

func TestConvertNamedParams_MultipleParamsMaintainOrder(t *testing.T) {
	tmpl := "SELECT :a, :b, :c"
	params := map[string]any{"a": 1, "b": 2, "c": 3}

	sql, args := convertNamedParams(tmpl, params, models.DatabaseTypePostgreSQL)

	if sql != "SELECT $1, $2, $3" {
		t.Fatalf("unexpected SQL: %s", sql)
	}
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(args))
	}
	// Params appear in template order: a, b, c
	if args[0] != 1 || args[1] != 2 || args[2] != 3 {
		t.Errorf("args = %v, want [1 2 3]", args)
	}
}

func TestConvertNamedParams_NoParams(t *testing.T) {
	tmpl := "SELECT 1"
	params := map[string]any{}

	sql, args := convertNamedParams(tmpl, params, models.DatabaseTypeMySQL)

	if sql != "SELECT 1" {
		t.Fatalf("unexpected SQL: %s", sql)
	}
	if len(args) != 0 {
		t.Fatalf("expected 0 args, got %d", len(args))
	}
}

func TestConvertNamedParams_ClickHouse_DefaultsToQuestionMark(t *testing.T) {
	tmpl := "SELECT :x"
	params := map[string]any{"x": 42}

	for _, dbType := range []models.DatabaseType{
		models.DatabaseTypeClickHouse,
		models.DatabaseTypeSQLite,
		models.DatabaseTypeHive,
	} {
		sql, args := convertNamedParams(tmpl, params, dbType)
		if sql != "SELECT ?" {
			t.Errorf("[%s] unexpected SQL: %s", dbType, sql)
		}
		if len(args) != 1 || args[0] != 42 {
			t.Errorf("[%s] unexpected args: %v", dbType, args)
		}
	}
}

// ---------------------------------------------------------------------------
// serializeValue
// ---------------------------------------------------------------------------

func TestSerializeValue_Nil(t *testing.T) {
	if got := serializeValue(nil); got != nil {
		t.Errorf("serializeValue(nil) = %v, want nil", got)
	}
}

func TestSerializeValue_ByteSlicePrintable(t *testing.T) {
	got := serializeValue([]byte("hello"))
	s, ok := got.(string)
	if !ok || s != "hello" {
		t.Errorf("serializeValue([]byte(\"hello\")) = %v (%T), want \"hello\"", got, got)
	}
}

func TestSerializeValue_ByteSliceBinary(t *testing.T) {
	got := serializeValue([]byte{0x00, 0x01, 0x02})
	s, ok := got.(string)
	if !ok || s != "000102" {
		t.Errorf("serializeValue(binary bytes) = %v, want \"000102\"", got)
	}
}

func TestSerializeValue_Time(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 123456789, time.UTC)
	got := serializeValue(ts)
	s, ok := got.(string)
	if !ok {
		t.Fatalf("serializeValue(time.Time) returned %T, want string", got)
	}
	want := ts.Format(time.RFC3339Nano)
	if s != want {
		t.Errorf("serializeValue(time) = %q, want %q", s, want)
	}
}

func TestSerializeValue_PrimitiveTypes(t *testing.T) {
	tests := []struct {
		name string
		in   any
	}{
		{"int", 42},
		{"int64", int64(99)},
		{"float64", 3.14},
		{"string", "hello"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := serializeValue(tc.in)
			if got != tc.in {
				t.Errorf("serializeValue(%v) = %v, want same value", tc.in, got)
			}
		})
	}
}

func TestSerializeValue_MapAndSlice(t *testing.T) {
	m := map[string]any{"k": "v"}
	if got := serializeValue(m); got == nil {
		t.Error("serializeValue(map) returned nil")
	}

	s := []any{1, 2, 3}
	if got := serializeValue(s); got == nil {
		t.Error("serializeValue(slice) returned nil")
	}
}

func TestSerializeValue_UnknownType(t *testing.T) {
	type custom struct{ X int }
	got := serializeValue(custom{X: 7})
	want := fmt.Sprintf("%v", custom{X: 7})
	if got != want {
		t.Errorf("serializeValue(unknown) = %v, want %v", got, want)
	}
}

// ---------------------------------------------------------------------------
// isPrintable
// ---------------------------------------------------------------------------

func TestIsPrintable_Empty(t *testing.T) {
	if !isPrintable([]byte{}) {
		t.Error("isPrintable(empty) = false, want true")
	}
}

func TestIsPrintable_ASCII(t *testing.T) {
	if !isPrintable([]byte("Hello, World! 123")) {
		t.Error("isPrintable(ASCII) = false, want true")
	}
}

func TestIsPrintable_ControlChars(t *testing.T) {
	if isPrintable([]byte{0x01}) {
		t.Error("isPrintable(0x01) = true, want false")
	}
	if isPrintable([]byte{0x1F}) {
		t.Error("isPrintable(0x1F) = true, want false")
	}
}

func TestIsPrintable_WhitespaceAllowed(t *testing.T) {
	if !isPrintable([]byte{'\n', '\r', '\t'}) {
		t.Error("isPrintable(\\n\\r\\t) = false, want true")
	}
}

// ---------------------------------------------------------------------------
// truncate
// ---------------------------------------------------------------------------

func TestTruncate_ShortString(t *testing.T) {
	if got := truncate("hi", 10); got != "hi" {
		t.Errorf("truncate(\"hi\", 10) = %q, want \"hi\"", got)
	}
}

func TestTruncate_LongString(t *testing.T) {
	if got := truncate("hello world", 5); got != "hello" {
		t.Errorf("truncate(\"hello world\", 5) = %q, want \"hello\"", got)
	}
}

func TestTruncate_ExactLength(t *testing.T) {
	if got := truncate("exact", 5); got != "exact" {
		t.Errorf("truncate(\"exact\", 5) = %q, want \"exact\"", got)
	}
}

// ---------------------------------------------------------------------------
// roundFloat
// ---------------------------------------------------------------------------

func TestRoundFloat_TwoDecimals(t *testing.T) {
	got := roundFloat(3.14159, 2)
	if got != 3.14 {
		t.Errorf("roundFloat(3.14159, 2) = %f, want 3.14", got)
	}
}

func TestRoundFloat_ZeroDecimals(t *testing.T) {
	got := roundFloat(3.14159, 0)
	if got != 3.0 {
		t.Errorf("roundFloat(3.14159, 0) = %f, want 3.0", got)
	}
}

// ---------------------------------------------------------------------------
// isExplainSupported
// ---------------------------------------------------------------------------

func TestIsExplainSupported_Supported(t *testing.T) {
	supported := []models.DatabaseType{
		models.DatabaseTypeMySQL,
		models.DatabaseTypePostgreSQL,
		models.DatabaseTypeSQLite,
		models.DatabaseTypeClickHouse,
		models.DatabaseTypeMSSQL,
	}
	for _, dt := range supported {
		if !isExplainSupported(dt) {
			t.Errorf("isExplainSupported(%s) = false, want true", dt)
		}
	}
}

func TestIsExplainSupported_NotSupported(t *testing.T) {
	unsupported := []models.DatabaseType{
		models.DatabaseTypeElasticsearch,
		models.DatabaseTypeHive,
	}
	for _, dt := range unsupported {
		if isExplainSupported(dt) {
			t.Errorf("isExplainSupported(%s) = true, want false", dt)
		}
	}
}
