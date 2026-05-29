package services

import (
	"log/slog"
	"time"

	"github.com/gnodux/adb-link/internal/config"
)

// Container groups all services for dependency injection.
type Container struct {
	Settings          *config.Settings
	ConfigService     *config.ConfigService
	ConnectionService *ConnectionService
	MetadataService   *MetadataService
	PermissionService *PermissionService
	SchemaService     *SchemaService
	QueryService      *QueryService
	AsyncQueryService *AsyncQueryService

	watcher *config.Watcher
}

// NewContainer wires all services together.
func NewContainer(settings *config.Settings) *Container {
	cfgSvc := config.NewConfigService(settings)
	connSvc := NewConnectionService(cfgSvc)

	metaSvc := NewMetadataService(cfgSvc.AllMetadata())
	permSvc := NewPermissionService(cfgSvc.AllAuthUsers(), cfgSvc.AllPermissions())
	schemaSvc := NewSchemaService(cfgSvc, connSvc, metaSvc, permSvc)
	querySvc := NewQueryService(connSvc, cfgSvc, permSvc)
	asyncSvc := NewAsyncQueryService(querySvc, cfgSvc, settings.AsyncQueryTTL)

	SetupAuditLogging(settings.LogDir)

	c := &Container{
		Settings:          settings,
		ConfigService:     cfgSvc,
		ConnectionService: connSvc,
		MetadataService:   metaSvc,
		PermissionService: permSvc,
		SchemaService:     schemaSvc,
		QueryService:      querySvc,
		AsyncQueryService: asyncSvc,
	}

	// Hot-reload wiring: when YAML configs change, refresh derived services
	// and invalidate cached connections so DSN/credential changes take effect.
	cfgSvc.AddReloadCallback(func() {
		permSvc.UpdateState(cfgSvc.AllAuthUsers(), cfgSvc.AllPermissions())
		metaSvc.UpdateState(cfgSvc.AllMetadata())
		connSvc.InvalidateAll()
		slog.Info("config reload applied: permissions/metadata refreshed, connections invalidated")
	})

	return c
}

// Start starts background services.
func (c *Container) Start() {
	c.AsyncQueryService.Start()
	c.ConnectionService.StartHealthCheck(5*time.Minute, 3)

	if w, err := config.NewWatcher(c.ConfigService, 500*time.Millisecond); err != nil {
		slog.Warn("config hot-reload watcher disabled", "error", err)
	} else {
		c.watcher = w
		w.Start()
		slog.Info("config hot-reload watcher started", "dir", c.ConfigService.ConfigDir())
	}
}

// Stop shuts down background services and releases resources.
func (c *Container) Stop() {
	if c.watcher != nil {
		c.watcher.Stop()
		c.watcher = nil
	}
	c.AsyncQueryService.Stop()
	_ = c.ConnectionService.DisposeAll()
}
