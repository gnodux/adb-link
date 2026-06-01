---
title: 配置参考
---

# 配置参考

所有配置基于 YAML 格式，存放在配置目录（默认 `~/.adb-link/conf/`）。

## 配置目录

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `ADB_LINK_CONFIG_DIR` | `~/.adb-link/conf` | 配置目录路径 |

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `ADB_LINK_API_HOST` | `0.0.0.0` | API 绑定地址 |
| `ADB_LINK_API_PORT` | `8000` | API 绑定端口 |
| `ADB_LINK_LOG_LEVEL` | `INFO` | 日志级别（DEBUG, INFO, WARN, ERROR） |
| `ADB_LINK_LOG_DIR` | `~/.adb-link/logs` | 日志目录 |
| `ADB_LINK_RELOAD` | `true` | 启用热重载 |
| `ADB_LINK_ASYNC_QUERY_TTL` | `3600` | 异步结果 TTL（秒） |

## 配置文件类型

每个 YAML 文件使用 `kind:` 字段作为标识：

| Kind | 用途 | 示例文件 |
|------|------|----------|
| `datasource` | 数据库连接定义 | `datasource.yaml` |
| `users` | API Key 和用户账号 | `auth.yaml` |
| `permission` | RBAC 访问控制规则 | `permission.yaml` |
| `tool` | 自定义查询工具定义 | `tool-reports.yaml` |
| `metadata` | 列/表注释标注 | `metadata-mydb.yaml` |

---

## 数据源配置

```yaml
kind: datasource
name: my-postgres
type: postgresql
description: "生产环境 PostgreSQL"
connection:
  host: 127.0.0.1
  port: 5432
  username: app_user
  password: ${PG_PASSWORD}
  default_database: mydb
options:
  pool_size: 10
```

### 支持的类型

`mysql`, `postgresql`, `clickhouse`, `sqlite`, `mssql`, `hive`, `oracle`, `gaussdb`, `tidb`, `redis`, `mongodb`, `milvus`, `elasticsearch`

### 连接字段

| 字段 | 说明 |
|------|------|
| `host` | 数据库主机 |
| `port` | 数据库端口 |
| `username` | 连接用户名 |
| `password` | 连接密码（支持 `${ENV_VAR}`） |
| `default_database` | 默认连接的数据库 |
| `dsn` | 完整 DSN 字符串（替代各个独立字段） |

### 选项

| 选项 | 说明 |
|------|------|
| `pool_size` | 连接池大小 |
| `max_idle` | 最大空闲连接数 |
| `max_lifetime` | 连接最大生存时间 |

---

## 认证配置（Users）

```yaml
kind: users
users:
  - name: admin
    api_key: "your-secret-api-key"
    group: admin
    email: "admin@example.com"
    description: "管理员"
  - name: readonly
    api_key: "readonly-key"
    group: viewer
  - name: mcp_stdio_user
    group: mcp
    description: "MCP stdio 传输默认用户"
```

### 字段

| 字段 | 必填 | 说明 |
|------|------|------|
| `name` | 是 | 用户名 |
| `api_key` | 否 | Bearer Token（支持 `${ENV_VAR}`） |
| `group` | 是 | 权限组 |
| `email` | 否 | 用户邮箱 |
| `description` | 否 | 用户描述 |

---

## 权限规则

```yaml
kind: permission
groups: ["admin"]
enable: true
rules:
  - datasource: "*"
    databases: ["*"]
    tables: ["*"]
    fields: ["*"]
tools: ["*"]
```

### 字段

| 字段 | 说明 |
|------|------|
| `groups` | 此权限适用的组列表 |
| `enable` | 此权限集是否启用 |
| `rules` | 访问规则列表 |
| `rules[].datasource` | 数据源 Glob 模式 |
| `rules[].databases` | 数据库 Glob 模式 |
| `rules[].tables` | 表 Glob 模式 |
| `rules[].fields` | 字段 Glob 模式 |
| `tools` | 工具 Glob 模式 |

Glob 模式支持 `*`（匹配任意）和精确名称。

---

## 自定义工具

```yaml
kind: tool
name: get_user_orders
description: "获取用户最近订单"
datasource: my-postgres
database: mydb
sql: "SELECT * FROM orders WHERE user_id = :user_id ORDER BY created_at DESC LIMIT :limit"
parameters:
  - name: user_id
    type: integer
    description: "用户 ID"
    required: true
  - name: limit
    type: integer
    description: "最大返回数"
    default: 10
```

### 工具字段

| 字段 | 说明 |
|------|------|
| `name` | 工具名称（用于 MCP `tools/call`） |
| `description` | 工具描述（展示给智能体） |
| `datasource` | 目标数据源 |
| `database` | 目标数据库 |
| `sql` | SQL 模板，使用命名参数（`:param`） |
| `parameters` | 参数定义，使用 JSON Schema 类型 |

---

## 元数据标注

```yaml
kind: metadata
datasource: my-postgres
database: mydb
tables:
  - name: users
    comment: "应用用户表"
    columns:
      - name: id
        comment: "主键"
      - name: status
        comment: "0=未激活, 1=正常, 2=封禁"
```

元数据标注为 Schema 发现提供可读的注释信息。

---

## 热重载

配置变更通过文件系统通知（fsnotify）自动检测，变更在数秒内生效，无需重启服务。

支持热重载的操作：
- 新增/删除/修改数据源
- 更新用户和权限
- 新增/删除自定义工具
- 更新元数据标注

---

## 环境变量插值

所有 YAML 配置值支持 `${ENV_VAR}` 语法：

```yaml
connection:
  password: ${DB_PASSWORD}
  host: ${DB_HOST}
```

如果环境变量未设置，原始 `${VAR}` 字符串保持不变。

---

## MCP Stdio 组合配置

MCP stdio 模式下，单个文件可包含所有配置类型：

```yaml
kind: users
users:
  - name: mcp_stdio_user
    group: admin
---
kind: permission
groups: ["admin"]
enable: true
rules:
  - datasource: "*"
    databases: ["*"]
    tables: ["*"]
    fields: ["*"]
tools: ["*"]
---
kind: datasource
name: local-sqlite
type: sqlite
connection:
  dsn: "/path/to/database.db"
```

支持多文档 YAML（以 `---` 分隔）。
