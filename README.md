# NexusAI

NexusAI 是一个建设中的企业级 AI 应用开发框架，并将内置多租户“企业知识库与智能工单”SaaS 参考应用。项目定位是 AI 应用框架 / AI PaaS，而不是 MaaS。

当前仓库已打通用户端、管理后台、CopilotKit Runtime、go-zero 与 AG-UI SSE 流式聊天链路，并提供 Mock 和 OpenAI 兼容模型两种 Agent 运行模式。业务工具、知识库、工单、真实认证、租户隔离、人工审批和审计仍在后续建设中。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 用户端 Web | Next.js 16、React 19、Tailwind CSS、CopilotKit |
| 管理后台 | Ant Design Pro、Umi Max、CopilotKit |
| 企业与 Agent 后端 | Go、go-zero，第一阶段采用模块化单体 |
| AI 交互协议 | CopilotKit Runtime、AG-UI；A2UI 规划用于受控动态 UI |
| 数据基础设施 | PostgreSQL 17、pgvector、Redis 7.4、Docker Compose |
| 工程化 | pnpm Monorepo、TypeScript、Vitest、ESLint、Biome、GitHub Actions |

完整的当前架构、请求时序与目标架构见 [架构文档](docs/architecture.md)。

## 当前能力

- go-zero 健康检查接口 `GET /healthz`。
- go-zero AG-UI 接口 `POST /ag-ui`，以 SSE 返回 Agent 事件。
- Web 使用 Next.js 内嵌 CopilotKit Runtime；Admin 使用 `apps/agent-runtime` 独立 Runtime，两者均注册 `default` Agent 并向 go-zero 转发 AG-UI 请求。
- 用户端和管理后台均已接入 CopilotKit 流式聊天界面，Admin 聊天不再依赖 Web 进程。
- Agent 默认使用 Mock；也可通过环境变量切换到 OpenAI 兼容的 `/chat/completions` 流式接口。
- Docker Compose 提供 PostgreSQL、pgvector 和 Redis 本地基础设施。
- 管理后台支持从真实 OpenAPI JSON 生成 TypeScript API 代码。

> 当前页面仍保留部分 Mock 阶段文案。切换到真实模型只改变 Agent 的回复来源，不代表后台业务工具、权限审批或持久化已经接入。

## 目录结构

```text
nexus-ai/
├── apps/
│   ├── web/                 # Next.js 用户端与内嵌 CopilotKit Runtime
│   ├── admin/               # Ant Design Pro 管理后台
│   └── agent-runtime/       # Admin 独立 CopilotKit Runtime
├── services/
│   └── api/                 # go-zero API、AG-UI Handler 与 Agent Runner
├── docs/
│   └── architecture.md      # 架构图、时序图与边界说明
├── scripts/
│   └── test-ag-ui.py        # AG-UI 跨进程联调脚本
├── .github/workflows/       # CI
├── compose.yaml             # PostgreSQL、pgvector 与 Redis
├── package.json             # 根级开发、测试和构建命令
└── pnpm-workspace.yaml      # pnpm 工作区
```

## 本地启动

### 环境要求

| 工具 | 版本 |
| --- | --- |
| Node.js | 22 或更高版本（管理后台与独立 Runtime 要求） |
| pnpm | 11.15.1 |
| Go | 1.24 或更高版本 |
| Docker 与 Docker Compose | 可选，仅基础设施需要 |

安装工作区依赖：

```bash
pnpm install --frozen-lockfile
```

### 启动应用

Admin 聊天需要分别启动以下三个服务，不需要启动 Web：

```bash
# 终端 1：go-zero API，默认 Mock 模式
pnpm dev:api
```

```bash
# 终端 2：Admin 独立 CopilotKit Runtime，监听 127.0.0.1:4000
pnpm dev:runtime
```

```bash
# 终端 3：管理后台，代理到独立 Runtime
pnpm dev:admin
```

需要用户端时，再在第四个终端启动 Next.js 用户端及其内嵌 Runtime：

```bash
pnpm dev:web
```

默认访问地址：

| 服务 | 地址 |
| --- | --- |
| 用户端 | http://127.0.0.1:3000 |
| 管理后台 | http://127.0.0.1:8000 |
| 管理后台 AI 助手 | http://127.0.0.1:8000/ai-chat |
| Admin 独立 Runtime | http://127.0.0.1:4000/api/copilotkit/info |
| API 健康检查 | http://127.0.0.1:8080/healthz |
| AG-UI 接口 | http://127.0.0.1:8080/ag-ui |

可使用以下命令检查 API：

```bash
curl http://127.0.0.1:8080/healthz
```

预期响应：

```json
{"status":"ok","service":"nexus-api"}
```

### 可选：启动数据基础设施

当前聊天链路不依赖数据库或 Redis；只有开发相关数据功能时才需要启动：

```bash
docker compose up -d postgres redis
```

停止容器但保留命名数据卷：

```bash
docker compose down
```

## 配置真实模型

Agent 配置由 **go-zero 进程**读取。`.env.example` 是配置模板，应用不会自动加载根目录 `.env`，请将变量实际注入启动 API 的进程环境。

Mock 模式无需密钥，也是默认模式：

```bash
export NEXUS_AGENT_MODE=mock
pnpm dev:api
```

OpenAI 兼容模式的 Bash 示例：

