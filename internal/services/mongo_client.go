package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gnodux/adb-link/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoClient implements NonSQLClient for MongoDB.
type MongoClient struct {
	client *mongo.Client
}

// NewMongoClient creates a new MongoDB client.
func NewMongoClient(cfg *models.DatasourceConfig) (*MongoClient, error) {
	conn := cfg.Connection
	port := conn.Port
	if port == 0 {
		port = 27017
	}

	uri := ""
	if v, ok := cfg.Options["uri"].(string); ok && v != "" {
		uri = v
	} else if conn.Username != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%d/",
			url.PathEscape(conn.Username), url.PathEscape(conn.Password),
			conn.Host, port)
	} else {
		uri = fmt.Sprintf("mongodb://%s:%d/", conn.Host, port)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongodb connect failed: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongodb ping failed: %w", err)
	}
	return &MongoClient{client: client}, nil
}

func (c *MongoClient) Ping(ctx context.Context) error {
	return c.client.Ping(ctx, nil)
}

func (c *MongoClient) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.client.Disconnect(ctx)
}

func (c *MongoClient) GetDatabases(ctx context.Context) ([]models.ObjectName, error) {
	names, err := c.client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	result := make([]models.ObjectName, len(names))
	for i, n := range names {
		result[i] = models.ObjectName{Name: n}
	}
	return result, nil
}

func (c *MongoClient) GetTableNames(ctx context.Context, database string) ([]models.ObjectName, error) {
	names, err := c.client.Database(database).ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	result := make([]models.ObjectName, len(names))
	for i, n := range names {
		result[i] = models.ObjectName{Name: n}
	}
	return result, nil
}

func (c *MongoClient) GetTableInfo(ctx context.Context, database, table string) (*models.TableInfo, error) {
	sampleSize := 100
	coll := c.client.Database(database).Collection(table)

	opts := options.Find().SetLimit(int64(sampleSize))
	cursor, err := coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	fieldTypes := map[string]map[string]bool{}
	fieldCounts := map[string]int{}
	docCount := 0

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		docCount++
		walkBsonDoc(doc, "", fieldTypes, fieldCounts)
	}

	var columns []models.ColumnInfo
	for name, types := range fieldTypes {
		typeStr := "mixed"
		if len(types) == 1 {
			for t := range types {
				typeStr = t
			}
		} else {
			var typeList []string
			for t := range types {
				typeList = append(typeList, t)
			}
			sort.Strings(typeList)
			typeStr = strings.Join(typeList, "|")
		}
		columns = append(columns, models.ColumnInfo{
			Name:         name,
			Type:         typeStr,
			Nullable:     fieldCounts[name] < docCount,
			IsPrimaryKey: name == "_id",
		})
	}
	sort.Slice(columns, func(i, j int) bool {
		if columns[i].IsPrimaryKey != columns[j].IsPrimaryKey {
			return columns[i].IsPrimaryKey
		}
		return columns[i].Name < columns[j].Name
	})

	return &models.TableInfo{Name: table, Columns: columns}, nil
}

func walkBsonDoc(doc bson.M, prefix string, fieldTypes map[string]map[string]bool, fieldCounts map[string]int) {
	for key, val := range doc {
		fullName := key
		if prefix != "" {
			fullName = prefix + "." + key
		}
		typeName := bsonTypeName(val)
		if _, ok := fieldTypes[fullName]; !ok {
			fieldTypes[fullName] = map[string]bool{}
		}
		fieldTypes[fullName][typeName] = true
		fieldCounts[fullName]++
		if subDoc, ok := val.(bson.M); ok {
			walkBsonDoc(subDoc, fullName, fieldTypes, fieldCounts)
		}
	}
}

func bsonTypeName(v any) string {
	switch v.(type) {
	case bson.ObjectID:
		return "objectId"
	case string:
		return "string"
	case int32:
		return "int32"
	case int64:
		return "int64"
	case float64:
		return "double"
	case bool:
		return "bool"
	case bson.DateTime:
		return "date"
	case bson.A:
		return "array"
	case bson.M, bson.D:
		return "object"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T", v)
	}
}

