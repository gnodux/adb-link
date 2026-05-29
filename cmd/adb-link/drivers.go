package main

// Database driver registrations via blank import.
// Each driver registers itself with database/sql on package init,
// matching the driver names returned by services.driverNameFor.

import (
	_ "github.com/ClickHouse/clickhouse-go/v2" // driver: "clickhouse"
	_ "github.com/beltran/gohive"              // driver: "hive"
	_ "github.com/go-sql-driver/mysql"         // driver: "mysql"
	_ "github.com/lib/pq"                      // driver: "postgres"
	_ "github.com/microsoft/go-mssqldb"        // driver: "sqlserver"
	_ "modernc.org/sqlite"                     // driver: "sqlite" (pure Go)
)
