package services

import (
	"testing"

	"github.com/gnodux/adb-link/internal/config"
	"github.com/gnodux/adb-link/internal/models"
)

// ---------------------------------------------------------------------------
// driverNameFor
// ---------------------------------------------------------------------------

func TestDriverNameFor(t *testing.T) {
	tests := []struct {
		dbType models.DatabaseType
		want   string
	}{
		{models.DatabaseTypeMySQL, "mysql"},
		{models.DatabaseTypePostgreSQL, "postgres"},
		{models.DatabaseTypeSQLite, "sqlite"},
		{models.DatabaseTypeClickHouse, "clickhouse"},
		{models.DatabaseTypeMSSQL, "sqlserver"},
		{models.DatabaseTypeHive, "hive"},
		{models.DatabaseType("unknown_db"), "unknown_db"},
	}
	for _, tc := range tests {
		t.Run(string(tc.dbType), func(t *testing.T) {
			got := driverNameFor(tc.dbType)
			if got != tc.want {
				t.Errorf("driverNameFor(%s) = %q, want %q", tc.dbType, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// toFloat
// ---------------------------------------------------------------------------

func TestToFloat_Int(t *testing.T) {
	v, ok := toFloat(42)
	if !ok || v != 42.0 {
		t.Errorf("toFloat(42) = (%f, %v), want (42, true)", v, ok)
	}
}

func TestToFloat_Int64(t *testing.T) {
	v, ok := toFloat(int64(100))
	if !ok || v != 100.0 {
		t.Errorf("toFloat(int64(100)) = (%f, %v), want (100, true)", v, ok)
	}
}

func TestToFloat_Float64(t *testing.T) {
	v, ok := toFloat(3.14)
	if !ok || v != 3.14 {
		t.Errorf("toFloat(3.14) = (%f, %v), want (3.14, true)", v, ok)
	}
}

func TestToFloat_Float32(t *testing.T) {
	v, ok := toFloat(float32(2.5))
	if !ok || v != 2.5 {
		t.Errorf("toFloat(float32(2.5)) = (%f, %v), want (2.5, true)", v, ok)
	}
}

func TestToFloat_String(t *testing.T) {
	_, ok := toFloat("nope")
	if ok {
		t.Error("toFloat(\"nope\") returned ok=true, want false")
	}
}

func TestToFloat_Nil(t *testing.T) {
	_, ok := toFloat(nil)
	if ok {
		t.Error("toFloat(nil) returned ok=true, want false")
	}
}

// ---------------------------------------------------------------------------
// Invalidate / InvalidateAll / DisposeAll on empty service
// ---------------------------------------------------------------------------

func newEmptyConnectionService() *ConnectionService {
	return NewConnectionService(&config.ConfigService{})
}

func TestInvalidate_EmptyService(t *testing.T) {
	cs := newEmptyConnectionService()
	// Must not panic.
	cs.Invalidate("nonexistent")
}

func TestInvalidateAll_EmptyService(t *testing.T) {
	cs := newEmptyConnectionService()
	// Must not panic.
	cs.InvalidateAll()
}

func TestDisposeAll_EmptyService(t *testing.T) {
	cs := newEmptyConnectionService()
	err := cs.DisposeAll()
	if err != nil {
		t.Errorf("DisposeAll() on empty service returned error: %v", err)
	}
}
