package services

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gnodux/adb-link/internal/config"
	"github.com/gnodux/adb-link/internal/dialects"
	"github.com/gnodux/adb-link/internal/models"
)

// ConnectionService manages cached database connections.
type ConnectionService struct {
	mu            sync.Mutex
	configService *config.ConfigService
	sqlConns      map[string]*sql.DB       // key: datasource::database
	nonSQLClients map[string]NonSQLClient  // key: datasource

	healthCancel context.CancelFunc
	healthDone   chan struct{}
}

// NewConnectionService creates a new ConnectionService.
func NewConnectionService(configService *config.ConfigService) *ConnectionService {
	return &ConnectionService{
		configService: configService,
		sqlConns:      make(map[string]*sql.DB),
		nonSQLClients: make(map[string]NonSQLClient),
	}
}

// GetSQLDB returns a cached *sql.DB for the given datasource and database.
// On a fresh open, a fast Ping (5s) is performed. If the ping fails, the
// connection is closed and the error is returned to the caller — no broken
// connection is cached.
func (cs *ConnectionService) GetSQLDB(datasourceName, database string) (*sql.DB, *models.DatasourceConfig, error) {
	cfg, err := cs.configService.GetDatasource(datasourceName)
	if err != nil {
		return nil, nil, err
	}
	if IsNonSQLType(cfg.Type) {
		return nil, cfg, fmt.Errorf("datasource '%s' is %s; use GetNonSQLClient", datasourceName, cfg.Type)
	}

	db := database
	if db == "" {
		db = cfg.Connection.DefaultDatabase
	}
	cacheKey := fmt.Sprintf("%s::%s", datasourceName, db)

	cs.mu.Lock()
	defer cs.mu.Unlock()

	if conn, ok := cs.sqlConns[cacheKey]; ok {
		return conn, cfg, nil
	}

	dialect, err := dialects.GetDialect(cfg.Type)
	if err != nil {
		return nil, cfg, err
	}

	dsn := dialect.BuildDSN(cfg, db)
	driverName := driverNameFor(cfg.Type)
	conn, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, cfg, err
	}

	// Configure pool
	poolSize := 5
	if v, ok := cfg.Options["pool_size"].(int); ok {
		poolSize = v
	}
	maxOverflow := 10
	if v, ok := cfg.Options["max_overflow"].(int); ok {
		maxOverflow = v
	}
	poolRecycle := 3600
	if v, ok := cfg.Options["pool_recycle"].(int); ok {
		poolRecycle = v
	}
	if cfg.Type != models.DatabaseTypeSQLite {
		conn.SetMaxOpenConns(poolSize + maxOverflow)
		conn.SetMaxIdleConns(poolSize)
		conn.SetConnMaxLifetime(time.Duration(poolRecycle) * time.Second)
	}

	// Fail-fast Ping. Use a short timeout so unreachable hosts return promptly.
	pingTimeout := 5 * time.Second
	if v, ok := cfg.Options["connect_timeout"]; ok {
		if seconds, ok := toFloat(v); ok && seconds > 0 {
			pingTimeout = time.Duration(seconds * float64(time.Second))
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		return nil, cfg, fmt.Errorf("ping datasource %q failed: %w", datasourceName, err)
	}

	cs.sqlConns[cacheKey] = conn
	return conn, cfg, nil
}

// newNonSQLClient creates a NonSQLClient for the given datasource config.
func newNonSQLClient(cfg *models.DatasourceConfig) (NonSQLClient, error) {
	switch cfg.Type {
	case models.DatabaseTypeElasticsearch:
		return NewESClient(cfg), nil
	case models.DatabaseTypeRedis:
		return NewRedisClient(cfg)
	case models.DatabaseTypeMongoDB:
		return NewMongoClient(cfg)
	case models.DatabaseTypeMilvus:
		return NewMilvusClient(cfg)
	default:
		return nil, fmt.Errorf("unsupported non-SQL type: %s", cfg.Type)
	}
}

// GetNonSQLClient returns a cached NonSQLClient for the given datasource.
func (cs *ConnectionService) GetNonSQLClient(datasourceName string) (NonSQLClient, *models.DatasourceConfig, error) {
	cfg, err := cs.configService.GetDatasource(datasourceName)
	if err != nil {
		return nil, nil, err
	}
	if !IsNonSQLType(cfg.Type) {
		return nil, cfg, fmt.Errorf("datasource '%s' is not a non-SQL type", datasourceName)
	}

	cs.mu.Lock()
	defer cs.mu.Unlock()

	if client, ok := cs.nonSQLClients[datasourceName]; ok {
		return client, cfg, nil
	}

	client, err := newNonSQLClient(cfg)
	if err != nil {
		return nil, cfg, err
	}
	cs.nonSQLClients[datasourceName] = client
	return client, cfg, nil
}

