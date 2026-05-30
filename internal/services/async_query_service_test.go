package services

import (
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// GetStatus / GetResult / Cancel with unknown ID
// ---------------------------------------------------------------------------

func newEmptyAsyncQueryService() *AsyncQueryService {
	return &AsyncQueryService{
		ttl:     time.Hour,
		queries: make(map[string]*asyncQueryEntry),
	}
}

func TestGetStatus_UnknownID(t *testing.T) {
	svc := newEmptyAsyncQueryService()
	_, err := svc.GetStatus("does-not-exist")
	if err == nil {
		t.Fatal("GetStatus(unknown) returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "query not found") {
		t.Errorf("GetStatus(unknown) error = %q, want containing \"query not found\"", err.Error())
	}
}

func TestGetResult_UnknownID(t *testing.T) {
	svc := newEmptyAsyncQueryService()
	_, err := svc.GetResult("does-not-exist")
	if err == nil {
		t.Fatal("GetResult(unknown) returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "query not found") {
		t.Errorf("GetResult(unknown) error = %q, want containing \"query not found\"", err.Error())
	}
}

func TestCancel_UnknownID(t *testing.T) {
	svc := newEmptyAsyncQueryService()
	err := svc.Cancel("does-not-exist")
	if err == nil {
		t.Fatal("Cancel(unknown) returned nil error, want error")
	}
	if !strings.Contains(err.Error(), "query not found") {
		t.Errorf("Cancel(unknown) error = %q, want containing \"query not found\"", err.Error())
	}
}

// ---------------------------------------------------------------------------
// NewAsyncQueryService TTL defaults
// ---------------------------------------------------------------------------

func TestNewAsyncQueryService_DefaultTTL(t *testing.T) {
	svc := NewAsyncQueryService(nil, nil, 0)
	if svc.ttl != 3600*time.Second {
		t.Errorf("ttl = %v, want %v", svc.ttl, 3600*time.Second)
	}
}

func TestNewAsyncQueryService_CustomTTL(t *testing.T) {
	svc := NewAsyncQueryService(nil, nil, 120)
	if svc.ttl != 120*time.Second {
		t.Errorf("ttl = %v, want %v", svc.ttl, 120*time.Second)
	}
}
