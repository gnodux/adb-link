package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gnodux/adb-link/internal/models"
	"gopkg.in/yaml.v3"
)

var envVarRegex = regexp.MustCompile(`\$\{(\w+)\}`)

// ReloadCallback is invoked after a successful hot-reload. The callback is
// expected to refresh any derived state (e.g. permission rules, cached
// connections). It runs synchronously on the reload goroutine.
type ReloadCallback func()

// configSnapshot is the immutable bundle of all configuration loaded from
// the YAML directory at a single point in time. The snapshot is replaced
// atomically on reload; readers receive a stable view for the duration of
// their request.
type configSnapshot struct {
	Datasources map[string]*models.DatasourceConfig
	Metadata    []*models.MetadataConfig
	Tools       map[string]*models.ToolConfig
	Toolsets    map[string]*models.ToolsetConfig
	AuthUsers   map[string]*models.AuthUser // keyed by api_key
	Permissions []*models.PermissionConfig
}

func newEmptySnapshot() *configSnapshot {
	return &configSnapshot{
		Datasources: make(map[string]*models.DatasourceConfig),
		Tools:       make(map[string]*models.ToolConfig),
		Toolsets:    make(map[string]*models.ToolsetConfig),
		AuthUsers:   make(map[string]*models.AuthUser),
	}
}

// ConfigService loads and manages all YAML-based configurations.
// Reads are lock-free against an atomically-published snapshot. Mutations
// (RegisterTool, UnregisterTool, hot reload) build a new snapshot and swap.
type ConfigService struct {
	configDir string
	snap      atomic.Pointer[configSnapshot]

	mu        sync.Mutex // serializes mutations + reload
	callbacks []ReloadCallback
}

// NewConfigService creates a new ConfigService and loads all configs.
func NewConfigService(settings *Settings) *ConfigService {
	cs := &ConfigService{configDir: settings.ConfigDir}
	cs.snap.Store(newEmptySnapshot())
	if err := cs.Reload(); err != nil {
		slog.Warn("initial config load failed", "error", err)
	}
	return cs
}

// ConfigDir returns the configuration directory path.
func (cs *ConfigService) ConfigDir() string { return cs.configDir }

// snapshot returns the current immutable snapshot for read access.
func (cs *ConfigService) snapshot() *configSnapshot { return cs.snap.Load() }

// AddReloadCallback registers a callback to be fired after each successful
// reload (initial load is not announced via callback).
func (cs *ConfigService) AddReloadCallback(cb ReloadCallback) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.callbacks = append(cs.callbacks, cb)
}

// ListDatasources returns public-facing info for all enabled datasources.
func (cs *ConfigService) ListDatasources() []models.DatasourceInfo {
	snap := cs.snapshot()
	var result []models.DatasourceInfo
	for _, cfg := range snap.Datasources {
		dialect := models.DialectInfoMap[cfg.Type]
		result = append(result, models.DatasourceInfo{
			Name:        cfg.Name,
			Type:        cfg.Type,
			Description: cfg.Description,
			Shadow:      cfg.Shadow,
			Dialect:     dialect,
		})
	}
	return result
}

// GetDatasource retrieves a datasource config by name.
func (cs *ConfigService) GetDatasource(name string) (*models.DatasourceConfig, error) {
	snap := cs.snapshot()
	cfg, ok := snap.Datasources[name]
	if !ok {
		return nil, fmt.Errorf("datasource config not found: %s", name)
	}
	return cfg, nil
}

// GetTool retrieves a tool config by name.
func (cs *ConfigService) GetTool(name string) (*models.ToolConfig, error) {
	snap := cs.snapshot()
	tool, ok := snap.Tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool, nil
}

// AllTools returns a snapshot slice of every loaded tool. The slice is owned
// by the caller; the underlying tool pointers are immutable.
func (cs *ConfigService) AllTools() []*models.ToolConfig {
	snap := cs.snapshot()
	out := make([]*models.ToolConfig, 0, len(snap.Tools))
	for _, t := range snap.Tools {
		out = append(out, t)
	}
	return out
}

// AllToolsets returns a snapshot slice of every loaded toolset.
func (cs *ConfigService) AllToolsets() []*models.ToolsetConfig {
	snap := cs.snapshot()
	out := make([]*models.ToolsetConfig, 0, len(snap.Toolsets))
	for _, t := range snap.Toolsets {
		out = append(out, t)
	}
	return out
}