// GetESClient returns a cached ES client. Kept for backward compatibility;
// prefer GetNonSQLClient for new code.
func (cs *ConnectionService) GetESClient(datasourceName string) (*ESClient, *models.DatasourceConfig, error) {
	client, cfg, err := cs.GetNonSQLClient(datasourceName)
	if err != nil {
		return nil, cfg, err
	}
	es, ok := client.(*ESClient)
	if !ok {
		return nil, cfg, fmt.Errorf("datasource '%s' is not Elasticsearch", datasourceName)
	}
	return es, cfg, nil
}

// Invalidate removes all cached connections for the given datasource and
// closes them. Safe to call when the datasource has no cached connections.
func (cs *ConnectionService) Invalidate(datasourceName string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	prefix := datasourceName + "::"
	for key, conn := range cs.sqlConns {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			_ = conn.Close()
			delete(cs.sqlConns, key)
		}
	}
	if client, ok := cs.nonSQLClients[datasourceName]; ok {
		_ = client.Close()
		delete(cs.nonSQLClients, datasourceName)
	}
}

// InvalidateAll closes and clears every cached connection. Used during
// hot-reload when configuration may have changed substantially.
func (cs *ConnectionService) InvalidateAll() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for key, conn := range cs.sqlConns {
		_ = conn.Close()
		delete(cs.sqlConns, key)
	}
	for key, client := range cs.nonSQLClients {
		_ = client.Close()
		delete(cs.nonSQLClients, key)
	}
}

// StartHealthCheck launches a background goroutine that pings cached SQL
// connections at the given interval. Connections failing the configured
// number of consecutive pings are evicted and closed.
func (cs *ConnectionService) StartHealthCheck(interval time.Duration, maxFailures int) {
	if interval <= 0 {
		return
	}
	if maxFailures <= 0 {
		maxFailures = 3
	}
	cs.mu.Lock()
	if cs.healthCancel != nil {
		cs.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	cs.healthCancel = cancel
	cs.healthDone = make(chan struct{})
	cs.mu.Unlock()

	go cs.runHealthCheck(ctx, interval, maxFailures)
}

// StopHealthCheck terminates the background health-check goroutine.
func (cs *ConnectionService) StopHealthCheck() {
	cs.mu.Lock()
	cancel := cs.healthCancel
	done := cs.healthDone
	cs.healthCancel = nil
	cs.healthDone = nil
	cs.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (cs *ConnectionService) runHealthCheck(ctx context.Context, interval time.Duration, maxFailures int) {
	defer func() {
		cs.mu.Lock()
		if cs.healthDone != nil {
			close(cs.healthDone)
		}
		cs.mu.Unlock()
	}()

	failures := make(map[string]int)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		// Snapshot to avoid holding the lock during ping.
		cs.mu.Lock()
		snapshot := make(map[string]*sql.DB, len(cs.sqlConns))
		for k, v := range cs.sqlConns {
			snapshot[k] = v
		}
		cs.mu.Unlock()

		for key, conn := range snapshot {
			pctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err := conn.PingContext(pctx)
			cancel()
			if err == nil {
				delete(failures, key)
				continue
			}
			failures[key]++
			slog.Warn("connection health check failed", "key", key, "failures", failures[key], "error", err)
			if failures[key] >= maxFailures {
				cs.mu.Lock()
				if cur, ok := cs.sqlConns[key]; ok && cur == conn {
					_ = cur.Close()
					delete(cs.sqlConns, key)
					slog.Warn("connection evicted after repeated ping failures", "key", key)
				}
				cs.mu.Unlock()
				delete(failures, key)
			}
		}
	}
}

// DisposeAll closes all cached connections.
func (cs *ConnectionService) DisposeAll() error {
	cs.StopHealthCheck()
	cs.mu.Lock()
	defer cs.mu.Unlock()

	for _, conn := range cs.sqlConns {
		_ = conn.Close()
	}
	cs.sqlConns = make(map[string]*sql.DB)

	for _, client := range cs.nonSQLClients {
		_ = client.Close()
	}
	cs.nonSQLClients = make(map[string]NonSQLClient)
	return nil
}

func driverNameFor(t models.DatabaseType) string {
	switch t {
	case models.DatabaseTypeMySQL, models.DatabaseTypeTiDB:
		return "mysql"
	case models.DatabaseTypePostgreSQL, models.DatabaseTypeGaussDB:
		return "postgres"
	case models.DatabaseTypeSQLite:
		return "sqlite"
	case models.DatabaseTypeClickHouse:
		return "clickhouse"
	case models.DatabaseTypeMSSQL:
		return "sqlserver"
	case models.DatabaseTypeHive:
		return "hive"
	case models.DatabaseTypeOracle:
		return "oracle"
	default:
		return string(t)
	}
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	case float32:
		return float64(x), true
	}
	return 0, false
}
