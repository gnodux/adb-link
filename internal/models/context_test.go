package models

import (
	"context"
	"testing"
)

func TestWithAuthUser_RoundTrip(t *testing.T) {
	user := &AuthUser{Name: "alice", APIKey: "key123"}
	ctx := WithAuthUser(context.Background(), user)
	got := AuthUserFromContext(ctx)
	if got == nil {
		t.Fatal("expected user in context, got nil")
	}
	if got.Name != "alice" {
		t.Errorf("Name = %q, want %q", got.Name, "alice")
	}
	if got.APIKey != "key123" {
		t.Errorf("APIKey = %q, want %q", got.APIKey, "key123")
	}
}

func TestAuthUserFromContext_Empty(t *testing.T) {
	got := AuthUserFromContext(context.Background())
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestAuthUserNameFromContext_WithName(t *testing.T) {
	ctx := WithAuthUser(context.Background(), &AuthUser{Name: "bob"})
	if name := AuthUserNameFromContext(ctx); name != "bob" {
		t.Errorf("got %q, want %q", name, "bob")
	}
}

func TestAuthUserNameFromContext_NilUser(t *testing.T) {
	if name := AuthUserNameFromContext(context.Background()); name != "" {
		t.Errorf("got %q, want empty", name)
	}
}

func TestAuthUserNameFromContext_EmptyName(t *testing.T) {
	ctx := WithAuthUser(context.Background(), &AuthUser{Name: ""})
	if name := AuthUserNameFromContext(ctx); name != "" {
		t.Errorf("got %q, want empty", name)
	}
}
