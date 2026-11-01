# NexusAI 项目上下文

## 项目目标

NexusAI 是暂定名称。项目旨在构建企业级 AI 应用开发框架，并内置多租户“企业知识库与智能工单”SaaS 参考应用。项目不是 MaaS；长期方向是 AI 应用框架 / AI PaaS。

## 已确定的技术边界

- go-zero 是唯一企业业务与 Agent 后端，第一阶段采用模块化单体，不提前拆微服务。
- Next.js 用于用户端 Web；Ant Design Pro 用于独立管理后台。
- CopilotKit 负责用户端和管理后台的 AI 前端交互；Web 使用 Next.js 内嵌 Copilot Runtime，Admin 使用 `apps/agent-runtime` 独立轻量 TypeScript 协议适配服务，两者都不重复实现企业业务。
- 管理后台必须提供 AI 聊天助手，支持通过聊天查询数据、辅助填写和触发后台操作；模型只能提议工具调用，go-zero 负责权限、租户隔离、参数校验、人工审批和审计。
- AG-UI 负责 Agent 事件通信，A2UI 负责受控动态 UI；普通业务仍使用 REST。
- 后续接入自带交互界面的 MCP 工具时，可按需引入 MCP Apps（MCP-UI），作为隔离的工具面板；它不替代 AG-UI、A2UI 或固定业务页面，UI 触发的业务操作仍必须由 go-zero 鉴权、校验、审批和审计。
- 管理后台必须支持通过 `@umijs/max-plugin-openapi` 从 NexusAI 的真实 OpenAPI JSON 生成 TypeScript API 代码；可通过 `OPENAPI_SCHEMA_PATH` 指定 Schema，不能默认依赖上游示例定义。
- PostgreSQL、pgvector、Redis 用于数据、向量检索和缓存；Docker Compose 用于本地环境。
- 后续使用 Expo + React Native 扩展安卓端。
- 当前不引入 NestJS、Python Agent 服务、Kubernetes 或复杂微服务体系。
- Eino 当前暂不引入：现阶段的模型接入与流式协议转换不需要额外 Agent 编排抽象。后续进入企业知识库、RAG、复杂工具调用或多步骤 Agent 工作流时重新评估；若引入，应作为 go-zero 内 Agent `Runner` 的一种编排实现，不替代 go-zero，也不承接权限、租户隔离、人工审批或审计。
- 工具权限、租户隔离、参数校验、人工审批和审计必须以 go-zero 后端为权威，不能依赖模型输出或前端状态。

## 依赖升级约束

- 当前 `@copilotkit/runtime@1.81.0` 将 `@ag-ui/client`、`@ag-ui/core`、`@ag-ui/encoder` 固定为 `1.0.2`；`@copilotkit/core@1.81.0` 将 `@ag-ui/client` 固定为 `1.0.2`；`@copilotkit/react-core@1.81.0` 将 `@ag-ui/client`、`@ag-ui/core` 固定为 `1.0.2`。
- 上述 CopilotKit 包仍固定 AG-UI `1.0.2` 时，禁止单独升级 AG-UI 依赖，也不要通过 pnpm overrides 等方式强行绕过版本约束。只有这些相关 CopilotKit 包全部升级、且其依赖声明共同兼容目标 AG-UI 版本后，才可同步升级 AG-UI。
- AG-UI 或 CopilotKit 升级前必须核对实际包依赖声明；升级后必须确认锁文件不保留互不兼容的 AG-UI 重复版本，并执行 Web 测试、Lint 和生产构建，未实际执行的验证不得写成通过。
- OpenAI 兼容协议当前继续采用 go-zero 内手写 Chat Completions 客户端，不在第 14 步混入 SDK 替换。手写范围只维护当前文本与函数调用子集，不继续扩展 Responses API、多模态或通用重试等完整 SDK 能力；OpenAI 协议类型与 SSE 解析必须留在 `OpenAIRunner` 内部。后续迁移官方 `openai-go` 时应作为独立小增量，保持 `Runner` 接口、AG-UI 事件和业务行为不变，并复用假服务测试覆盖文本流、工具参数分片、HTTP 错误、异常 EOF、非法 JSON、中途错误和取消。

## 日志规范

- go-zero 进程的新增和修改日志统一使用 `logx`，不再新增标准库 `log` 调用；既有标准库日志在触及相应代码时逐步迁移。
- 服务启动、关闭、配置加载、数据库连接与资源释放等进程级日志，直接使用包级 `logx.Info`、`logx.Infow`、`logx.Errorf`、`logx.Errorw`，不要求请求 Context。
- 后台任务等无请求日志使用包级 `logx`；任务拥有独立生命周期 Context 时，可使用 `logx.WithContext(ctx)`。
- HTTP 请求日志使用 `logx.WithContext(r.Context())`；goctl Logic 直接使用内嵌的 `l.Infof`、`l.Infow`、`l.Errorf`、`l.Errorw`，保留请求链路上下文。
- 关键操作优先使用结构化字段记录审批 ID、工单 ID、错误等信息；不得记录 DSN、API Key、完整模型请求或用户敏感内容。同一个错误尽量只在一个层级记录，避免重复日志。
- 资源关闭失败使用 Error 级别，不使用 Fatal。启动失败应优先返回错误，让资源清理完成后由 `main` 记录并退出，避免直接退出跳过 defer。

## 协作规则

