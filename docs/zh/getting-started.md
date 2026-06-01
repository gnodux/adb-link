---
title: 快速开始
---

# 快速开始

## 安装

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

### 系统要求

- Go 1.22+（仅从源码构建时需要）
- 无 CGO 依赖 -- 纯 Go 实现，单一静态二进制文件

---

## 初始配置

配置文件默认存放在 `~/.adb-link/conf/`（可通过 `ADB_LINK_CONFIG_DIR` 环境变量覆盖）。

```bash
mkdir -p ~/.adb-link/conf
cp conf/mcp_stdio.yaml.example ~/.adb-link/conf/mcp_stdio.yaml
```

### 添加数据源

创建 `~/.adb-link/conf/datasource.yaml`：

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

### 配置认证

创建 `~/.adb-link/conf/auth.yaml`：

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

### 设置权限

创建 `~/.adb-link/conf/permission.yaml`：

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

---

## 运行

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

在 MCP 客户端配置文件中添加：

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

### HTTP 传输

远程或多客户端访问时使用 HTTP 传输：

```bash
adb-link run-all  # MCP 服务在 /mcp 端点
```

客户端连接到 `http://host:8000/mcp`，使用 Bearer Token 认证。

---

## 下一步

- [配置参考](configuration) -- 所有配置选项
- [API 参考](api-reference) -- REST API 端点
- [MCP 工具](mcp-tools) -- 可用的 MCP 工具
- [数据库支持](databases) -- 支持的数据库和连接配置
