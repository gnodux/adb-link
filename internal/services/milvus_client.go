package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/gnodux/adb-link/internal/models"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// MilvusClient implements NonSQLClient for Milvus vector database.
type MilvusClient struct {
	client client.Client
}

// NewMilvusClient creates a new Milvus client.
func NewMilvusClient(cfg *models.DatasourceConfig) (*MilvusClient, error) {
	conn := cfg.Connection
	port := conn.Port
	if port == 0 {
		port = 19530
	}
	address := fmt.Sprintf("%s:%d", conn.Host, port)

	milvusCfg := client.Config{Address: address}
	if conn.Username != "" {
		milvusCfg.Username = conn.Username
		milvusCfg.Password = conn.Password
	}

	c, err := client.NewClient(context.Background(), milvusCfg)
	if err != nil {
		return nil, fmt.Errorf("milvus connect failed: %w", err)
	}
	return &MilvusClient{client: c}, nil
}

func (c *MilvusClient) Ping(ctx context.Context) error {
	_, err := c.client.ListDatabases(ctx)
	return err
}

func (c *MilvusClient) Close() error {
	return c.client.Close()
}

func (c *MilvusClient) GetDatabases(ctx context.Context) ([]models.ObjectName, error) {
	dbs, err := c.client.ListDatabases(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]models.ObjectName, len(dbs))
	for i, db := range dbs {
		result[i] = models.ObjectName{Name: db.Name}
	}
	return result, nil
}

func (c *MilvusClient) GetTableNames(ctx context.Context, database string) ([]models.ObjectName, error) {
	if database != "" {
		if err := c.client.UsingDatabase(ctx, database); err != nil {
			return nil, err
		}
	}
	collections, err := c.client.ListCollections(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]models.ObjectName, len(collections))
	for i, coll := range collections {
		result[i] = models.ObjectName{Name: coll.Name}
	}
	return result, nil
}

func (c *MilvusClient) GetTableInfo(ctx context.Context, database, table string) (*models.TableInfo, error) {
	if database != "" {
		if err := c.client.UsingDatabase(ctx, database); err != nil {
			return nil, err
		}
	}
	coll, err := c.client.DescribeCollection(ctx, table)
	if err != nil {
		return nil, err
	}

	var columns []models.ColumnInfo
	for _, field := range coll.Schema.Fields {
		columns = append(columns, models.ColumnInfo{
			Name:         field.Name,
			Type:         field.DataType.String(),
			Nullable:     false,
			IsPrimaryKey: field.PrimaryKey,
		})
	}
	return &models.TableInfo{Name: table, Columns: columns}, nil
}

func (c *MilvusClient) Execute(ctx context.Context, database, query string, limit int) (*models.QueryResult, error) {
	if database != "" {
		if err := c.client.UsingDatabase(ctx, database); err != nil {
			return nil, err
		}
	}

	var req map[string]any
	if err := json.Unmarshal([]byte(query), &req); err != nil {
		return nil, fmt.Errorf("invalid Milvus query JSON: %w", err)
	}

	collName, _ := req["collection"].(string)
	if collName == "" {
		return nil, fmt.Errorf("missing 'collection' field in query")
	}

	start := time.Now()

	if _, hasData := req["data"]; hasData {
		return c.executeVectorSearch(ctx, collName, req, limit, start)
	}
	return c.executeScalarQuery(ctx, collName, req, limit, start)
}

func (c *MilvusClient) executeScalarQuery(ctx context.Context, collName string, req map[string]any, limit int, start time.Time) (*models.QueryResult, error) {
	filter, _ := req["filter"].(string)
	outputFields := extractStringSlice(req, "output_fields")
	if len(outputFields) == 0 {
		outputFields = []string{"*"}
	}
	queryLimit := int64(limit)
	if l, ok := req["limit"].(float64); ok && l > 0 {
		queryLimit = int64(l)
	}

	rs, err := c.client.Query(ctx, collName, nil, filter, outputFields, client.WithLimit(queryLimit))
	if err != nil {
		return nil, err
	}

	return columnResultSetToQueryResult(rs, limit, start)
}