// AllAuthUsers returns the map keyed by API key. The returned map is the
// snapshot's own backing map and MUST NOT be mutated.
func (cs *ConfigService) AllAuthUsers() map[string]*models.AuthUser {
	return cs.snapshot().AuthUsers
}

// AllPermissions returns the permission configs. Read-only.
func (cs *ConfigService) AllPermissions() []*models.PermissionConfig {
	return cs.snapshot().Permissions
}

// AllMetadata returns the metadata configs. Read-only.
func (cs *ConfigService) AllMetadata() []*models.MetadataConfig {
	return cs.snapshot().Metadata
}

// RegisterTool stores a new tool definition by swapping in a snapshot whose
// Tools map contains the addition.
func (cs *ConfigService) RegisterTool(tool *models.ToolConfig) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cur := cs.snapshot()
	next := cloneSnapshot(cur)
	next.Tools[tool.Name] = tool
	cs.snap.Store(next)
}

// UnregisterTool removes a tool definition. Returns the removed tool, or nil.
func (cs *ConfigService) UnregisterTool(name string) *models.ToolConfig {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cur := cs.snapshot()
	tool, ok := cur.Tools[name]
	if !ok {
		return nil
	}
	next := cloneSnapshot(cur)
	delete(next.Tools, name)
	cs.snap.Store(next)
	return tool
}

// PersistTool writes a tool config to a YAML file.
func (cs *ConfigService) PersistTool(tool *models.ToolConfig) (string, error) {
	filePath := filepath.Join(cs.configDir, fmt.Sprintf("tool-%s.yaml", tool.Name))
	data, err := yaml.Marshal(tool)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", err
	}
	return filePath, nil
}

// RemoveToolFile removes the persisted YAML file for a tool.
func (cs *ConfigService) RemoveToolFile(name string) bool {
	filePath := filepath.Join(cs.configDir, fmt.Sprintf("tool-%s.yaml", name))
	if err := os.Remove(filePath); err != nil {
		return false
	}
	return true
}

// RegisterDatasource stores a new datasource definition by swapping in a snapshot
// whose Datasources map contains the addition.
func (cs *ConfigService) RegisterDatasource(cfg *models.DatasourceConfig) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cur := cs.snapshot()
	next := cloneSnapshot(cur)
	next.Datasources[cfg.Name] = cfg
	cs.snap.Store(next)
}

// UnregisterDatasource removes a datasource definition. Returns the removed config, or nil.
func (cs *ConfigService) UnregisterDatasource(name string) *models.DatasourceConfig {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cur := cs.snapshot()
	cfg, ok := cur.Datasources[name]
	if !ok {
		return nil
	}
	next := cloneSnapshot(cur)
	delete(next.Datasources, name)
	cs.snap.Store(next)
	return cfg
}

// PersistDatasource writes a datasource config to a YAML file.
func (cs *ConfigService) PersistDatasource(cfg *models.DatasourceConfig) (string, error) {
	filePath := filepath.Join(cs.configDir, fmt.Sprintf("datasource-%s.yaml", cfg.Name))
	cfg.Kind = "datasource"
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", err
	}
	return filePath, nil
}

// RemoveDatasourceFile removes the persisted YAML file for a datasource.
func (cs *ConfigService) RemoveDatasourceFile(name string) bool {
	filePath := filepath.Join(cs.configDir, fmt.Sprintf("datasource-%s.yaml", name))
	if err := os.Remove(filePath); err != nil {
		return false
	}
	return true
}

// Reload re-reads every YAML file in the configured directory and atomically
// swaps the in-memory snapshot. Reload is safe to call concurrently with
// readers; readers either see the prior or new snapshot, never a torn view.
// Tools that were registered at runtime via RegisterTool but never persisted
// to disk will be dropped on reload — persistent tools survive because they
// are loaded back from disk.
func (cs *ConfigService) Reload() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	next := newEmptySnapshot()
	if err := loadAllInto(cs.configDir, next); err != nil {
		return err
	}
	cs.snap.Store(next)
	for _, cb := range cs.callbacks {
		cb()
	}
	return nil
}

