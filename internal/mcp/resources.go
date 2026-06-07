package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gnodux/adb-link/internal/models"
	"github.com/gnodux/adb-link/internal/services"
)

// datasourceURIKind identifies the type of a datasource:// URI.
type datasourceURIKind string

const (
	uriList      datasourceURIKind = "list"
	uriDatasource datasourceURIKind = "datasource"
	uriDatabases datasourceURIKind = "databases"
	uriSchema    datasourceURIKind = "schema"
	uriTable     datasourceURIKind = "table"
	uriView      datasourceURIKind = "view"
)

// parsedURI holds the result of parsing a datasource:// URI.
type parsedURI struct {
	kind     datasourceURIKind
	segments []string // captured path segments
}

// parseDatasourceURI parses a datasource:// URI into its kind and segments.
//
//	datasource:///                              → list
//	datasource:///name                          → datasource, [name]
//	datasource:///name/db/databases             → databases, [name, db]
//	datasource:///name/db/schema                → schema, [name, db]
//	datasource:///name/db/tables/table          → table, [name, db, table]
//	datasource:///name/db/views/view            → view, [name, db, view]
func parseDatasourceURI(uri string) (parsedURI, error) {
	path := strings.TrimPrefix(uri, "datasource:///")
	parts := splitPath(path)

	switch {
	case len(parts) == 0 || (len(parts) == 1 && parts[0] == ""):
		return parsedURI{kind: uriList}, nil
	case len(parts) == 1:
		return parsedURI{kind: uriDatasource, segments: parts}, nil
	case len(parts) == 3 && parts[2] == "databases":
		return parsedURI{kind: uriDatabases, segments: parts[:2]}, nil
	case len(parts) == 3 && parts[2] == "schema":
		return parsedURI{kind: uriSchema, segments: parts[:2]}, nil
	case len(parts) == 4 && parts[2] == "tables":
		return parsedURI{kind: uriTable, segments: []string{parts[0], parts[1], parts[3]}}, nil
	case len(parts) == 4 && parts[2] == "views":
		return parsedURI{kind: uriView, segments: []string{parts[0], parts[1], parts[3]}}, nil
	default:
		return parsedURI{}, fmt.Errorf("invalid datasource URI: %s", uri)
	}
}

// RegisterCoreResources registers MCP resources and resource templates backed
// by the service container.
func RegisterCoreResources(srv *Server, c *services.Container) {
	// Static resource: datasource listing.
	srv.RegisterResource(Resource{
		URI:         "datasource:///",
		Name:        "Datasources",
		Description: "All configured datasources with name, type, description, and dialect info.",
		MimeType:    "application/json",
	}, func(ctx context.Context, uri string) ([]ResourceContent, error) {
		user := userFromCtx(ctx)
		all := c.ConfigService.ListDatasources()
		filtered := make([]models.DatasourceInfo, 0, len(all))
		for _, ds := range all {
			if c.PermissionService.CheckDatasource(user, ds.Name) {
				filtered = append(filtered, ds)
			}
		}
		return resourceJSON(uri, filtered)
	})

	// Template: datasource detail.
	srv.RegisterResourceTemplate(ResourceTemplate{
		URITemplate: "datasource:///{name}",
		Name:        "Datasource Detail",
		Description: "Configuration and connection details for a specific datasource (password masked).",
		MimeType:    "application/json",
	}, func(ctx context.Context, uri string) ([]ResourceContent, error) {
		parsed, err := parseDatasourceURI(uri)
		if err != nil {
			return nil, err
		}
		cfg, err := c.ConfigService.GetDatasource(parsed.segments[0])
		if err != nil {
			return nil, err
		}
		// Mask password.
		masked := *cfg
		conn := masked.Connection
		if conn.Password != "" {
			conn.Password = "***"
		}
		masked.Connection = conn
		return resourceJSON(uri, masked)
	})

	// Template: databases listing.
	srv.RegisterResourceTemplate(ResourceTemplate{
		URITemplate: "datasource:///{name}/{db}/databases",
		Name:        "Databases",
		Description: "List all databases in a datasource.",
		MimeType:    "application/json",
	}, func(ctx context.Context, uri string) ([]ResourceContent, error) {
		parsed, err := parseDatasourceURI(uri)
		if err != nil {
			return nil, err
		}
		dbs, err := c.SchemaService.GetDatabases(ctx, parsed.segments[0], userFromCtx(ctx))
		if err != nil {
			return nil, err
		}
		return resourceJSON(uri, dbs)
	})

	// Template: full schema.
	srv.RegisterResourceTemplate(ResourceTemplate{
		URITemplate: "datasource:///{name}/{db}/schema",
		Name:        "Database Schema",
		Description: "Complete schema (tables and views) for a database.",
		MimeType:    "application/json",
	}, func(ctx context.Context, uri string) ([]ResourceContent, error) {
		parsed, err := parseDatasourceURI(uri)
		if err != nil {
			return nil, err
		}
		schema, err := c.SchemaService.GetSchema(ctx, parsed.segments[0], parsed.segments[1], userFromCtx(ctx))
		if err != nil {
			return nil, err
		}
		return resourceJSON(uri, schema)
	})

	// Template: table info.
	srv.RegisterResourceTemplate(ResourceTemplate{
		URITemplate: "datasource:///{name}/{db}/tables/{table}",
		Name:        "Table Schema",
		Description: "Column details for a specific table.",
		MimeType:    "application/json",
	}, func(ctx context.Context, uri string) ([]ResourceContent, error) {
		parsed, err := parseDatasourceURI(uri)
		if err != nil {
			return nil, err
		}
		ti, err := c.SchemaService.GetTableInfo(ctx, parsed.segments[0], parsed.segments[1], parsed.segments[2], userFromCtx(ctx))
		if err != nil {
			return nil, err
		}
		return resourceJSON(uri, ti)
	})

	// Template: view info.
	srv.RegisterResourceTemplate(ResourceTemplate{
		URITemplate: "datasource:///{name}/{db}/views/{view}",
		Name:        "View Schema",
		Description: "Column details for a specific view.",
		MimeType:    "application/json",
	}, func(ctx context.Context, uri string) ([]ResourceContent, error) {
		parsed, err := parseDatasourceURI(uri)
		if err != nil {
			return nil, err
		}
		vi, err := c.SchemaService.GetViewInfo(ctx, parsed.segments[0], parsed.segments[1], parsed.segments[2], userFromCtx(ctx))
		if err != nil {
			return nil, err
		}
		return resourceJSON(uri, vi)
	})
}

// resourceJSON marshals v into a single-element ResourceContent slice.
func resourceJSON(uri string, v any) ([]ResourceContent, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return []ResourceContent{{
		URI:      uri,
		MimeType: "application/json",
		Text:     string(b),
	}}, nil
}
