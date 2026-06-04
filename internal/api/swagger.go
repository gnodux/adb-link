package api

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed openapi.yaml
var openapiSpecYAML string

// openAPISpecJSON converts the embedded YAML spec to JSON.
// The result is cached after the first call.
func openAPISpecJSON() ([]byte, error) {
	var node interface{}
	if err := yaml.Unmarshal([]byte(openapiSpecYAML), &node); err != nil {
		return nil, err
	}
	node = normalizeYAMLNode(node)
	return json.MarshalIndent(node, "", "  ")
}

// normalizeYAMLNode recursively converts map[interface{}]interface{} (produced
// by yaml.Unmarshal) into map[string]interface{} (required by json.Marshal).
func normalizeYAMLNode(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, v := range val {
			out[k] = normalizeYAMLNode(v)
		}
		return out
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, v := range val {
			out[k.(string)] = normalizeYAMLNode(v)
		}
		return out
	case []interface{}:
		for i, item := range val {
			val[i] = normalizeYAMLNode(item)
		}
		return val
	default:
		return v
	}
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>ADB-Link API Documentation</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; padding: 0; }
    #swagger-ui { max-width: 1200px; margin: 0 auto; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/api/swagger/doc.json",
      dom_id: "#swagger-ui",
      deepLinking: true,
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIBundle.SwaggerUIStandalonePreset
      ],
      layout: "BaseLayout"
    });
  </script>
</body>
</html>`

// registerSwaggerRoutes adds Swagger UI and spec endpoints to the mux.
func registerSwaggerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/api/swagger/", http.StatusMovedPermanently)
	})

	mux.HandleFunc("GET /api/swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
		data, err := openAPISpecJSON()
		if err != nil {
			http.Error(w, "failed to convert spec: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})

	mux.HandleFunc("GET /api/swagger/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/doc.json") {
			data, err := openAPISpecJSON()
			if err != nil {
				http.Error(w, "failed to convert spec: "+err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Cache-Control", "public, max-age=3600")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(swaggerUIHTML))
	})
}
