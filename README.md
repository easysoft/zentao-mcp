# ZenTao MCP Server

基于 OpenAPI 规范自动生成 MCP (Model Context Protocol) 工具服务。

## 一：简介
此项目是桥接代理服务，将任何遵循 OpenAPI/Swagger 规范的 REST API 自动转换为 MCP (Model Context Protocol) 工具，使 AI 助手（如 Claude、Cursor）能够无缝调用各类 API 服务。

**1. 核心特性：**

-  **自动转换**：从 OpenAPI/Swagger 文档自动生成 MCP 工具，无需手动编写
-  **双协议支持**：同时支持 Streamable HTTP 和 SSE 两种 MCP 传输协议
- ️ **访问控制**：支持基于方法和路径正则的 allow/block 规则
-  **可观测性**：内置 OpenTelemetry 链路追踪和指标收集
-  **多服务支持**：单个实例可同时代理多个不同的 API 服务

**2. 技术栈：**

- Go 1.24+
- [kin-openapi](https://github.com/getkin/kin-openapi) - OpenAPI 解析
- [go-sdk](https://github.com/modelcontextprotocol/go-sdk) - MCP 协议实现
- OpenTelemetry - 遥测和监控

## 二：支持的 MCP 传输协议
| 传输协议 | 请求方式与接口                                                    | 特点 |
| :--- |:-----------------------------------------------------------| :--- |
| **Streamable HTTP** | 主要通信端点： `POST /{name}/mcp`                                 | 1. 基于纯 HTTP 协议，兼容性好<br>2. 支持会话 ID (Mcp-Session-Id) 管理<br>3. 支持 JSON-RPC 消息格式 |
| **SSE (Server-Sent Events)** | 建立 SSE 连接：`GET /{name}/sse`<br>发送消息到服务端：`POST /{name}/sse` | 1. 单向服务器推送<br>2. 实时事件流<br>3. 兼容传统 MCP 客户端 |

说明：`{name}` 为配置文件中定义的 server 名称
## 三：快速开始
### 1. 安装

```bash
# 克隆项目
git clone https://github.com/easysoft/zentao-mcp.git
cd zentao-mcp

# 构建
go build -o zentao-mcp ./cmd/app
# 或使用：task build
```

### 2. 配置
```yaml
# 服务监听端口
port: ":9090"

# 是否启用 TOON 格式响应
enable_toon: false

# 监控配置
telemetry:
  # 上报端点：为空时输出到控制台或填写链路追踪的地址
  endpoint: ""
  # 是否使用不安全的 HTTP 连接（跳过 TLS 验证）
  insecure: true
  
# 上游 API 服务器配置
servers:
  - name: petstore
    # OpenAPI Schema URL 或本地文件路径
    schema_url: "https://b7du.corp.cc/zentao-openapi.json"
    # API 基础 URL
    base_url: "https://b7du.corp.cc/api.php/v1"
    # 允许访问的规则
    allow:
      - methods: ["GET", "POST", "PUT", "DELETE"]
        regex: ".*"
    # 禁止访问的规则
    block: []
```

### 3. 运行

```bash
./zentao-mcp -config config.yaml
# 或使用：task dev
```

### 4. 开发命令

```bash
task default        # 显示帮助信息
task build          # 构建
task dev            # 开发模式运行
task package        # 打包所有平台
```

### 5. 客户端连接示例

#### 1. 获取凭证
```bash
# 禅道域名
ZENTAO_DOMAIN="http://您的禅道域名"

# 禅道API
curl -X POST "${ZENTAO_DOMAIN}/api.php/v2/user/login" \
   -H "Content-Type: application/json" \
   -d '{"account":"用户名","password":"密码"}'
```

#### 2.1 Claude Desktop 配置

```json
{
  "mcpServers": {
    "zentao": {
      "url": "http://127.0.0.1:9090/zentao/mcp"
    }
  }
}
```

#### 2.2 CodeBuddy 配置
```json
{
  "mcpServers": {
    "zentao": {
      "disabled": false,
      "type": "mcp",
      "url": "http://127.0.0.1:9090/zentao/mcp",
      "timeout": 60000,
      "headers": {
        "token": "318511bf858e6ee9e62ce4135990098d",
        "Authorization": ""
      }
    },
    "gitfox": {
      "disabled": false,
      "type": "sse",
      "url": "http://127.0.0.1:9090/gitfox/sse",
      "timeout": 60000,
      "headers": {
        "Authorization": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOjE3NzU2MjMwMDMsImlzcyI6IkdpdGZveCIsInBpZCI6NSwidGtuIjp7InR5cCI6InBhdCIsImlkIjo1fX0.JLjiPLmEXBmAvppW9-Rz8V_wqo5EWn6M_x2b8aDStI4"
      }
    }
  }
}

```

## 四：支持的平台

- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64, arm64)

## 五：许可证

MIT License
