package api

import (
	"net/http"
	"strings"

	"github.com/gnodux/adb-link/internal/config"
)

// skipAuthPaths are paths exempt from authentication.
var skipAuthPaths = map[string]bool{
	"/api/health":   true,
	"/docs":         true,
	"/openapi.json": true,
	"/redoc":        true,
	"/favicon.ico":  true,
}

// BearerAuth returns middleware enforcing Bearer-token authentication.
// If no auth users are configured, requests pass through with no user context.
func BearerAuth(cs *config.ConfigService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			users := cs.AllAuthUsers()
			if len(users) == 0 || skipAuthPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				WriteErrorStatus(w, http.StatusUnauthorized,
					"Missing or invalid Authorization header. Expected: Bearer <api_key>")
				return
			}
			apiKey := strings.TrimSpace(authHeader[len("Bearer "):])
			user, ok := users[apiKey]
			if !ok {
				WriteErrorStatus(w, http.StatusForbidden, "Invalid API key")
				return
			}
			ctx := WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CORS adds permissive CORS headers (suitable for local dev / internal services).
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
