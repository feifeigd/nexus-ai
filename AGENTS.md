# NexusAI 项目上下文

## 项目目标

NexusAI 是暂定名称。项目旨在构建企业级 AI 应用开发框架，并内置多租户“企业知识库与智能工单”SaaS 参考应用。项目不是 MaaS；长期方向是 AI 应用框架 / AI PaaS。

## 已确定的技术边界

- go-zero 是唯一企业业务与 Agent 后端，第一阶段采用模块化单体，不提前拆微服务。
- Next.js 用于用户端 Web；Ant Design Pro 用于独立管理后台。
- CopilotKit 负责用户端和管理后台的 AI 前端交互；Copilot Runtime 仅是 Next.js 内的轻量 TypeScript 协议适配层，不重复实现企业业务。
- 管理后台必须提供 AI 聊天助手，支持通过聊天查询数据、辅助填写和触发后台操作；模型只能提议工具调用，go-zero 负责权限、租户隔离、参数校验、人工审批和审计。
- AG-UI 负责 Agent 事件通信，A2UI 负责受控动态 UI；普通业务仍使用 REST。
- 管理后台必须支持通过 `@umijs/max-plugin-openapi` 从 NexusAI 的真实 OpenAPI JSON 生成 TypeScript API 代码；可通过 `OPENAPI_SCHEMA_PATH` 指定 Schema，不能默认依赖上游示例定义。
- PostgreSQL、pgvector、Redis 用于数据、向量检索和缓存；Docker Compose 用于本地环境。
- 后续使用 Expo + React Native 扩展安卓端。
- 当前不引入 NestJS、Python Agent 服务、Kubernetes 或复杂微服务体系。
- 工具权限、租户隔离、参数校验、人工审批和审计必须以 go-zero 后端为权威，不能依赖模型输出或前端状态。

## 协作规则

- 任务尽量拆细，每一步都必须可以运行和验证。
- 关键位置必须有代码测试，不能把未执行的验证写成已通过。
- 使用小特性分支和小提交。
- 每次增量完成后暂停并展示变更与验证结果。
- 未经用户明确 review 通过，不执行 Git 提交。
- 用户明确回复“通过并提交”后，才可提交当前增量。
- 不遗留临时开发服务器或后台进程。

## 当前进度

当前分支：`feat/06-copilotkit-runtime`。

截至 2026-10-29，第 1 至 5 步已合并到 `main` 并推送远端。第 5 步合并提交为 `34a7467`。第 6 步变更尚未提交，当前等待用户 review。

### 已完成并提交

1. 第 1 步提交 `87788d5 feat: add go-zero health service`：pnpm Monorepo 根结构、go-zero 最小服务、`GET /healthz`、`.api` 契约和 Handler 单元测试。已验证 Go 测试、静态检查、格式和真实 HTTP 响应。
2. 第 2 步提交 `30e763e feat: add postgres and redis infrastructure`：Docker Compose 配置 PostgreSQL 17、pgvector 和 Redis 7.4。已验证容器健康检查、PostgreSQL 17.11、pgvector 0.8.7、真实向量运算及 Redis 读写。
3. 第 3 步提交 `302f82e feat: add nextjs web skeleton`：Next.js 16.4.3、React 19.3.0、TypeScript、Tailwind CSS、ESLint、NexusAI 最小首页、页面测试和根级 Web 脚本。
4. 第 4 步提交 `b0b33cc feat: connect web to go-zero health api`：Next.js 服务端请求 go-zero 健康接口，使用 `Suspense` 保留静态页面外壳，并区分 API 连接状态。已验证测试、静态检查、生产构建和真实前后端联调。
5. 第 5 步提交 `e3070ed`、`106da6e`：Ant Design Pro 管理后台骨架、登录与权限基础页面、OpenAPI 代码生成命令、项目 CI 配置及验证修复。合并提交为 `34a7467`。

### 当前待 review 增量

第 6 步：Next.js CopilotKit Runtime 与用户端 Provider。

- 已在 `apps/web` 接入 CopilotKit Runtime v2，通过 `/api/copilotkit/[[...path]]` 暴露 Runtime 端点。
- 已注册指向 `NEXUS_AG_UI_AGENT_URL` 的 `default` HttpAgent，默认地址为 `http://127.0.0.1:8080/ag-ui`；转发头仅允许 `authorization` 和 `x-tenant-id`。
- 已在用户端根布局接入 `CopilotKitProvider`，并用 `Suspense` 满足 Next.js Cache Components 的预渲染约束。
- 已将 Web 直接依赖的 `@ag-ui/client` 固定为 `1.0.2`，与 `@copilotkit/runtime 1.81.0` 的内部依赖一致，避免重复版本导致私有字段类型不兼容。
- 已添加 Runtime Agent 发现契约测试；Web 共 3 个测试文件、4 个用例全部通过，ESLint 和完整生产构建通过。
- 已通过真实生产服务验证 `GET /api/copilotkit/info` 返回 HTTP 200 且包含 `default`，`GET /` 返回 HTTP 200 且包含 `NexusAI`。
- 当前仅完成 Runtime、Provider 和 Agent 发现；go-zero AG-UI 端点、真实 Agent 事件流及 AI 聊天尚未实现，不能据此声称聊天已可用。

### 环境注意事项

- Go 已安装在 `C:\Program Files\Go\bin`，但当前 Shell 的 PATH 可能未包含它。
- Docker Desktop 可用；需要时将 `C:\Program Files\Docker\Docker\resources\bin` 加入 PATH，以便 Compose 和凭据助手正常工作。
- pnpm 未全局安装，可通过 `npx --yes pnpm@11.15.1 <命令>` 执行；Web 和 Admin 依赖均已安装。
- 当前 npm 默认源为 `https://registry.npmmirror.com`，依赖解析可使用官方源 `https://registry.npmjs.org`。
- goctl 因依赖下载超时未用于第 1 步代码生成，当前采用与 go-zero 标准结构一致的手写最小实现，保留 `.api` 契约。
- 第 2 步验证容器已停止并删除，命名数据卷保留。
- 第 6 步临时生产服务器已停止，端口 7100 无监听，当前无项目相关 Node 进程。

## 下一步

等待用户 review 第 6 步；用户明确允许后提交当前增量。

后续步骤依次为：go-zero Mock AG-UI 流、完整 AI 流式闭环、管理后台 CopilotKit Provider、后台 AI 工具调用与人工审批。
