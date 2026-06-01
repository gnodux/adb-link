---
title: Home
---

# ADB-Link

**Bridging AI Agents to Your Databases**

A lightweight, high-performance database gateway designed for AI agents -- providing unified SQL access, schema discovery, and tool orchestration across multiple database engines via REST API and MCP (Model Context Protocol).

[Getting Started](getting-started) | [Configuration](configuration) | [API Reference](api-reference) | [MCP Tools](mcp-tools) | [Databases](databases) | [中文文档](zh/)

---

## Why ADB-Link?

- **One gateway, 13 databases** -- MySQL, PostgreSQL, ClickHouse, SQLite, SQL Server, Hive, Oracle, GaussDB, TiDB, Redis, MongoDB, Milvus, Elasticsearch
- **AI-native** -- First-class MCP protocol support for Claude, Cursor, and any MCP-compatible agent
- **Zero CGO** -- Single static binary, cross-compile anywhere
- **Dynamic** -- Register/unregister datasources and tools at runtime
- **Secure** -- RBAC permissions with glob-based access control

## Core Features

| Feature | Description |
|---------|-------------|
| Multi-Database Support | 13 database engines with unified interface |
| MCP Protocol | Full JSON-RPC 2.0 (stdio + HTTP transport) |
| Dynamic Tool Registry | Register/unregister query tools at runtime |
| Dynamic Datasource | Register/unregister datasources with connection validation |
| Async Query Engine | Submit long-running queries, poll status, retrieve results |
| Schema Discovery | Databases, tables, views, columns with type & comment info |
| Hot Reload | YAML config changes detected and applied within seconds |
| RBAC Permissions | Glob-based access control on datasources, databases, tables, fields, and tools |
| Connection Health | Auto-ping, periodic health checks, fail-fast on unreachable hosts |
| Pure Go | Zero CGO dependencies -- single static binary |

## Architecture

```
+---------------------------------------------------------+
|                   AI Agent / Client                      |
+----------+-------------------------+--------------------+
           | REST API                | MCP (JSON-RPC)
           v                         v
+---------------------------------------------------------+
|                    ADB-Link Server                       |
|  +----------+ +-----------+ +--------------------+      |
|  |  Router  | |    MCP    | |   Config Service   |      |
|  |  + Auth  | |  Server   | |  (Hot-Reload/YAML) |      |
|  +-----+----+ +-----+-----+ +--------------------+      |
|        |             |                                   |
|  +-----v-------------v---------------------------+       |
|  |          Service Layer                        |       |
|  |  Schema . Query . Async . Permission . Meta   |       |
|  +---------------------+------------------------+       |
|                        |                                 |
|  +---------------------v------------------------+       |
|  |         Connection Service (Pool + Health)    |       |
|  +---------------------+------------------------+       |
|                        |                                 |
|  +---------------------v------------------------+       |
|  |   Dialect Layer (DSN Builder per DB engine)   |       |
|  +-----------------------------------------------+      |
+---------------------------------------------------------+
           |         |         |         |
     +-----+    +----+    +----+    +----+
     v          v         v         v
  MySQL    PostgreSQL  ClickHouse  SQLite ...
```

## Quick Demo

```bash
# List available datasources via MCP
curl -s -X POST http://localhost:8000/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-api-key>" \
  -d '{
    "jsonrpc": "2.0", "id": 1, "method": "tools/call",
    "params": {"name": "list_datasources", "arguments": {}}
  }'
```

## License

[MIT](https://github.com/gnodux/adb-link/blob/main/LICENSE)
