package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/gnodux/adb-link/internal/models"
)

// ESClient is a simple Elasticsearch HTTP client.
type ESClient struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

// NewESClient creates a new Elasticsearch client.
func NewESClient(config *models.DatasourceConfig) *ESClient {
	conn := config.Connection
	scheme := "http"
	if v, ok := config.Options["use_ssl"].(bool); ok && v {
		scheme = "https"
	}
	port := conn.Port
	if port == 0 {
		port = 9200
	}
	timeout := 30 * time.Second
	if v, ok := config.Options["request_timeout"].(int); ok {
		timeout = time.Duration(v) * time.Second
	}
	return &ESClient{
		baseURL:    fmt.Sprintf("%s://%s:%d", scheme, conn.Host, port),
		username:   conn.Username,
		password:   conn.Password,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *ESClient) doRequest(ctx context.Context, method, path string, body any) (map[string]any, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ES request failed (%d): %s", resp.StatusCode, string(respBody))
	}
	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Info returns cluster info.
func (c *ESClient) Info(ctx context.Context) (map[string]any, error) {
	return c.doRequest(ctx, "GET", "/", nil)
}

// GetDatabases returns the cluster name as a virtual database.
func (c *ESClient) GetDatabases(ctx context.Context) ([]models.ObjectName, error) {
	info, err := c.Info(ctx)
	if err != nil {
		return nil, err
	}
	clusterName, _ := info["cluster_name"].(string)
	if clusterName == "" {
		clusterName = "default"
	}
	return []models.ObjectName{{Name: clusterName}}, nil
}

// GetTableNames returns concrete indices (excluding system).
func (c *ESClient) GetTableNames(ctx context.Context) ([]models.ObjectName, error) {
	resp, err := c.doRequest(ctx, "GET", "/_all/_alias", nil)
	if err != nil {
		return nil, err
	}
	var names []string
	for name := range resp {
		if len(name) > 0 && name[0] != '.' {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	result := make([]models.ObjectName, len(names))
	for i, name := range names {
		result[i] = models.ObjectName{Name: name}
	}
	return result, nil
}

// GetTableInfo returns column info from index mappings.
func (c *ESClient) GetTableInfo(ctx context.Context, index string) (*models.TableInfo, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/%s/_mapping", index), nil)
	if err != nil {
		return nil, err
	}
	var indexData map[string]any
	if d, ok := resp[index].(map[string]any); ok {
		indexData = d
	} else {
		// Get first available
		for _, v := range resp {
			if d, ok := v.(map[string]any); ok {
				indexData = d
				break
			}
		}
	}
	if indexData == nil {
		return &models.TableInfo{Name: index}, nil
	}
	mappings, _ := indexData["mappings"].(map[string]any)
	properties, _ := mappings["properties"].(map[string]any)

	var columns []models.ColumnInfo
	collectFields(&columns, "", properties)

	return &models.TableInfo{Name: index, Columns: columns}, nil
}

func collectFields(columns *[]models.ColumnInfo, prefix string, props map[string]any) {
	for fieldName, def := range props {
		fieldDef, ok := def.(map[string]any)
		if !ok {
			continue
		}
		fullName := fieldName
		if prefix != "" {
			fullName = prefix + "." + fieldName
		}
		if fieldType, ok := fieldDef["type"].(string); ok {
			*columns = append(*columns, models.ColumnInfo{
				Name:         fullName,
				Type:         fieldType,
				Nullable:     true,
				IsPrimaryKey: fullName == "_id",
			})
		}
		if subProps, ok := fieldDef["properties"].(map[string]any); ok {
			collectFields(columns, fullName, subProps)
		}
	}
}

// Search executes an ES search query.
func (c *ESClient) Search(ctx context.Context, index string, body map[string]any, size int) (map[string]any, error) {
	q := url.Values{}
	q.Set("size", fmt.Sprintf("%d", size))
	path := fmt.Sprintf("/%s/_search?%s", index, q.Encode())
	return c.doRequest(ctx, "POST", path, body)
}

// Close is a no-op (HTTP client doesn't need explicit close).
func (c *ESClient) Close() error {
	return nil
}
