# Datasource → MCP Resource 完全替代方案 — 影响评估

## Context

用户希望将 datasource 的只读交互从 MCP Tool 完全迁移到 MCP Resource（`resources/list` + `resources/read`），通过 URI 体系（如 `datasource:///mysql-prod/mydb/tables/users`）暴露所有数据源元信息，移除 5 个只读 datasource tool。

---

## URI 设计

| URI / Template | 对应操作 | 后端方法 |
|---|---|---|
| `datasource:///` (静态 resource) | 列出所有数据源 | `ConfigService.ListDatasources()` |
| `datasource:///{name}` (模板) | 数据源详情 | `ConfigService.GetDatasource(name)` |
| `datasource:///{name}/{db}/databases` | 列出数据库 | `SchemaService.GetDatabases()` |
| `datasource:///{name}/{db}/schema` | 完整 schema | `SchemaService.GetSchema()` |
| `datasource:///{name}/{db}/tables/{table}` | 表结构 | `SchemaService.GetTableInfo()` |
| `datasource:///{name}/{db}/views/{view}` | 视图结构 | `SchemaService.GetViewInfo()` |

---

## 改动范围

### 需要修改的文件（4 个）

| 文件 | 变更 | 估算行数 |
|------|------|----------|
| `internal/mcp/server.go` | Server struct 加 `resources`/`templates` 字段；HandleRequest 加 3 个 case；capabilities 加 `resources`；新增 5 个方法 | **+150** |
| `internal/mcp/tools.go` | 删除 5 个只读 tool 注册：`list_datasources`、`list_databases`、`get_schema`、`get_table_info`、`get_view_info` | **-90** |
| `cmd/adb-link/main.go` | 加 `mcp.RegisterCoreResources(mcpServer, container)` 调用 + reload 回调 | **+5** |
| `internal/services/container.go` | 暴露 `NotifyResourceListChanged` 方法 | **+3** |

### 需要新建的文件（1 个）

| 文件 | 内容 | 估算行数 |
|------|------|----------|
| `internal/mcp/resources.go` | `Resource`/`ResourceTemplate`/`ResourceHandler`/`ResourceContent` 类型定义；`RegisterCoreResources` 函数；URI 解析器；模板匹配器；6 个 resource handler | **+250** |

### 需要修改的测试文件（2 个）

| 文件 | 变更 |
|------|------|
| `internal/mcp/server_test.go` | 新增 ~8 个 resource 相关测试 |
| `internal/mcp/tools_test.go` | 删除已移除 tool 的测试 |

### 完全不需要改动的层

- **`internal/services/`**：SchemaService、QueryService、ConnectionService、PermissionService、AsyncQueryService — **零改动**
- **`internal/dialects/`**：全部 13 个 dialect — **零改动**
- **`internal/config/`**：loader、watcher — **零改动**
- **`internal/api/`**：REST API handlers + router — **零改动**
- **`internal/models/`**：所有数据模型 — **零改动**
- **`conf/`**：YAML 配置文件 — **零改动**
- **`internal/mcp/http.go`**、**`internal/mcp/stdio.go`**：传输层 — **零改动**

---

## 保持为 Tool 的操作

以下操作涉及副作用、状态变更或多步流程，不适合 Resource 语义：

| Tool | 原因 |
|------|------|
| `execute_query` | 执行 SQL，有副作用 |
| `explain_query` | 对线上 DB 执行 EXPLAIN |
| `submit_async_query` | 创建后台协程 |
| `get_async_query_status/result` | 读取临时内存状态 |
| `submit_async_tool` | 后台执行 tool |
| `register_datasource` / `unregister_datasource` | 变更配置、持久化 YAML |
| `register_tool` / `unregister_tool` | 变更 tool 注册表 |
| Dynamic tools（YAML 定义的） | 参数化查询执行 |

---

## 迁移策略（推荐分 3 阶段）

### Phase 1: 增量共存（先加不删）
- 新增完整 resource 层，5 个只读 tool 保留并行运行
- MCP client 可通过 `resources/list` + `resources/read` 获取相同数据
- 老客户端不受影响

### Phase 2: 标记废弃
- 5 个只读 tool 的 Description 加 `DEPRECATED: Use resources/read with URI 'datasource:///' instead`

### Phase 3: 移除
- 删除 5 个 tool 注册 + 对应测试
- 更新文档

---

## 核心架构图

```
BEFORE:
  MCP Client → tools/call { "name": "get_schema", args: {ds, db} }
               → tools.go → SchemaService.GetSchema() → dialect

AFTER:
  MCP Client → resources/read { "uri": "datasource:///mysql-prod/mydb/schema" }
               → resources.go → SchemaService.GetSchema() → dialect

  （Service 层、Dialect 层、Permission 模型全部复用，零改动）
```

## 总结

| 指标 | 数值 |
|------|------|
| 新增代码 | ~400 行 |
| 删除代码 | ~140 行 |
| 修改代码 | ~15 行 |
| 新增测试 | ~200 行 |
| **净新增** | **~475 行** |
| **改动文件数** | **7 个（含 1 个新建）** |
| **不改动文件** | **全部 service / dialect / config / api / model 层** |

改动集中在 MCP 展示层，核心业务逻辑完全不受影响。
