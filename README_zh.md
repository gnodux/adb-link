<div align="center">

# ADB-Link

**连接 AI 智能体与数据库的桥梁**

一款为 AI 智能体设计的轻量级、高性能数据库网关 — 通过 REST API 和 MCP（Model Context Protocol）协议提供统一的 SQL 访问、Schema 发现和工具编排能力，支持多种数据库引擎。

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![MCP](https://img.shields.io/badge/MCP-2024--11--05-blueviolet?style=flat-square)](https://modelcontextprotocol.io)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey?style=flat-square)]()

📚 **[文档](https://gnodux.github.io/adb-link)** | [English](README.md) | [中文](README_zh.md)

</div>

---

## 快速安装

```bash
# 一键安装
curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/scripts/install-adb-link.sh | bash

# 或从源码构建（需要 Go 1.22+）
git clone https://github.com/gnodux/adb-link.git
cd adb-link
make build
```

---

## 演示

完整工作流：发现数据源、浏览 Schema、执行查询 — 全部通过 MCP 协议完成。

```bash
# 1. 列出所有数据源
curl -s -X POST http://localhost:8000/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-api-key>" \
  -d '{
    "jsonrpc": "2.0", "id": 1, "method": "tools/call",
    "params": {"name": "list_datasources", "arguments": {}}
  }'
# => [{"name":"my-postgres","type":"postgresql","description":"Production DB", ...}]

# 2. 列出数据源中的数据库
curl -s -X POST http://localhost:8000/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-api-key>" \
  -d '{
    "jsonrpc": "2.0", "id": 2, "method": "tools/call",
    "params": {"name": "list_databases", "arguments": {"datasource_name": "my-postgres"}}
  }'
# => [{"name":"mydb","comment":"Main application database"}, ...]

# 3. 获取数据库 Schema（表和列信息）
curl -s -X POST http://localhost:8000/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-api-key>" \
  -d '{
    "jsonrpc": "2.0", "id": 3, "method": "tools/call",
    "params": {"name": "get_schema", "arguments": {"datasource_name": "my-postgres", "database": "mydb"}}
  }'
# => {"tables":[{"name":"users","columns":[{"name":"id","type":"INT4"},{"name":"username","type":"VARCHAR"}, ...]}]}

# 4. 执行查询
curl -s -X POST http://localhost:8000/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-api-key>" \
  -d '{
    "jsonrpc": "2.0", "id": 5, "method": "tools/call",
    "params": {
      "name": "execute_query",
      "arguments": {
        "datasource_name": "my-postgres",
        "database": "mydb",
        "sql": "SELECT id, username, created_at FROM users ORDER BY created_at DESC LIMIT 10"
      }
    }
  }'
# => {"columns":[...],"rows":[[1,"alice","2026-03-15"], ...],"row_count":3}
```

---

## 核心特性

| 特性 | 说明 |
|------|------|
| **多数据库支持** | MySQL、PostgreSQL、ClickHouse、SQLite、SQL Server、Hive、Oracle、GaussDB、TiDB、Redis、MongoDB、Milvus、Elasticsearch |
| **MCP 协议** | 完整的 JSON-RPC 2.0 实现（支持 stdio 和 HTTP 传输） |
| **动态工具注册** | 通过 API 或 MCP 在运行时注册/注销查询工具 |
| **动态数据源** | 运行时注册/注销数据源，支持连接验证 |
| **异步查询引擎** | 提交长时间运行的查询，轮询状态，获取结果 |
| **Schema 发现** | 数据库、表、视图、列信息（含类型和注释） |
| **热重载** | YAML 配置变更在数秒内被检测并应用 |
| **RBAC 权限** | 基于 Glob 模式的访问控制：数据源、数据库、表、字段、工具 |
| **连接健康检查** | 自动 Ping、定期健康检查、不可达主机快速失败 |
| **纯 Go 实现** | 零 CGO 依赖 — 单一静态二进制文件，可交叉编译到任意平台 |

---

## 快速开始

### 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/scripts/install-adb-link.sh | bash
```

安装最新版本到 `~/.adb-link/`，并在 `~/.local/bin/adb-link` 创建符号链接。

### 从源码构建

```bash
git clone https://github.com/gnodux/adb-link.git
cd adb-link
make build
# 二进制文件: bin/adb-link
```

### 配置

配置文件默认存放在 `~/.adb-link/conf/`（可通过 `ADB_LINK_CONFIG_DIR` 环境变量覆盖）。

```bash
# 复制示例配置文件
mkdir -p ~/.adb-link/conf
cp conf/mcp_stdio.yaml.example ~/.adb-link/conf/mcp_stdio.yaml
# 编辑文件，填入你的数据源信息
```

数据源配置示例：

```yaml
kind: datasource
name: my-postgres
type: postgresql
description: "生产环境 PostgreSQL"
connection:
  host: 127.0.0.1
  port: 5432
  username: app_user
  password: ${PG_PASSWORD}   # 支持环境变量插值
  default_database: mydb
options:
  pool_size: 10
```

认证配置：

```yaml
kind: users
users:
  - name: admin
    api_key: "your-secret-api-key"
    group: admin
  - name: mcp_stdio_user
    group: mcp
    description: "MCP stdio 传输默认用户"
```

### 运行

```bash
# API + MCP HTTP 单端口启动（默认 :8000）
adb-link run-all

# 仅启动 API
adb-link run-api

# 通过 stdio 启动 MCP 服务（用于 IDE/智能体集成）
adb-link run-mcp
```

### 验证

```bash
curl http://localhost:8000/api/health
# {"status":"ok"}
```

---

## MCP 集成

### Claude Desktop / Cursor

在 MCP 客户端配置中添加：

```json
{
  "mcpServers": {
    "adb-link": {
      "command": "adb-link",
      "args": ["run-mcp"]
    }
  }
}
```

stdio 传输使用 `mcp_stdio_user` 作为默认身份。请在 auth/permission YAML 配置文件中为该用户配置权限。

### Claude Code

```bash
claude mcp add adb-link -- adb-link run-mcp
```

### HTTP 传输

远程或多客户端访问时使用 HTTP 传输：

```bash
adb-link run-all  # MCP 服务在 /mcp 端点
```

### 一键安装与更多 Agent 支持

查看 **[Agent 集成指南](docs/install-mcp-agents.md)**，获取：
- 一键安装 Prompt（直接粘贴给任意 Agent，自动完成安装与配置）
- Cursor、Windsurf、Continue 等 Agent 的配置片段
- Windows 路径示例
- 适用于 Qoder CLI 的 [Skill 文件](skills/adb-link.md)

---

## 文档

### 架构

```
┌─────────────────────────────────────────────────────┐
│                  AI 智能体 / 客户端                    │
└──────────┬─────────────────────────┬────────────────┘
           │ REST API                │ MCP (JSON-RPC)
           ▼                         ▼
┌──────────────────────────────────────────────────────┐
│                   ADB-Link 服务器                      │
│  ┌──────────┐ ┌───────────┐ ┌────────────────────┐  │
│  │  路由器   │ │   MCP    │ │    配置服务         │  │
│  │  + 认证   │ │   服务器  │ │  (热重载/YAML)     │  │
│  └─────┬────┘ └─────┬─────┘ └────────────────────┘  │
│        │             │                               │
│  ┌─────▼─────────────▼───────────────────────────┐   │
│  │              服务层                             │   │
│  │  Schema · 查询 · 异步 · 权限 · 元数据          │   │
│  └──────────────────┬────────────────────────────┘   │
│                     │                                │
│  ┌──────────────────▼────────────────────────────┐   │
│  │          连接服务（连接池 + 健康检查）           │   │
│  └──────────────────┬────────────────────────────┘   │
│                     │                                │
│  ┌──────────────────▼────────────────────────────┐   │
│  │      方言层（各数据库引擎 DSN 构建器）          │   │
│  └───────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────┘
           │         │         │         │
     ┌─────┘    ┌────┘    ┌────┘    ┌────┘
     ▼          ▼         ▼         ▼
  MySQL    PostgreSQL  ClickHouse  SQLite ...
```

### API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/health` | 健康检查 |
| POST | `/api/datasources/list` | 列出数据源 |
| POST | `/api/datasources/detail` | 数据源详情 |
| POST | `/api/datasources/test` | 测试连通性 |
| POST | `/api/datasources/register` | 注册数据源 |
| POST | `/api/datasources/unregister` | 注销数据源 |
| POST | `/api/databases/list` | 列出数据库 |
| POST | `/api/schema/get` | 获取完整 Schema |
| POST | `/api/schema/table` | 表信息 |
| POST | `/api/schema/view` | 视图信息 |
| POST | `/api/query/execute` | 执行 SQL |
| POST | `/api/query/explain` | 执行计划 |
| POST | `/api/async/query/submit` | 异步查询提交 |
| POST | `/api/async/query/status` | 异步查询状态 |
| POST | `/api/async/query/result` | 异步查询结果 |
| POST | `/api/async/query/cancel` | 取消异步查询 |
| GET | `/api/tools` | 列出已注册工具 |
| POST | `/api/tool/register` | 注册工具 |
| POST | `/api/tool/unregister` | 注销工具 |
| POST | `/api/tool/{name}` | 执行工具 |
| POST | `/mcp` | MCP JSON-RPC 端点 |

### MCP 工具

所有 MCP 工具通过 `tools/call` 调用：

- `list_datasources` — 列出所有数据源
- `list_databases` — 列出数据源中的数据库
- `get_schema` — 获取完整 Schema
- `get_table_info` / `get_view_info` — 列详情
- `execute_query` — 执行 SQL/DSL
- `explain_query` — 执行计划
- `submit_async_query` — 异步查询
- `get_async_query_status` / `get_async_query_result` — 轮询异步结果
- `register_tool` / `unregister_tool` — 动态工具管理
- `register_datasource` / `unregister_datasource` — 动态数据源管理
- *任何动态注册的工具*

### 配置文件

所有配置基于 YAML 格式，存放在配置目录（默认 `~/.adb-link/conf/`）：

| 文件　　　　　　　　| Kind　　　　 | 用途　　　　　　　　　　　　　　　　　　　 |
| ---------------------| --------------| --------------------------------------------|
| `datasource.yaml`　 | `datasource` | 数据库连接定义　　　　　　　　　　　　　　 |
| `auth.yaml`　　　　 | `users`　　　| API Key 和用户　　　　　　　　　　　　　　 |
| `permission-*.yaml` | `permission` | RBAC 权限规则　　　　　　　　　　　　　　　|
| `tool-*.yaml`　　　 | `tool`　　　 | 自定义查询工具　　　　　　　　　　　　　　 |
| `metadata-*.yaml`　 | `metadata`　 | 列/表注释　　　　　　　　　　　　　　　　　|
| `mcp_stdio.yaml`　　| 混合　　　　 | MCP stdio 默认配置（认证 + 权限 + 数据源） |

支持通过 `${VAR_NAME}` 语法使用环境变量。配置文件变更会自动热重载。

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `ADB_LINK_CONFIG_DIR` | `~/.adb-link/conf` | 配置目录路径 |
| `ADB_LINK_API_HOST` | `0.0.0.0` | API 绑定地址 |
| `ADB_LINK_API_PORT` | `8000` | API 绑定端口 |
| `ADB_LINK_LOG_LEVEL` | `INFO` | 日志级别 |
| `ADB_LINK_LOG_DIR` | `~/.adb-link/logs` | 日志目录 |
| `ADB_LINK_RELOAD` | `true` | 启用热重载 |
| `ADB_LINK_ASYNC_QUERY_TTL` | `3600` | 异步结果 TTL（秒） |

---

## 贡献

欢迎贡献代码！请随时提交 Issue 和 Pull Request。

```bash
# 开发工作流
make fmt       # 格式化代码
make vet       # 静态检查
make test      # 运行测试
make build     # 构建二进制
```

---

## 许可证

[MIT](LICENSE)

---

<div align="center">

**如果 ADB-Link 对你有帮助，请给个 Star！**

</div>