func (c *MongoClient) Execute(ctx context.Context, database, query string, limit int) (*models.QueryResult, error) {
	var req map[string]any
	if err := json.Unmarshal([]byte(query), &req); err != nil {
		return nil, fmt.Errorf("invalid MongoDB query JSON: %w", err)
	}

	collName, _ := req["collection"].(string)
	if collName == "" {
		return nil, fmt.Errorf("missing 'collection' field in query")
	}
	coll := c.client.Database(database).Collection(collName)

	start := time.Now()

	if pipeline, ok := req["pipeline"]; ok {
		return c.executeAggregation(ctx, coll, pipeline, limit, start)
	}
	return c.executeFind(ctx, coll, req, limit, start)
}

func (c *MongoClient) executeFind(ctx context.Context, coll *mongo.Collection, req map[string]any, limit int, start time.Time) (*models.QueryResult, error) {
	filter := bson.M{}
	if f, ok := req["filter"].(map[string]any); ok {
		filter = toBsonM(f)
	}

	opts := options.Find()
	if proj, ok := req["projection"].(map[string]any); ok {
		opts.SetProjection(toBsonM(proj))
	}
	if sortSpec, ok := req["sort"].(map[string]any); ok {
		opts.SetSort(toBsonM(sortSpec))
	}
	queryLimit := int64(limit)
	if l, ok := req["limit"].(float64); ok && l > 0 {
		queryLimit = int64(l)
	}
	opts.SetLimit(queryLimit + 1)

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	return c.cursorToResult(ctx, cursor, limit, start)
}

func (c *MongoClient) executeAggregation(ctx context.Context, coll *mongo.Collection, pipeline any, limit int, start time.Time) (*models.QueryResult, error) {
	pipelineArr, ok := pipeline.([]any)
	if !ok {
		return nil, fmt.Errorf("'pipeline' must be an array")
	}
	bsonPipeline := make([]bson.D, len(pipelineArr))
	for i, stage := range pipelineArr {
		if stageMap, ok := stage.(map[string]any); ok {
			bsonPipeline[i] = toBsonD(stageMap)
		}
	}

	cursor, err := coll.Aggregate(ctx, bsonPipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	return c.cursorToResult(ctx, cursor, limit, start)
}

func (c *MongoClient) cursorToResult(ctx context.Context, cursor *mongo.Cursor, limit int, start time.Time) (*models.QueryResult, error) {
	var docs []bson.M
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		docs = append(docs, doc)
	}

	truncated := len(docs) > limit
	if truncated {
		docs = docs[:limit]
	}

	columnNames := []string{}
	seen := map[string]bool{}
	for _, doc := range docs {
		flatDoc := flattenBson(doc, "")
		for k := range flatDoc {
			if !seen[k] {
				columnNames = append(columnNames, k)
				seen[k] = true
			}
		}
	}
	sort.Strings(columnNames)

	columns := make([]models.QueryColumnMeta, len(columnNames))
	for i, n := range columnNames {
		columns[i] = models.QueryColumnMeta{Name: n, Type: "TEXT"}
	}

	rows := make([][]any, 0, len(docs))
	for _, doc := range docs {
		flatDoc := flattenBson(doc, "")
		row := make([]any, len(columnNames))
		for i, name := range columnNames {
			if v, ok := flatDoc[name]; ok {
				row[i] = v
			}
		}
		rows = append(rows, serializeRow(row))
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

func flattenBson(doc bson.M, prefix string) map[string]any {
	result := map[string]any{}
	for k, v := range doc {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}
		switch val := v.(type) {
		case bson.M:
			for fk, fv := range flattenBson(val, fullKey) {
				result[fk] = fv
			}
		default:
			result[fullKey] = fmt.Sprintf("%v", val)
		}
	}
	return result
}

func toBsonM(m map[string]any) bson.M {
	result := bson.M{}
	for k, v := range m {
		switch val := v.(type) {
		case map[string]any:
			result[k] = toBsonM(val)
		default:
			result[k] = v
		}
	}
	return result
}

func toBsonD(m map[string]any) bson.D {
	var result bson.D
	for k, v := range m {
		switch val := v.(type) {
		case map[string]any:
			result = append(result, bson.E{Key: k, Value: toBsonD(val)})
		default:
			result = append(result, bson.E{Key: k, Value: v})
		}
	}
	return result
}
