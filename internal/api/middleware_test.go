package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gnodux/adb-link/internal/config"
	"github.com/gnodux/adb-link/internal/models"
)

// newTestConfigService creates a ConfigService backed by a temp directory.
// If authYAML is non-empty it is written as auth.yaml before loading.
func newTestConfigService(t *testing.T, authYAML string) *config.ConfigService {
	t.Helper()
	dir := t.TempDir()
	if authYAML != "" {
		if err := os.WriteFile(filepath.Join(dir, "auth.yaml"), []byte(authYAML), 0644); err != nil {
			t.Fatalf("write auth.yaml: %v", err)
		}
	}
	return config.NewConfigService(&config.Settings{ConfigDir: dir})
}

const testAuthYAML = `kind: users
users:
  - name: alice
    api_key: key-alice
  - name: bob
    api_key: key-bob
`

// okHandler is a simple handler that writes 200 OK.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestBearerAuth_NoUsers_PassThrough(t *testing.T) {
	cs := newTestConfigService(t, "") // no auth.yaml
	handler := BearerAuth(cs)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/query", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	// No user should be injected in context.
	if u := UserFromRequest(req); u != nil {
		t.Fatalf("expected no user in context, got %+v", u)
	}
}

func TestBearerAuth_SkipPath_Health(t *testing.T) {
	cs := newTestConfigService(t, testAuthYAML)
	handler := BearerAuth(cs)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/health, got %d", rec.Code)
	}
}

func TestBearerAuth_SkipPath_Docs(t *testing.T) {
	cs := newTestConfigService(t, testAuthYAML)
	handler := BearerAuth(cs)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /docs, got %d", rec.Code)
	}
}

func TestBearerAuth_SkipPath_SwaggerUI(t *testing.T) {
	cs := newTestConfigService(t, testAuthYAML)
	handler := BearerAuth(cs)(okHandler)

	paths := []string{"/api/swagger", "/api/swagger/", "/api/swagger/doc.json"}
	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, p, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200 for %s, got %d", p, rec.Code)
			}
		})
	}
}

func TestBearerAuth_MissingHeader_401(t *testing.T) {
	cs := newTestConfigService(t, testAuthYAML)
	handler := BearerAuth(cs)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/query", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	var body models.APIResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Success {
		t.Fatal("expected success=false in error response")
	}
}

func TestBearerAuth_InvalidScheme_401(t *testing.T) {
	cs := newTestConfigService(t, testAuthYAML)
	handler := BearerAuth(cs)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/query", nil)
	req.Header.Set("Authorization", "Basic xxx")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestBearerAuth_InvalidKey_403(t *testing.T) {
	cs := newTestConfigService(t, testAuthYAML)
	handler := BearerAuth(cs)(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/query", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestBearerAuth_ValidKey_InjectsUser(t *testing.T) {
	cs := newTestConfigService(t, testAuthYAML)

	var capturedUser *models.AuthUser
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser = UserFromRequest(r)
		w.WriteHeader(http.StatusOK)
	})
	handler := BearerAuth(cs)(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/query", nil)
	req.Header.Set("Authorization", "Bearer key-alice")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if capturedUser == nil {
		t.Fatal("expected user in context, got nil")
	}
	if capturedUser.Name != "alice" {
		t.Fatalf("expected user name 'alice', got %q", capturedUser.Name)
	}
}

func TestCORS_SetsHeaders(t *testing.T) {
	handler := CORS(okHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/query", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected Access-Control-Allow-Origin: *, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("expected Access-Control-Allow-Methods header to be set")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("expected Access-Control-Allow-Headers header to be set")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCORS_OptionsRequest_204(t *testing.T) {
	handler := CORS(okHandler)

	req := httptest.NewRequest(http.MethodOptions, "/api/query", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected CORS headers on OPTIONS response, got origin %q", got)
	}
}
