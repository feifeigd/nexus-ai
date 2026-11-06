# 生产部署指南

NexusAI 由四个进程组成（go-zero 后端、Admin 独立 Runtime、Web 用户端、Admin 管理后台），
外加 PostgreSQL（pgvector）与 Redis 两个基础设施。本文说明生产环境如何构建、配置与部署。

## 1. 服务拓扑

| 服务 | 端口（默认） | 职责 |
| --- | --- | --- |
| go-zero API | `8080` | 企业业务 + Agent 后端 + AG-UI SSE |
| agent-runtime | `4000` | Admin 的独立 CopilotKit Runtime（轻量协议适配） |
| Web（Next.js） | `3000` | 用户端，内嵌 CopilotKit Runtime |
| Admin（Ant Design Pro） | `8000` | 管理后台 |
| PostgreSQL + pgvector | `5432` | 数据与向量检索 |
| Redis | `6379` | 缓存 |

go-zero 是认证、租户隔离、权限、审批与审计的权威；两个 Runtime 只做协议转发，不承载业务。

## 2. 构建产物

```bash
# go-zero 后端（生成二进制）
cd services/api && go build -o nexus-api .

# Admin 独立 Runtime（TypeScript → dist）
pnpm build:runtime

# Web 用户端（Next.js 生产构建）
pnpm build:web

# Admin 管理后台（Umi 生产构建）
pnpm build:admin
```

Web 用 `next start` 运行生产构建；Admin 与 Runtime 用其 `dist` 产物部署。

## 3. 生产环境变量（go-zero）

密钥只注入 go-zero，Runtime、Web、Admin 不持有模型密钥。参考
[`services/api/.env.example`](../services/api/.env.example)。

| 变量 | 说明 |
| --- | --- |
| `NEXUS_AUTH_SECRET` | JWT 签名密钥，必须用强随机值且长度 ≥ 8 |
| `NEXUS_OPENAI_*` / `NEXUS_OPENAI_EMBEDDING_*` | 对话/嵌入模型的提供商、密钥与模型 |
| `NEXUS_POSTGRES_DSN` | PostgreSQL 连接串，配置后启用持久化与 pgvector |
| `NEXUS_GITHUB_*` / `NEXUS_GITEE_*` | OAuth App 凭据与回调地址（生产回调必须 HTTPS） |
| `NEXUS_WEB_BASE_URL` | 三方登录成功后跳回 Web 的地址 |
| `NEXUS_ADMIN_PASSWORD` 等 | 首次启动 seed 的内部账号；生产建议用后即删或改强密码 |

## 4. 反向代理

Admin 与 Web 的 Umi / Next.js 开发代理在生产构建中不生效，需由网关配置：

- **Admin**：`/api/nexus/*` → go-zero；`/api/copilotkit/*` → agent-runtime（同源）。
- **Web**：`/api/auth/*`、`/api/usage`、`/api/tenants/*` → go-zero；`/api/copilotkit/*` → Web 内嵌 Runtime。
- **OAuth 回调**：GitHub / Gitee 的回调地址（`/auth/github/callback`、`/auth/gitee/callback`）指向
  go-zero 的 HTTPS 地址，且必须与 OAuth App 里注册的回调地址完全一致。

## 5. 数据库

使用 `pgvector/pgvector:pg17` 镜像（或自装 PostgreSQL 17 + pgvector 扩展）。go-zero 启动时
按 `services/api/internal/data/migrations/` 下的版本化 SQL（`001` ~ `013`）顺序迁移，幂等可重复执行。

## 6. 安全清单

- [ ] `NEXUS_AUTH_SECRET` 用强随机值，不要用默认值。
- [ ] OAuth Client Secret 只注入 go-zero，不进 Git / 前端 / 日志。
- [ ] OAuth 回调地址改为 HTTPS 域名，并与平台后台注册一致。
- [ ] seed 的 `admin`/`auditor`/`readonly` 账号改强密码，或生产环境不启用 seed。
- [ ] PostgreSQL 连接串、模型密钥等通过 Secret 注入，不写进 ConfigMap 或镜像。