func cloneSnapshot(src *configSnapshot) *configSnapshot {
	if src == nil {
		return newEmptySnapshot()
	}
	dst := &configSnapshot{
		Datasources: make(map[string]*models.DatasourceConfig, len(src.Datasources)),
		Metadata:    append([]*models.MetadataConfig(nil), src.Metadata...),
		Tools:       make(map[string]*models.ToolConfig, len(src.Tools)),
		Toolsets:    make(map[string]*models.ToolsetConfig, len(src.Toolsets)),
		AuthUsers:   make(map[string]*models.AuthUser, len(src.AuthUsers)),
		Permissions: append([]*models.PermissionConfig(nil), src.Permissions...),
	}
	for k, v := range src.Datasources {
		dst.Datasources[k] = v
	}
	for k, v := range src.Tools {
		dst.Tools[k] = v
	}
	for k, v := range src.Toolsets {
		dst.Toolsets[k] = v
	}
	for k, v := range src.AuthUsers {
		dst.AuthUsers[k] = v
	}
	return dst
}

// loadAllInto walks configDir and dispatches each YAML document into snap.
func loadAllInto(configDir string, snap *configSnapshot) error {
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return fmt.Errorf("read config directory %q: %w", configDir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		filePath := filepath.Join(configDir, entry.Name())
		if err := loadFileInto(filePath, snap); err != nil {
			slog.Warn("Failed to load config file", "file", filePath, "error", err)
		}
	}
	return nil
}

func loadFileInto(path string, snap *configSnapshot) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	// Interpolate environment variables: ${VAR_NAME}
	content := envVarRegex.ReplaceAllStringFunc(string(raw), func(match string) string {
		varName := match[2 : len(match)-1]
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		return match
	})

	// Split multi-document YAML
	decoder := yaml.NewDecoder(strings.NewReader(content))
	for {
		var doc map[string]any
		if err := decoder.Decode(&doc); err != nil {
			break
		}
		if doc == nil {
			continue
		}
		dispatchDocumentInto(doc, path, snap)
	}
	return nil
}

func dispatchDocumentInto(doc map[string]any, source string, snap *configSnapshot) {
	kind, _ := doc["kind"].(string)
	if kind == "" {
		kind = "datasource"
	}

	data, err := yaml.Marshal(doc)
	if err != nil {
		slog.Warn("Failed to marshal document", "kind", kind, "source", source, "error", err)
		return
	}

	switch kind {
	case "datasource":
		var cfg models.DatasourceConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			slog.Warn("Failed to parse datasource", "source", source, "error", err)
			return
		}
		if !cfg.IsEnabled() {
			slog.Info("Datasource disabled, skipping", "name", cfg.Name)
			return
		}
		snap.Datasources[cfg.Name] = &cfg

	case "metadata":
		var cfg models.MetadataConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			slog.Warn("Failed to parse metadata", "source", source, "error", err)
			return
		}
		if !cfg.IsEnabled() {
			slog.Info("Metadata config disabled, skipping", "datasource", cfg.Datasource)
			return
		}
		snap.Metadata = append(snap.Metadata, &cfg)

	case "tool":
		var cfg models.ToolConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			slog.Warn("Failed to parse tool", "source", source, "error", err)
			return
		}
		if !cfg.IsEnabled() {
			slog.Info("Tool disabled, skipping", "name", cfg.Name)
			return
		}
		snap.Tools[cfg.Name] = &cfg

	case "toolset":
		var cfg models.ToolsetConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			slog.Warn("Failed to parse toolset", "source", source, "error", err)
			return
		}
		if !cfg.IsEnabled() {
			slog.Info("Toolset disabled, skipping", "name", cfg.Name)
			return
		}
		snap.Toolsets[cfg.Name] = &cfg

	case "auth_api":
		var cfg models.AuthAPIConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			slog.Warn("Failed to parse auth_api", "source", source, "error", err)
			return
		}
		for i := range cfg.Users {
			user := &cfg.Users[i]
			if !user.IsEnabled() {
				slog.Info("Auth user disabled, skipping", "name", user.Name)
				continue
			}
			snap.AuthUsers[user.APIKey] = user
		}

	case "permission":
		var cfg models.PermissionConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			slog.Warn("Failed to parse permission", "source", source, "error", err)
			return
		}
		if !cfg.IsEnabled() {
			slog.Info("Permission config disabled, skipping")
			return
		}
		snap.Permissions = append(snap.Permissions, &cfg)

	default:
		slog.Warn("Unknown kind in config", "kind", kind, "source", source)
	}
}
