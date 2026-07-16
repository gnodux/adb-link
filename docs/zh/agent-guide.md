---
title: Agent 安装指南
---

# Agent 安装指南

几分钟内安装 adb-link 并为任意 AI Agent 配置 MCP。

---

## 快速安装

运行一键安装脚本 — 自动检测 Agent 平台并完成全部配置：

```bash
curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/skills/adb-link/scripts/setup-mcp.sh | bash
```

脚本将执行以下操作：

1. 安装 adb-link 二进制文件到 `~/.adb-link/bin/`
2. 创建默认配置文件 `~/.adb-link/conf/mcp_stdio.yaml`（如不存在）
3. 检测 Agent 平台（Claude Desktop / Cursor / Windsurf / Qoder CLI）并自动注册 MCP 服务器

运行完成后，**重启 Agent** 并调用 `list_datasources` 验证。

---

## 三种安装模式

adb-link 支持三种安装模式，根据使用场景选择：

### 模式 1 — 仅 CLI

安装二进制文件用于命令行使用，不需要配置 Agent。

```bash
curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/scripts/install-adb-link.sh | bash
```

验证：

```bash
adb-link version
```

### 模式 2 — 远程 MCP

将 Agent 连接到远程 adb-link MCP HTTP 端点，本机无需安装二进制。

向管理员获取 MCP URL（如 `http://10.0.0.1:8000/mcp`），然后添加到 Agent 的 MCP 配置：

```json
{
  "mcpServers": {
    "adb-link": {
      "url": "http://your-server:8000/mcp"
    }
  }
}
```

### 模式 3 — 本地 MCP stdio（默认）

安装 adb-link 二进制并配置 Agent 使用本地 MCP stdio 传输。

```bash
curl -fsSL https://raw.githubusercontent.com/gnodux/adb-link/main/skills/adb-link/scripts/setup-mcp.sh | bash
```

或使用显式参数：

```bash
# --mode cli: 仅 CLI
# --mode remote --url http://...: 远程 MCP
# --mode stdio: 本地 MCP stdio（默认）
bash setup-mcp.sh --mode stdio --agent claude
```

---

## 粘贴给 Agent 的安装指令

将 [AGENT_INSTALL.md](https://github.com/gnodux/adb-link/blob/main/AGENT_INSTALL.md) 的内容复制并粘贴到任意 AI Agent 对话中（Claude、Cursor、Windsurf、Qoder 等），Agent 会读取指令并自动执行安装。

---

## 手动平台配置

如果自动安装未检测到你的平台，请手动配置。

### Claude Desktop

配置文件路径：

| 操作系统 | 路径 |
|---------|------|
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Windows | `%APPDATA%\Claude\claude_desktop_config.json` |
| Linux | `~/.config/Claude/claude_desktop_config.json` |

**本地 stdio：**

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

**远程 MCP：**

```json
{
  "mcpServers": {
    "adb-link": {
      "url": "http://your-server:8000/mcp"
    }
  }
}
```

编辑后重启 Claude Desktop。

### Cursor

配置文件：`~/.cursor/mcp.json`（全局）或 `.cursor/mcp.json`（项目级）

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

Cursor 启动时读取配置。修改后重新加载窗口即可生效。

### Windsurf

配置文件：`~/.codeium/windsurf/mcp_config.json`

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

编辑后重启 Windsurf。

### Qoder CLI

安装 Skill 包：

```bash
# 用户级（所有项目可用）
cp -r skills/adb-link ~/.qoder/skills/
```

注册 MCP 服务器：

```bash
# stdio 模式
qoder mcp add adb-link -- adb-link run-mcp

# 或远程模式
qoder mcp add adb-link --url http://your-server:8000/mcp
```

使用 `/skills reload` 重载 Skill，`/skills list` 验证。

---

## 添加数据库

安装完成后，编辑 `~/.adb-link/conf/mcp_stdio.yaml` 添加数据源：

```yaml
---
kind: datasource
name: my-postgres
type: postgresql
description: "我的 PostgreSQL 数据库"
connection:
  host: 127.0.0.1
  port: 5432
  username: app_user
  password: ${PG_PASSWORD}
  default_database: mydb
```

支持的数据库类型：`mysql`、`postgresql`、`sqlite`、`clickhouse`、`mssql`、`elasticsearch`、`hive`、`gaussdb`、`redis`、`mongodb`、`milvus`、`oracle`、`tidb`

完整配置参考见 [配置参考](configuration)。

---

## 验证

在 Agent 中调用 `list_datasources` MCP 工具，应返回已配置的数据源列表。

```bash
# 或从命令行验证
adb-link version
```

---

## 常见问题

**`adb-link: command not found`** — 将 `~/.local/bin` 加入 PATH：

```bash
export PATH="$HOME/.local/bin:$PATH"
```

**没有数据源返回** — 编辑 `~/.adb-link/conf/mcp_stdio.yaml`，添加至少一个数据源配置。

**权限拒绝** — 检查 `~/.adb-link/conf/mcp_stdio.yaml` 是否包含 `mcp_stdio_user` 用户定义及相应权限。

**远程 MCP 连接失败** — 确保远程 adb-link 服务器正在运行（`adb-link run-all`），且 URL 可达。

更多帮助：[GitHub Issues](https://github.com/gnodux/adb-link/issues)

---

## 可用 MCP 工具

安装完成后，adb-link 通过 MCP 暴露以下工具：

| 工具 | 说明 |
|------|------|
| `list_datasources` | 列出所有已配置的数据源 |
| `list_databases` | 列出数据源中的数据库 |
| `get_schema` | 获取完整 Schema（表 + 列） |
| `get_table_info` | 获取表的列详情 |
| `get_view_info` | 获取视图的列详情 |
| `execute_query` | 执行 SQL/DSL 查询 |
| `explain_query` | 获取 SQL 执行计划 |
| `submit_async_query` | 提交长时间运行的查询 |
| `register_datasource` | 添加新的数据库连接 |
| `register_tool` | 创建参数化查询工具 |

完整参数参考见 [MCP 工具](mcp-tools)。