- 任务尽量拆细，每一步都必须可以运行和验证。
- 关键位置必须有代码测试，不能把未执行的验证写成已通过。
- 使用小特性分支和小提交。
- 每次增量完成后暂停并展示变更与验证结果。
- 未经用户明确 review 通过，不执行 Git 提交。
- 用户明确回复“通过并提交”后，才可提交当前增量。
- 不遗留临时开发服务器或后台进程。

## 当前进度

当前分支：`main`。

截至 2026-11-01，第 1 至 21 步均已完成并合并到本地 `main`。第 20 步合并提交 `06f3b1e` 已推送到 `origin/main`；第 21 步合并提交为 `a6c5fe3`，本地 `main` 当前领先远端 4 个提交，尚未推送。`feat/21-goctl-api` 分支仍保留，工作区在本节更新前无未提交变更。

### 当前已具备能力

- pnpm Monorepo、Next.js 用户端、Ant Design Pro 管理后台、go-zero API，以及 Docker Compose 中的 PostgreSQL、pgvector 和 Redis 本地基础设施。
- Web 内嵌 CopilotKit Runtime 与 Admin 独立 `apps/agent-runtime` 均已接入 go-zero AG-UI SSE；Agent 支持 Mock 和 OpenAI 兼容 Chat Completions 流式模型。
- go-zero 提供服务端白名单 `create_ticket` 工具提议、审批列表、批准和拒绝接口；批准后才创建工单，重复决策返回冲突。
- Admin 提供 AI 聊天、工具审批和工单页面；工单状态按 `open → in_progress → resolved` 顺序流转，不支持删除、跳级或重新打开。
- `.api` 契约可生成真实 OpenAPI JSON，并通过 `@umijs/max-plugin-openapi` 生成 Admin TypeScript Service；生成文件按 `.gitignore` 规则不纳入 Git，不手动修改。
- 数据访问已迁移到 Ent，Approval 与 Ticket Schema 位于 `services/api/internal/data/ent/schema`；Ent 生成代码同样不纳入 Git。
- go-zero API 已迁移到 `goctl api go --api api/nexus.api --dir .` 标准结构，根命令 `pnpm generate:api` 固定使用 goctl `v1.10.2`。

### 数据源与代码生成约束

- `data.NewData` 是进程唯一的数据库连接所有者，统一负责驱动校验、连接池、Ping、迁移、日志和关闭；Store 必须复用其 Ent Client，不得单独打开或关闭数据库。
- 默认 `services/api/etc/nexus-api.yaml` 使用 SQLite 共享内存 DSN，服务重启后数据清空；设置 `NEXUS_POSTGRES_DSN` 后统一切换到 PostgreSQL。
- 已移除 `MemoryStore`，内存开发和测试也通过 SQLite DSN，不再维护另一套内存业务实现。
- SQLite 使用纯 Go `modernc.org/sqlite`，配置中的 Ent dialect 为 `sqlite3`；PostgreSQL 使用版本化 SQL 迁移，SQLite AutoMigrate 使用 Ent `Schema.Create`。
- PostgreSQL 审批与状态更新使用事务和 `FOR UPDATE`；SQLite 跳过 `FOR UPDATE`，方言差异只保留在 EntStore 的锁定查询方法中。
- 修改 `.api` 后运行 `pnpm generate:api`；修改 Ent Schema 后运行 `pnpm generate:ent`；需要同步 Admin Client 时运行 `pnpm generate:contracts`。不要手动编辑生成文件。

### 最近验证记录

- 第 21 步已执行并通过 `go test ./...`、`go vet ./...`、gofmt 和 `git diff HEAD --check`；审批与 Handler 测试均使用独立 SQLite 内存数据库。
- 已验证真实 SQLite 进程的 `GET /healthz`、空审批与工单列表，以及 `POST /ag-ui` 完整有序事件流和中文文本增量；临时进程与文件已清理。
- 当前没有配置真实 PostgreSQL DSN 执行集成测试，不能声称 PostgreSQL 实库验证通过。
- 真实模型供应商及密钥、最新真实模型 Web 与 Admin 浏览器闭环仍未验证，不能沿用早期 Mock 或假服务结果宣称通过。
- 真实后台认证、租户隔离、生产级授权与审计、企业知识库和 RAG 尚未接入。

### 环境注意事项

- Go 已安装在 `C:\Program Files\Go\bin`，但当前 Shell 的 PATH 可能未包含它。
- Docker Desktop 可用；需要时将 `C:\Program Files\Docker\Docker\resources\bin` 加入 PATH，以便 Compose 和凭据助手正常工作。
- pnpm 未全局安装，可通过 `npx --yes pnpm@11.15.1 <命令>` 执行；Web 和 Admin 依赖均已安装。
- 当前 npm 默认源为 `https://registry.npmmirror.com`，依赖解析可使用官方源 `https://registry.npmjs.org`。
- goctl 通过根脚本中的 `go run github.com/zeromicro/go-zero/tools/goctl@v1.10.2` 执行，不再采用手写 Handler、Logic 和 Routes 结构。
- 不遗留临时开发服务器、无头浏览器、验证脚本或后台进程；用户自行启动的开发服务不在自动清理范围内。

## 下一步

第 21 步已在本地 `main` 合并，等待用户明确指示后再推送远端。后续功能方向是企业知识库、RAG 与智能工单能力深化；真实 PostgreSQL 实库验证和真实供应商浏览器闭环应在具备相应环境后补做。
