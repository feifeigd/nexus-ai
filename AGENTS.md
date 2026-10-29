# NexusAI 项目上下文

## 项目目标

NexusAI 是暂定名称。项目旨在构建企业级 AI 应用开发框架，并内置多租户“企业知识库与智能工单”SaaS 参考应用。项目不是 MaaS；长期方向是 AI 应用框架 / AI PaaS。

## 已确定的技术边界

- go-zero 是唯一企业业务与 Agent 后端，第一阶段采用模块化单体，不提前拆微服务。
- Next.js 用于用户端 Web；Ant Design Pro 用于独立管理后台。
- CopilotKit 负责用户端和管理后台的 AI 前端交互；Web 使用 Next.js 内嵌 Copilot Runtime，Admin 使用 `apps/agent-runtime` 独立轻量 TypeScript 协议适配服务，两者都不重复实现企业业务。
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

当前分支：`feat/10-go-zero-real-model`。

截至 2026-10-29，第 1 至 9 步已合并到 `main` 并推送远端。第 9 步提交为 `c810437`，合并提交为 `9f0d48b`；后续 CI 修复提交为 `8e747aa`。第 10 步实现和本地验证已完成，正在等待用户 review，尚未提交。

### 已完成并提交

1. 第 1 步提交 `87788d5 feat: add go-zero health service`：pnpm Monorepo 根结构、go-zero 最小服务、`GET /healthz`、`.api` 契约和 Handler 单元测试。已验证 Go 测试、静态检查、格式和真实 HTTP 响应。
2. 第 2 步提交 `30e763e feat: add postgres and redis infrastructure`：Docker Compose 配置 PostgreSQL 17、pgvector 和 Redis 7.4。已验证容器健康检查、PostgreSQL 17.11、pgvector 0.8.7、真实向量运算及 Redis 读写。
3. 第 3 步提交 `302f82e feat: add nextjs web skeleton`：Next.js 16.4.3、React 19.3.0、TypeScript、Tailwind CSS、ESLint、NexusAI 最小首页、页面测试和根级 Web 脚本。
4. 第 4 步提交 `b0b33cc feat: connect web to go-zero health api`：Next.js 服务端请求 go-zero 健康接口，使用 `Suspense` 保留静态页面外壳，并区分 API 连接状态。已验证测试、静态检查、生产构建和真实前后端联调。
5. 第 5 步提交 `e3070ed`、`106da6e`：Ant Design Pro 管理后台骨架、登录与权限基础页面、OpenAPI 代码生成命令、项目 CI 配置及验证修复。合并提交为 `34a7467`。
6. 第 6 步提交 `4fe5376 feat: add copilotkit runtime`：Next.js CopilotKit Runtime v2、用户端 Provider、`default` Agent 注册及 Agent 发现契约测试。已验证 Web 测试、ESLint、生产构建和真实 Runtime 端点。
7. 第 7 步提交 `1e0e7b0 feat: add go-zero ag-ui mock stream`：go-zero `POST /ag-ui`、SSE Mock 事件流、请求校验、Go 测试和跨平台 Python 联调脚本。已验证直接及经 Runtime 的完整协议链路。
8. 第 8 步提交 `0ded9cf feat: add web mock chat stream`：用户端 CopilotChat、响应式双栏首页和聊天测试。已验证 Web 测试、ESLint、生产构建和真实浏览器流式聊天闭环。
9. 第 9 步提交 `c810437 feat: add admin copilotkit chat`：Admin 根容器接入 `CopilotKitProvider`，新增 `/ai-chat` Mock 流式聊天页面，复用 Web 同源 `/api/copilotkit` Runtime，不新增 Agent 后端。补充 Admin 开发代理、Provider 和页面测试、`swagger-ui-dist` 显式依赖、菜单国际化键、现代浏览器构建目标及 goctl VS Code 插件推荐。

### 第 9 步验证记录

- Admin 共 6 个测试文件、30 个用例均已执行通过；最近一次完整运行在前 28 项后超时，剩余 2 项单独运行通过。Biome 和 TypeScript 检查通过。
- 第 9 步主体及补充 `swagger-ui-dist` 后的 Admin 生产构建均通过，并生成 `/ai-chat` 静态页面；Web 5 个测试、ESLint 和生产构建通过，Go 测试与 vet 通过。
- 最后的 ES2017 Webpack 输出目标修改已通过 TypeScript 检查，但随后的 3 次生产构建均在 5 分钟工具时限内未完成，不能标记为最新配置构建通过，后续需要重新验证。
- 已在真实浏览器中验证 Admin Mock 流式聊天：`/info` 与 Agent `/run` 请求成功，中文消息收到完整 Mock 回复；发送消息过程中未捕获新的控制台错误。
- 真实模型、知识库、业务工具、租户权限、人工审批和持久化尚未接入；本步没有验证真实后台认证或业务操作。

### 第 10 步变更与验证记录

- go-zero 新增可注入的 Agent `Runner`，保留默认 Mock 流，并新增 OpenAI 兼容 `POST /chat/completions` 流式客户端，将模型文本增量转换为 AG-UI 事件。
- 通过 `NEXUS_AGENT_MODE`、`NEXUS_OPENAI_BASE_URL`、`NEXUS_OPENAI_API_KEY` 和 `NEXUS_OPENAI_MODEL` 配置模型；根目录 `.env.example` 提供示例。本地兼容服务允许不配置 API Key。
- 模型请求在 SSE 开始前失败时返回 HTTP 502；流开始后失败时发送通用 `RUN_ERROR`，内部错误仅记录到服务端日志。
- `go test ./...`、`go vet ./...`、gofmt 和 `git diff --check` 均已执行通过。
- 已运行独立进程级验证：本地 OpenAI 兼容假服务收到正确的 `/v1/chat/completions` 路径、Bearer 测试密钥、模型名和消息；go-zero 输出两条中文文本增量及完整有序的 AG-UI 事件流。验证进程、假服务和临时脚本已清理。
- 尚未使用真实供应商和真实密钥验证，也尚未重新验证 Next.js Runtime 到 Web、Admin 的真实模型浏览器闭环，不能声称这些验证已通过。

### 环境注意事项

- Go 已安装在 `C:\Program Files\Go\bin`，但当前 Shell 的 PATH 可能未包含它。
- Docker Desktop 可用；需要时将 `C:\Program Files\Docker\Docker\resources\bin` 加入 PATH，以便 Compose 和凭据助手正常工作。
- pnpm 未全局安装，可通过 `npx --yes pnpm@11.15.1 <命令>` 执行；Web 和 Admin 依赖均已安装。
- 当前 npm 默认源为 `https://registry.npmmirror.com`，依赖解析可使用官方源 `https://registry.npmjs.org`。
- goctl 因依赖下载超时未用于第 1 步代码生成，当前采用与 go-zero 标准结构一致的手写最小实现，保留 `.api` 契约。
- 第 2 步验证容器已停止并删除，命名数据卷保留。
- 第 9 步临时 Web、Admin 静态服务、无头浏览器、脚本和日志已清理；用户自行启动的 API、Web 和 Admin 开发服务不在自动清理范围内。
- 第 10 步进程级验证产生的临时服务、配置、二进制和脚本已清理。

## 下一步

等待用户 review 第 10 步；只有用户明确回复“通过并提交”后才可提交。真实供应商的 Web 与 Admin 浏览器闭环仍需配置有效模型端点和密钥后验证。

后续步骤依次为：后台 AI 工具调用与人工审批、企业知识库与智能工单。
