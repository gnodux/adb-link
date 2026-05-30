//go:build integration

package integration_test

// Database driver imports for integration tests.
import (
	_ "github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"
)
