package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gnodux/adb-link/internal/models"
	"github.com/redis/go-redis/v9"
)

// RedisClient implements NonSQLClient for Redis.
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient creates a new Redis client.
func NewRedisClient(cfg *models.DatasourceConfig) (*RedisClient, error) {
	conn := cfg.Connection
	port := conn.Port
	if port == 0 {
		port = 6379
	}
	db := 0
	if v, ok := cfg.Options["db"].(int); ok {
		db = v
	}
	poolSize := 5
	if v, ok := cfg.Options["pool_size"].(int); ok {
		poolSize = v
	}

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", conn.Host, port),
		Username:     conn.Username,
		Password:     conn.Password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		PoolSize:     poolSize,
	})
	return &RedisClient{client: client}, nil
}

func (c *RedisClient) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *RedisClient) Close() error {
	return c.client.Close()
}

func (c *RedisClient) GetDatabases(ctx context.Context) ([]models.ObjectName, error) {
	result, err := c.client.ConfigGet(ctx, "databases").Result()
	if err != nil {
		return []models.ObjectName{{Name: "db0"}}, nil
	}
	count := 16
	if v, ok := result["databases"]; ok {
		fmt.Sscanf(v, "%d", &count)
	}
	names := make([]models.ObjectName, count)
	for i := 0; i < count; i++ {
		names[i] = models.ObjectName{Name: fmt.Sprintf("db%d", i)}
	}
	return names, nil
}

func (c *RedisClient) GetTableNames(ctx context.Context, database string) ([]models.ObjectName, error) {
	prefixCounts := map[string]int{}
	var cursor uint64
	iterations := 0
	maxIterations := 100

	for iterations < maxIterations {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, "*", 100).Result()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			prefix := extractKeyPrefix(key)
			prefixCounts[prefix]++
		}
		cursor = nextCursor
		iterations++
		if cursor == 0 {
			break
		}
	}

	type prefixCount struct {
		prefix string
		count  int
	}
	var sorted []prefixCount
	for p, c := range prefixCounts {
		sorted = append(sorted, prefixCount{p, c})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	limit := 50
	if len(sorted) < limit {
		limit = len(sorted)
	}
	result := make([]models.ObjectName, limit)
	for i := 0; i < limit; i++ {
		result[i] = models.ObjectName{
			Name:    sorted[i].prefix,
			Comment: fmt.Sprintf("%d keys", sorted[i].count),
		}
	}
	return result, nil
}

func (c *RedisClient) GetTableInfo(ctx context.Context, database, table string) (*models.TableInfo, error) {
	pattern := table
	if !strings.Contains(pattern, "*") {
		pattern = table + "*"
	}

	var columns []models.ColumnInfo
	fieldTypes := map[string]string{}
	var cursor uint64
	sampled := 0

	for sampled < 20 {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 10).Result()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			if sampled >= 20 {
				break
			}
			typeResult, err := c.client.Type(ctx, key).Result()
			if err != nil {
				continue
			}
			sampled++
			switch typeResult {
			case "hash":
				fields, err := c.client.HKeys(ctx, key).Result()
				if err == nil {
					for _, f := range fields {
						if _, exists := fieldTypes[f]; !exists {
							fieldTypes[f] = "hash_field"
						}
					}
				}
			case "list":
				if _, exists := fieldTypes["_list"]; !exists {
					fieldTypes["_list"] = "list"
				}
			case "set":
				if _, exists := fieldTypes["_set"]; !exists {
					fieldTypes["_set"] = "set"
				}
			case "zset":
				if _, exists := fieldTypes["_zset"]; !exists {
					fieldTypes["_zset"] = "sorted_set"
				}
			case "string":
				if _, exists := fieldTypes["_value"]; !exists {
					fieldTypes["_value"] = "string"
				}
			default:
				if _, exists := fieldTypes["_value"]; !exists {
					fieldTypes["_value"] = typeResult
				}
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	for name, typ := range fieldTypes {
		columns = append(columns, models.ColumnInfo{
			Name:     name,
			Type:     typ,
			Nullable: true,
		})
	}
	sort.Slice(columns, func(i, j int) bool {
		return columns[i].Name < columns[j].Name
	})

	return &models.TableInfo{Name: table, Columns: columns}, nil
}

func (c *RedisClient) Execute(ctx context.Context, database, query string, limit int) (*models.QueryResult, error) {
	parts := strings.Fields(query)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty Redis command")
	}

	args := make([]any, len(parts))
	for i, p := range parts {
		args[i] = p
	}

	start := time.Now()
	result, err := c.client.Do(ctx, args...).Result()
	if err != nil {
		return nil, err
	}

	columns, rows := formatRedisResult(result, limit)
	elapsedMs := float64(time.Since(start).Microseconds()) / 1000.0
	return &models.QueryResult{
		Columns:         columns,
		Rows:            rows,
		RowCount:        len(rows),
		ExecutionTimeMs: roundFloat(elapsedMs, 2),
		Truncated:       false,
		Limit:           limit,
	}, nil
}

func formatRedisResult(result any, limit int) ([]models.QueryColumnMeta, [][]any) {
	switch v := result.(type) {
	case []any:
		columns := []models.QueryColumnMeta{
			{Name: "index", Type: "INT"},
			{Name: "value", Type: "TEXT"},
		}
		rows := make([][]any, 0, len(v))
		for i, item := range v {
			if i >= limit {
				break
			}
			rows = append(rows, serializeRow([]any{i, fmt.Sprintf("%v", item)}))
		}
		return columns, rows
	case map[any]any:
		columns := []models.QueryColumnMeta{
			{Name: "field", Type: "TEXT"},
			{Name: "value", Type: "TEXT"},
		}
		rows := make([][]any, 0, len(v))
		for field, val := range v {
			if len(rows) >= limit {
				break
			}
			rows = append(rows, serializeRow([]any{fmt.Sprintf("%v", field), fmt.Sprintf("%v", val)}))
		}
		return columns, rows
	default:
		columns := []models.QueryColumnMeta{
			{Name: "result", Type: "TEXT"},
		}
		rows := [][]any{serializeRow([]any{fmt.Sprintf("%v", result)})}
		return columns, rows
	}
}

// extractKeyPrefix extracts the prefix from a Redis key by splitting on ':'.
func extractKeyPrefix(key string) string {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) > 1 {
		return parts[0] + ":*"
	}
	return key
}

// GetServerInfo returns runtime metadata from Redis.
func (c *RedisClient) GetServerInfo(ctx context.Context) (*models.ServerInfo, error) {
	result, err := c.client.Info(ctx, "server").Result()
	if err != nil {
		return nil, err
	}

	serverInfo := &models.ServerInfo{}

	// Parse INFO output (key:value format)
	for _, line := range strings.Split(result, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]
		switch key {
		case "redis_version":
			serverInfo.Version = value
		}
	}

	return serverInfo, nil
}