func (c *MilvusClient) executeVectorSearch(ctx context.Context, collName string, req map[string]any, limit int, start time.Time) (*models.QueryResult, error) {
	annsField, _ := req["anns_field"].(string)
	if annsField == "" {
		return nil, fmt.Errorf("missing 'anns_field' for vector search")
	}

	dataRaw, _ := req["data"].([]any)
	if len(dataRaw) == 0 {
		return nil, fmt.Errorf("missing or empty 'data' for vector search")
	}

	var vectors [][]float32
	for _, vec := range dataRaw {
		if arr, ok := vec.([]any); ok {
			f32vec := make([]float32, len(arr))
			for i, v := range arr {
				if f, ok := v.(float64); ok {
					f32vec[i] = float32(f)
				}
			}
			vectors = append(vectors, f32vec)
		}
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("no valid vectors in 'data'")
	}

	searchLimit := limit
	if l, ok := req["limit"].(float64); ok && l > 0 {
		searchLimit = int(l)
	}

	outputFields := extractStringSlice(req, "output_fields")
	filter, _ := req["filter"].(string)

	sp, _ := entity.NewIndexFlatSearchParam()
	vectorsF := make([]entity.Vector, len(vectors))
	for i, v := range vectors {
		vectorsF[i] = entity.FloatVector(v)
	}

	results, err := c.client.Search(
		ctx, collName, nil, filter, outputFields, vectorsF,
		annsField, entity.L2, searchLimit, sp,
	)
	if err != nil {
		return nil, err
	}

	return searchResultsToQueryResult(results, limit, start)
}

func columnResultSetToQueryResult(rs client.ResultSet, limit int, start time.Time) (*models.QueryResult, error) {
	if len(rs) == 0 {
		elapsedMs := float64(time.Since(start).Microseconds()) / 1000.0
		return &models.QueryResult{
			Columns:         []models.QueryColumnMeta{},
			Rows:            [][]any{},
			ExecutionTimeMs: roundFloat(elapsedMs, 2),
			Limit:           limit,
		}, nil
	}

	columnNames := make([]string, len(rs))
	for i, col := range rs {
		columnNames[i] = col.Name()
	}

	rowCount := rs[0].Len()
	truncated := rowCount > limit
	if truncated {
		rowCount = limit
	}

	columns := make([]models.QueryColumnMeta, len(columnNames))
	for i, n := range columnNames {
		columns[i] = models.QueryColumnMeta{Name: n, Type: "TEXT"}
	}

	rows := make([][]any, rowCount)
	for r := 0; r < rowCount; r++ {
		row := make([]any, len(rs))
		for c, col := range rs {
			v, _ := col.Get(r)
			row[c] = fmt.Sprintf("%v", v)
		}
		rows[r] = serializeRow(row)
	}

	elapsedMs := float64(time.Since(start).Microseconds()) / 1000.0
	return &models.QueryResult{
		Columns:         columns,
		Rows:            rows,
		RowCount:        len(rows),
		ExecutionTimeMs: roundFloat(elapsedMs, 2),
		Truncated:       truncated,
		Limit:           limit,
	}, nil
}

func searchResultsToQueryResult(results []client.SearchResult, limit int, start time.Time) (*models.QueryResult, error) {
	if len(results) == 0 {
		elapsedMs := float64(time.Since(start).Microseconds()) / 1000.0
		return &models.QueryResult{
			Columns:         []models.QueryColumnMeta{},
			Rows:            [][]any{},
			ExecutionTimeMs: roundFloat(elapsedMs, 2),
			Limit:           limit,
		}, nil
	}

	result := results[0]
	columnNames := []string{"id", "score"}
	for _, col := range result.Fields {
		columnNames = append(columnNames, col.Name())
	}
	sort.Strings(columnNames[2:])

	columns := make([]models.QueryColumnMeta, len(columnNames))
	for i, n := range columnNames {
		columns[i] = models.QueryColumnMeta{Name: n, Type: "TEXT"}
	}

	rowCount := result.IDs.Len()
	truncated := rowCount > limit
	if truncated {
		rowCount = limit
	}

	rows := make([][]any, rowCount)
	for r := 0; r < rowCount; r++ {
		row := make([]any, len(columnNames))
		for c, name := range columnNames {
			switch name {
			case "id":
				idVal, _ := result.IDs.GetAsInt64(r)
				row[c] = fmt.Sprintf("%v", idVal)
			case "score":
				if r < len(result.Scores) {
					row[c] = result.Scores[r]
				}
			default:
				for _, col := range result.Fields {
					if col.Name() == name {
						v, _ := col.Get(r)
						row[c] = fmt.Sprintf("%v", v)
						break
					}
				}
			}
		}
		rows[r] = serializeRow(row)
	}

	elapsedMs := float64(time.Since(start).Microseconds()) / 1000.0
	return &models.QueryResult{
		Columns:         columns,
		Rows:            rows,
		RowCount:        len(rows),
		ExecutionTimeMs: roundFloat(elapsedMs, 2),
		Truncated:       truncated,
		Limit:           limit,
	}, nil
}

func extractStringSlice(req map[string]any, key string) []string {
	raw, ok := req[key].([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