```bash
export NEXUS_AGENT_MODE=openai
export NEXUS_OPENAI_BASE_URL=https://api.openai.com/v1
export NEXUS_OPENAI_API_KEY=replace-with-your-key
export NEXUS_OPENAI_MODEL=your-model-name
pnpm dev:api
```

PowerShell 示例：

```powershell
$env:NEXUS_AGENT_MODE = "openai"
$env:NEXUS_OPENAI_BASE_URL = "https://api.openai.com/v1"
$env:NEXUS_OPENAI_API_KEY = "replace-with-your-key"
$env:NEXUS_OPENAI_MODEL = "your-model-name"
pnpm dev:api
```

| 环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| `NEXUS_AGENT_MODE` | `mock` | `mock` 或 `openai` |
| `NEXUS_OPENAI_BASE_URL` | `https://api.openai.com/v1` | OpenAI 兼容服务的基础地址，不包含 `/chat/completions` |
| `NEXUS_OPENAI_API_KEY` | 空 | 模型服务密钥；允许密钥为空的本地兼容服务可不设置 |
| `NEXUS_OPENAI_MODEL` | 空 | 模型名称，`openai` 模式必填 |

不要将真实密钥写入仓库或提交到 Git。

### 其他环境变量

| 变量 | 使用方 | 默认值 | 用途 |
| --- | --- | --- | --- |
| `NEXUS_API_INTERNAL_URL` | Web 服务端 | `http://127.0.0.1:8080` | 健康检查的 go-zero 地址 |
| `NEXUS_AG_UI_AGENT_URL` | 两个 CopilotKit Runtime | `http://127.0.0.1:8080/ag-ui` | `default` Agent 的 AG-UI 地址 |
| `NEXUS_COPILOT_RUNTIME_HOST` | Admin 独立 Runtime | `127.0.0.1` | Runtime 监听地址；容器部署时按内网拓扑调整 |
| `NEXUS_COPILOT_RUNTIME_PORT` | Admin 独立 Runtime | `4000` | Runtime 监听端口 |
| `NEXUS_COPILOT_RUNTIME_PROXY_TARGET` | Admin 开发代理 | `http://127.0.0.1:4000` | Admin 独立 Runtime 地址 |
| `OPENAPI_SCHEMA_PATH` | Admin OpenAPI 生成 | `apps/admin/schema/openapi.json` | NexusAI 真实 OpenAPI JSON 路径 |

生产环境需要为 Admin 的 `/api/copilotkit` 配置指向独立 Runtime 的同源反向代理；Umi 的开发代理不会在生产构建中生效。独立 Runtime 和 go-zero 均可只暴露在内网。

## 常用命令

以下命令均在仓库根目录执行：

| 命令 | 作用 |
| --- | --- |
| `pnpm dev:api` | 启动 go-zero API |
| `pnpm dev:runtime` | 启动 Admin 独立 Runtime |
| `pnpm dev:web` | 启动用户端 |
| `pnpm dev:admin` | 启动管理后台 |
| `pnpm test:api` | 运行全部 Go 测试 |
| `pnpm test:runtime` | 运行独立 Runtime 协议测试 |
| `pnpm typecheck:runtime` | 检查独立 Runtime TypeScript 类型 |
| `pnpm test:web` | 运行 Web 测试 |
| `pnpm test:admin` | 运行 Admin 测试 |
| `pnpm lint:web` | 运行 Web ESLint |
| `pnpm lint:admin` | 运行 Admin Biome 与 TypeScript 检查 |
| `pnpm build:runtime` | 构建独立 Runtime 生产版本 |
| `pnpm build:web` | 构建 Web 生产版本 |
| `pnpm build:admin` | 构建 Admin 生产版本 |
| `pnpm openapi:admin` | 根据配置的 OpenAPI Schema 生成 Admin API 代码 |
| `pnpm doctor:admin` | 检查 Admin React 使用情况 |

Go 静态检查：

```bash
cd services/api
go vet ./...
```

CI 还会执行 `pnpm --filter admin exec max setup` 和 `pnpm --filter admin exec antd lint ./src`。修改代码后，应在本地执行与变更范围对应的测试、静态检查和构建；不要把未运行的验证写成已通过。

## 安全与职责边界

- CopilotKit Runtime 只是轻量协议适配层，不承载企业业务，也不是第二个 Agent 后端；Web 使用 Next.js 内嵌 Runtime，Admin 使用独立 Runtime 服务。
- go-zero 是企业业务与 Agent 后端；权限、租户隔离、参数校验、人工审批和审计必须以它为权威。
- 模型只能提议工具调用，不能凭模型输出或前端状态绕过服务端校验。
- 两个 Runtime 都只允许向 Agent 后端转发 `authorization` 和 `x-tenant-id` 请求头；这不等于真实认证和租户隔离已经实现。
- 普通业务使用 REST；AG-UI 用于 Agent 事件通信，A2UI 规划用于受控动态 UI。

## 后续路线

1. 完成真实模型接入与 Web、Admin 流式聊天验证，同时保留 Mock 开关。
2. 接入后台 AI 工具调用、权限校验、人工审批和审计。
3. 建设多租户企业知识库与智能工单。
4. 后续使用 Expo + React Native 扩展安卓端。

第一阶段保持 go-zero 模块化单体，不提前拆微服务，不引入 NestJS、Python Agent 服务或 Kubernetes。

## 协作约定

采用小特性分支和小提交，每个增量都应可运行、可验证。未经明确 review 通过，不执行 Git 提交。详细项目边界与进度见 [AGENTS.md](AGENTS.md)。
