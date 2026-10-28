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

当前分支：`feat/07-go-zero-ag-ui-mock`。

截至 2026-10-29，第 1 至 5 步已合并到 `main` 并推送远端。第 5 步合并提交为 `34a7467`，第 6 步提交为 `4fe5376`。第 7 步变更尚未提交，当前等待用户 review。

### 已完成并提交

1. 第 1 步提交 `87788d5 feat: add go-zero health service`：pnpm Monorepo 根结构、go-zero 最小服务、`GET /healthz`、`.api` 契约和 Handler 单元测试。已验证 Go 测试、静态检查、格式和真实 HTTP 响应。
2. 第 2 步提交 `30e763e feat: add postgres and redis infrastructure`：Docker Compose 配置 PostgreSQL 17、pgvector 和 Redis 7.4。已验证容器健康检查、PostgreSQL 17.11、pgvector 0.8.7、真实向量运算及 Redis 读写。
3. 第 3 步提交 `302f82e feat: add nextjs web skeleton`：Next.js 16.4.3、React 19.3.0、TypeScript、Tailwind CSS、ESLint、NexusAI 最小首页、页面测试和根级 Web 脚本。
4. 第 4 步提交 `b0b33cc feat: connect web to go-zero health api`：Next.js 服务端请求 go-zero 健康接口，使用 `Suspense` 保留静态页面外壳，并区分 API 连接状态。已验证测试、静态检查、生产构建和真实前后端联调。
5. 第 5 步提交 `e3070ed`、`106da6e`：Ant Design Pro 管理后台骨架、登录与权限基础页面、OpenAPI 代码生成命令、项目 CI 配置及验证修复。合并提交为 `34a7467`。
6. 第 6 步提交 `4fe5376 feat: add copilotkit runtime`：Next.js CopilotKit Runtime v2、用户端 Provider、`default` Agent 注册及 Agent 发现契约测试。已验证 Web 测试、ESLint、生产构建和真实 Runtime 端点。

### 当前待 review 增量

第 7 步：go-zero Mock AG-UI 流。

- 已添加 `POST /ag-ui` 和对应 `.api` 契约，接收包含 `threadId`、`runId`、`messages` 的 AG-UI RunAgentInput 子集。
- 已实现 SSE 流式响应，依次发送 `RUN_STARTED`、`TEXT_MESSAGE_START`、两个 `TEXT_MESSAGE_CONTENT`、`TEXT_MESSAGE_END`、`RUN_FINISHED`。
- Mock 回复会拼接最新一条纯文本用户消息，不调用模型，不实现业务工具、持久化、权限或租户逻辑。
- 已限制请求体最大为 1 MiB，并在开始 SSE 前校验必填 ID 和纯文本用户消息。
- 已添加正常事件序列、消息拼接、响应头和非法输入测试；`go test ./...`、`go vet ./...` 和 gofmt 均已执行通过。
- 已添加跨平台 `scripts/test-ag-ui.py`，仅使用 Python 标准库，支持直接测试 go-zero 或通过 Next.js Runtime 测试，自动校验 HTTP 状态、SSE 事件序列和回复文本；两种模式均已实际执行通过。
- 已真实验证直接 `POST http://127.0.0.1:8080/ag-ui` 返回 HTTP 200 和完整事件流。
- 已真实验证通过 `POST http://127.0.0.1:7100/api/copilotkit/agent/default/run` 的 CopilotKit Runtime 链路返回 HTTP 200、相同完整事件序列和正确 Mock 文本。
- 当前仅完成 Mock Agent 事件流；真实模型调用和 Web 聊天界面尚未实现，不能据此声称真实 AI 聊天已可用。

### 环境注意事项

- Go 已安装在 `C:\Program Files\Go\bin`，但当前 Shell 的 PATH 可能未包含它。
- Docker Desktop 可用；需要时将 `C:\Program Files\Docker\Docker\resources\bin` 加入 PATH，以便 Compose 和凭据助手正常工作。
- pnpm 未全局安装，可通过 `npx --yes pnpm@11.15.1 <命令>` 执行；Web 和 Admin 依赖均已安装。
- 当前 npm 默认源为 `https://registry.npmmirror.com`，依赖解析可使用官方源 `https://registry.npmjs.org`。
- goctl 因依赖下载超时未用于第 1 步代码生成，当前采用与 go-zero 标准结构一致的手写最小实现，保留 `.api` 契约。
- 第 2 步验证容器已停止并删除，命名数据卷保留。
- 第 7 步临时 API 与 Web 联调服务及临时文件已清理，端口 7100 和 8080 无监听。用户此前在端口 3000 启动的 Web 开发服务器保持运行，未由本步停止。

## 下一步

等待用户 review 第 7 步；用户明确允许后提交当前增量。

后续步骤依次为：完整 AI 流式聊天界面闭环、管理后台 CopilotKit Provider、真实模型接入、后台 AI 工具调用与人工审批。
