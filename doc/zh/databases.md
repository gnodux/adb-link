---
title: 数据库支持
---

# 支持的数据库

ADB-Link 通过统一接口支持 13 种数据库引擎。每种引擎使用方言层进行 DSN 构建和 Schema 自省。

## 概览

| 数据库 | 类型标识 | SQL | 非 SQL 客户端 | 备注 |
|--------|----------|-----|---------------|------|
| MySQL | `mysql` | 是 | -- | 完整 Schema 自省 |
| PostgreSQL | `postgresql` | 是 | -- | 完整 Schema 自省 |
| SQLite | `sqlite` | 是 | -- | 文件型，零配置 |
| ClickHouse | `clickhouse` | 是 | -- | 列式分析引擎 |
| SQL Server | `mssql` | 是 | -- | Microsoft SQL Server |
| Hive | `hive` | 是 | -- | Apache Hive / HiveServer2 |
| Oracle | `oracle` | 是 | -- | Oracle Database |
| GaussDB | `gaussdb` | 是 | -- | 华为 GaussDB |
| TiDB | `tidb` | 是 | -- | MySQL 兼容，分布式 |
| Redis | `redis` | 自定义 | 是 | 键值命令 |
| MongoDB | `mongodb` | 自定义 | 是 | 文档查询 |
| Milvus | `milvus` | 自定义 | 是 | 向量数据库 |
| Elasticsearch | `elasticsearch` | 自定义 | 是 | 搜索和分析 |

---

## SQL 数据库

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

无外部依赖，使用纯 Go SQLite 驱动（modernc.org/sqlite）。

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

TiDB 使用 MySQL 兼容方言。

---

## 非 SQL 数据库

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

通过 `execute_query` 工具使用 Redis 命令语法执行操作。

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

MongoDB 通过 `execute_query` 工具使用 JSON DSL 语法查询。

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

Milvus 向量数据库，用于相似度搜索操作。

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

Elasticsearch 使用 JSON DSL 语法查询。

---

## Schema 发现

所有数据库通过统一接口支持 Schema 发现：

- `list_databases` -- 可用的数据库/索引/集合
- `get_schema` -- 表/集合及列信息
- `get_table_info` -- 详细列类型、可空性和注释
- `get_view_info` -- 视图定义和列信息

方言层负责将数据库特定的元数据查询转换为统一的响应格式。

---

## 连接选项

所有 SQL 数据库通用选项：

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `pool_size` | 最大打开连接数 | 10 |
| `max_idle` | 最大空闲连接数 | 5 |
| `max_lifetime` | 连接最大生存时间 | 1h |

连接健康通过定期 Ping 检查自动监控。
