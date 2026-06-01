---
title: Database Support
---

# Supported Databases

ADB-Link supports 13 database engines through a unified interface. Each engine uses a dialect layer for DSN building and schema introspection.

## Overview

| Database | Type Key | SQL | Non-SQL Client | Notes |
|----------|----------|-----|----------------|-------|
| MySQL | `mysql` | Yes | -- | Full schema introspection |
| PostgreSQL | `postgresql` | Yes | -- | Full schema introspection |
| SQLite | `sqlite` | Yes | -- | File-based, zero config |
| ClickHouse | `clickhouse` | Yes | -- | Columnar analytics |
| SQL Server | `mssql` | Yes | -- | Microsoft SQL Server |
| Hive | `hive` | Yes | -- | Apache Hive / HiveServer2 |
| Oracle | `oracle` | Yes | -- | Oracle Database |
| GaussDB | `gaussdb` | Yes | -- | Huawei GaussDB |
| TiDB | `tidb` | Yes | -- | MySQL-compatible, distributed |
| Redis | `redis` | Custom | Yes | Key-value commands |
| MongoDB | `mongodb` | Custom | Yes | Document queries |
| Milvus | `milvus` | Custom | Yes | Vector database |
| Elasticsearch | `elasticsearch` | Custom | Yes | Search and analytics |

---

## SQL Databases

### MySQL

```yaml
kind: datasource
name: my-mysql
type: mysql
connection:
  host: 127.0.0.1
  port: 3306
  username: root
  password: ${MYSQL_PASSWORD}
  default_database: mydb
```

### PostgreSQL

```yaml
kind: datasource
name: my-postgres
type: postgresql
connection:
  host: 127.0.0.1
  port: 5432
  username: postgres
  password: ${PG_PASSWORD}
  default_database: mydb
```

### SQLite

```yaml
kind: datasource
name: local-db
type: sqlite
connection:
  dsn: "/path/to/database.db"
```

No external dependencies. Uses pure Go SQLite driver (modernc.org/sqlite).

### ClickHouse

```yaml
kind: datasource
name: my-clickhouse
type: clickhouse
connection:
  host: 127.0.0.1
  port: 9000
  username: default
  password: ${CH_PASSWORD}
  default_database: default
```

### SQL Server (MSSQL)

```yaml
kind: datasource
name: my-mssql
type: mssql
connection:
  host: 127.0.0.1
  port: 1433
  username: sa
  password: ${MSSQL_PASSWORD}
  default_database: master
```

### Hive

```yaml
kind: datasource
name: my-hive
type: hive
connection:
  host: 127.0.0.1
  port: 10000
  username: hive
  default_database: default
```

### Oracle

```yaml
kind: datasource
name: my-oracle
type: oracle
connection:
  host: 127.0.0.1
  port: 1521
  username: system
  password: ${ORACLE_PASSWORD}
  default_database: ORCL
```

### GaussDB

```yaml
kind: datasource
name: my-gaussdb
type: gaussdb
connection:
  host: 127.0.0.1
  port: 5432
  username: gaussdb
  password: ${GAUSS_PASSWORD}
  default_database: postgres
```

### TiDB

```yaml
kind: datasource
name: my-tidb
type: tidb
connection:
  host: 127.0.0.1
  port: 4000
  username: root
  password: ${TIDB_PASSWORD}
  default_database: test
```

TiDB uses the MySQL-compatible dialect.

---

## Non-SQL Databases

### Redis

```yaml
kind: datasource
name: my-redis
type: redis
connection:
  host: 127.0.0.1
  port: 6379
  password: ${REDIS_PASSWORD}
  default_database: "0"
```

Redis commands are executed via the `execute_query` tool with Redis command syntax.

### MongoDB

```yaml
kind: datasource
name: my-mongo
type: mongodb
connection:
  host: 127.0.0.1
  port: 27017
  username: admin
  password: ${MONGO_PASSWORD}
  default_database: mydb
```

MongoDB queries use JSON DSL syntax via the `execute_query` tool.

### Milvus

```yaml
kind: datasource
name: my-milvus
type: milvus
connection:
  host: 127.0.0.1
  port: 19530
  default_database: default
```

Milvus vector database for similarity search operations.

### Elasticsearch

```yaml
kind: datasource
name: my-es
type: elasticsearch
connection:
  host: 127.0.0.1
  port: 9200
  username: elastic
  password: ${ES_PASSWORD}
```

Elasticsearch queries use JSON DSL syntax.

---

## Schema Discovery

All databases support schema discovery through the unified interface:

- `list_databases` -- Available databases/indices/collections
- `get_schema` -- Tables/collections with column information
- `get_table_info` -- Detailed column types, nullability, and comments
- `get_view_info` -- View definitions and columns

The dialect layer handles the translation between database-specific metadata queries and the unified response format.

---

## Connection Options

Common options available for all SQL databases:

| Option | Description | Default |
|--------|-------------|---------|
| `pool_size` | Max open connections | 10 |
| `max_idle` | Max idle connections | 5 |
| `max_lifetime` | Connection max lifetime | 1h |

Connection health is monitored automatically with periodic ping checks.
