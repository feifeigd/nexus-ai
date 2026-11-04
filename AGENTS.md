# NexusAI 项目上下文

## 项目目标

NexusAI 是暂定名称。项目旨在构建企业级 AI 应用开发框架，并内置多租户“企业知识库与智能工单”SaaS 参考应用。项目不是 MaaS；长期方向是 AI 应用框架 / AI PaaS。

## 已确定的技术边界

- go-zero 是唯一企业业务与 Agent 后端，第一阶段采用模块化单体，不提前拆微服务。
- Ent 是唯一数据访问框架：`data.NewData` 统一拥有数据库连接，Store 复用其 Ent Client，方言差异封装在数据层；Ent 实体不得直接作为前端 API 契约，权限、租户隔离、审批、审计和 Agent 编排仍由 go-zero 负责，不引入第二套 ORM 或并行数据访问体系。
- Next.js 用于用户端 Web；Ant Design Pro 用于独立管理后台。
- CopilotKit 负责用户端和管理后台的 AI 前端交互；Web 使用 Next.js 内嵌 Copilot Runtime，Admin 使用 `apps/agent-runtime` 独立轻量 TypeScript 协议适配服务，两者都不重复实现企业业务。
- 管理后台必须提供 AI 聊天助手，支持通过聊天查询数据、辅助填写和触发后台操作；模型只能提议工具调用，go-zero 负责权限、租户隔离、参数校验、人工审批和审计。
- AG-UI 负责 Agent 事件通信，A2UI 负责受控动态 UI；普通业务仍使用 REST。
- 后续接入自带交互界面的 MCP 工具时，可按需引入 MCP Apps（MCP-UI），作为隔离的工具面板；它不替代 AG-UI、A2UI 或固定业务页面，UI 触发的业务操作仍必须由 go-zero 鉴权、校验、审批和审计。
- 管理后台必须支持通过 `@umijs/max-plugin-openapi` 从 NexusAI 的真实 OpenAPI JSON 生成 TypeScript API 代码；可通过 `OPENAPI_SCHEMA_PATH` 指定 Schema，不能默认依赖上游示例定义。
- PostgreSQL、pgvector、Redis 用于数据、向量检索和缓存；Docker Compose 用于本地环境。
- 后续使用 Expo + React Native 扩展安卓端。
- 当前不引入 NestJS、Python Agent 服务、Kubernetes 或复杂微服务体系。
- 后期规划：RESTful 业务继续用 go-zero，Agent 进程引入 go-micro（Go 微服务框架），运行环境基于 Istio（服务网格）。此为后期微服务化方向，当前阶段 Agent 仍在 go-zero 内、不拆微服务、不引入 Kubernetes；go-zero 与 go-micro 并存、各有分工，落地时机届时再定。
- 后期规划：可能引入无服务架构（Serverless），使用 MicroVM（微虚拟机）与沙箱承载隔离运行环境。此为后期方向，落地时机与具体方案届时再定。
- Eino 当前暂不引入：现阶段的模型接入与流式协议转换不需要额外 Agent 编排抽象。后续进入企业知识库、RAG、复杂工具调用或多步骤 Agent 工作流时重新评估；若引入，应作为 go-zero 内 Agent `Runner` 的一种编排实现，不替代 go-zero，也不承接权限、租户隔离、人工审批或审计。
- 工具权限、租户隔离、参数校验、人工审批和审计必须以 go-zero 后端为权威，不能依赖模型输出或前端状态。

## 依赖升级约束

- 当前 `@copilotkit/runtime@1.83.0` 将 `@ag-ui/client`、`@ag-ui/core`、`@ag-ui/encoder` 固定为 `1.0.2`；`@copilotkit/core@1.83.0` 将 `@ag-ui/client` 固定为 `1.0.2`；`@copilotkit/react-core@1.83.0` 将 `@ag-ui/client`、`@ag-ui/core` 固定为 `1.0.2`。
- 上述 CopilotKit 包仍固定 AG-UI `1.0.2` 时，禁止单独升级 AG-UI 依赖，也不要通过 pnpm overrides 等方式强行绕过版本约束。只有这些相关 CopilotKit 包全部升级、且其依赖声明共同兼容目标 AG-UI 版本后，才可同步升级 AG-UI。
- AG-UI 或 CopilotKit 升级前必须核对实际包依赖声明；升级后必须确认锁文件不保留互不兼容的 AG-UI 重复版本，并执行 Web 测试、Lint 和生产构建，未实际执行的验证不得写成通过。
- OpenAI 兼容协议当前继续采用 go-zero 内手写 Chat Completions 客户端，不在第 14 步混入 SDK 替换。手写范围只维护当前文本与函数调用子集，不继续扩展 Responses API、多模态或通用重试等完整 SDK 能力；OpenAI 协议类型与 SSE 解析必须留在 `OpenAIRunner` 内部。后续迁移官方 `openai-go` 时应作为独立小增量，保持 `Runner` 接口、AG-UI 事件和业务行为不变，并复用假服务测试覆盖文本流、工具参数分片、HTTP 错误、异常 EOF、非法 JSON、中途错误和取消。
- 已知问题（待修复，2026-11-03/04）：Web 端 CopilotChat 发送按钮 disabled，根因是 agent discovery（`GET /api/copilotkit/info`）在 Next.js App Router SSR 下未自动触发。有问题的版本组合：Next.js `16.4.3` + `@copilotkit/react-core@1.83.0` + `@copilotkit/runtime@1.83.0` + `@ag-ui/client@1.0.2`。Admin（Umi SPA 纯客户端）同版本正常，仅 Web（SSR）异常；后端链路正常（agent run 端点 curl 实测可流式返回），仅前端 UI 发送按钮。已排查：`@ag-ui/client` 重复版本已排除（1.0.3 为孤儿目录）；`styles.css` 移到 client 组件恢复了 Lit 初始化；但 `ssr: false` 也让 CopilotChat 客户端不渲染（textarea 不出现），说明非简单 SSR 边界问题。后续方向：升级 CopilotKit 观察是否修复，或查 CopilotKit 官方 issue（`useLayoutEffect` 时序、`onError` 错误码等）。已降级到 1.82.0 验证，失败——1.82.0 引入 `Can't resolve 'd3-array'` 错误（streamdown → mermaid → d3 依赖链在 pnpm 下解析失败），web 整个 500，比 1.83.0 更严重，已回退（revert 3a3ed27）。结论：1.82.0 与 1.83.0 在 Next.js 16 下均有问题，需查 CopilotKit 官方兼容性。已提 issue：https://github.com/CopilotKit/CopilotKit/issues/8192，最小复现 repo：https://github.com/feifeigd/copilotkit-ssr-repro。

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

截至 2026-11-03，第 1 至 30 步均已完成、合并到 `main` 并推送到 `origin/main`。第 22 步合并提交为 `a44ad27`，第 23 步合并提交为 `4a39302`（含 `c6c4710` 可折叠检索结果卡片），第 24 步为 `f002171`（检索距离阈值），第 25 步为 `22d9a69`（答案级引用校验），第 26 步为 `0c5568f`（检索重排序），第 27 步为 `bc307ef`（答案级事实核验），第 28 步为 `806c5e2`（PostgreSQL + pgvector 实库集成测试），第 29 步为 `feeab12`（语义级事实核验），第 30 步为 `4f05e85`（BM25 关键词打分融合），另有 `a8efcef` 强制 LF 换行配置；本地与远端 `main` 已同步。此后依赖升级到 CopilotKit `1.83.0`（仍锁 `@ag-ui/*` 为 `1.0.2`），dependabot 新增 `@ag-ui/*` ignore 规则，CI 在 Go 测试前新增 `pnpm generate:wire`；相关特性分支已随合并清理。存在未跟踪的 `services/api/__debug_bin.exe*` 调试文件，不要提交。

此后（2026-11-04/05）在分支 `feat/tenant-isolation` 完成认证与租户隔离，已合并回 `main`（合并提交 `02385e1`，fast-forward）并推送 `origin/main`，分支已清理：账号密码登录 + JWT 鉴权 + staff 角色校验（Web 登录注册、Admin 鉴权头、`auth.TenantIDFromContext` 统一租户解析）；对话 token 记账带租户归属（staff 可看全部、普通用户看自己）；数据层租户隔离 `tenant_id` 覆盖 `users`、`token_usages`、`documents`、`chunks`、`approvals`、`tickets` 六张表（迁移 `007_tenant_isolation.sql`），知识库检索、文档列表/详情/删除与审批工单均按租户过滤。存量数据迁移后 `tenant_id` 默认空串（无租户），staff 全量可见、普通用户不可见，是否做归属迁移待定。

### 当前已具备能力

- pnpm Monorepo、Next.js 用户端、Ant Design Pro 管理后台、go-zero API，以及 Docker Compose 中的 PostgreSQL、pgvector 和 Redis 本地基础设施。
- Web 内嵌 CopilotKit Runtime 与 Admin 独立 `apps/agent-runtime` 均已接入 go-zero AG-UI SSE；Agent 支持 Mock 和 OpenAI 兼容 Chat Completions 流式模型。
- go-zero 提供服务端白名单 `create_ticket` 工具提议、审批列表、批准和拒绝接口；批准后才创建工单，重复决策返回冲突。
- Admin 提供 AI 聊天、工具审批和工单页面；工单状态按 `open → in_progress → resolved` 顺序流转，不支持删除、跳级或重新打开。
- 知识库支持纯文本文档导入、Unicode 安全分块、1536 维向量生成、文档与分块事务写入，以及文档列表、详情和删除；删除文档时通过外键级联删除分块。
- Agent 已注册只读 `search_knowledge` 工具：第一次模型请求决定检索，go-zero 生成查询向量并通过 pgvector 按余弦距离检索、过滤超过 `0.5` 阈值的低相关分块、再按 BM25 关键词分数（bigram 词元 + 候选池内局部 IDF）加权重排序，第二次模型请求依据工具结果生成流式回答；go-zero 还会校验回答中的 `documentId` 引用是否来自本次检索结果，并按字面重叠对回答逐句做来源支撑校验，开启 `NEXUS_KNOWLEDGE_SEMANTIC_CHECK` 时改用 NLI/LLM 语义判定（单次非流式批量请求），未开启时回退到字面 bigram 启发式，不可验证或无支撑时在答案末尾追加告警并写结构化日志。该工具不需要人工审批。
- Admin 为 `search_knowledge` 提供专用检索结果卡片，展示命中文档 ID、分块序号、正文与余弦距离，并与模型最终回答分离；解析、空结果与异常结果均有前端测试覆盖。
- 对话与嵌入模型可分别配置 OpenAI 兼容提供商、地址、密钥和模型；密钥只注入 go-zero，不下发到 Runtime 或浏览器。
- `.api` 契约可生成真实 OpenAPI JSON，并通过 `@umijs/max-plugin-openapi` 生成 Admin TypeScript Service；生成文件按 `.gitignore` 规则不纳入 Git，不手动修改。
- 数据访问已迁移到 Ent，Approval、Ticket、Document 与 Chunk Schema 位于 `services/api/internal/data/ent/schema`；Ent 生成代码同样不纳入 Git。
- go-zero API 已迁移到 `goctl api go --api api/nexus.api --dir .` 标准结构，根命令 `pnpm generate:api` 固定使用 goctl `v1.10.2`。
- 当前架构与知识库两段式问答流程分别记录在 `docs/architecture.md` 和 `docs/knowledge-qa-flow.md`。

### 数据源与代码生成约束

- `data.NewData` 是进程唯一的数据库连接所有者，统一负责驱动校验、连接池、Ping、迁移、日志和关闭；Store 必须复用其 Ent Client，不得单独打开或关闭数据库。
- 默认 `services/api/etc/nexus-api.yaml` 使用 SQLite 共享内存 DSN，服务重启后数据清空；设置 `NEXUS_POSTGRES_DSN` 后统一切换到 PostgreSQL。
- 真实 PostgreSQL + pgvector 集成测试由 `NEXUS_TEST_POSTGRES_DSN` 门控（未设置则跳过）；每个用例通过 `internal/testpg` 创建独立临时数据库，避免共享库下的检索串扰。
- 已移除 `MemoryStore`，内存开发和测试也通过 SQLite DSN，不再维护另一套内存业务实现。
- SQLite 使用纯 Go `modernc.org/sqlite`，配置中的 Ent dialect 为 `sqlite3`；PostgreSQL 使用版本化 SQL 迁移，SQLite AutoMigrate 使用 Ent `Schema.Create`。
- PostgreSQL 审批与状态更新使用事务和 `FOR UPDATE`；SQLite 跳过 `FOR UPDATE`，方言差异只保留在 EntStore 的锁定查询方法中。
- 知识向量固定为 1536 维；PostgreSQL 使用 pgvector 和 HNSW 余弦索引。SQLite 可保存向量用于开发测试，但不实现相似度检索，不得添加与生产行为不同的本地检索替代实现。
- 修改 `.api` 后运行 `pnpm generate:api`；修改 Ent Schema 后运行 `pnpm generate:ent`；修改 Wire 后运行 `pnpm generate:wire`；需要同步 Admin Client 时运行 `pnpm generate:contracts`。不要手动编辑生成文件。

### 最近验证记录

- 第 22 步删除文档增量已执行并通过 `go test ./...`、`go vet ./...`、gofmt、Admin Biome 与 TypeScript 检查、`pnpm check:contracts` 和 `git diff HEAD --check`。
- Admin 全量测试共 10 个测试文件、47 项用例通过，其中知识库页面 6 项用例通过。
- 知识删除测试使用独立 SQLite 内存数据库，已验证文档与两个分块级联删除、重复删除返回 404，以及 Handler 不泄露内部数据库错误。
- 第 23 步检索结果卡片增量已执行并通过 Admin 测试（11 个测试文件、59 项用例，含 `KnowledgeSearchToolResult` 与 `AiChat` 用例）、Biome（138 个文件无问题）与 TypeScript 检查。
- 依赖升级验证已执行并通过：runtime/web/admin 的 test、lint、build 全部通过（web 5 项、admin 59 项测试），锁文件无互不兼容的 AG-UI 重复版本（client/core/encoder/proto 均 1.0.2）。
- 第 24 步检索距离阈值增量已执行并通过 `go test ./...`、`go vet ./...` 与 gofmt；新增 `TestSearchFiltersByDistanceThreshold` 覆盖阈值边界与空结果。阈值来源为常量 `SearchDistanceThreshold = 0.5`（余弦距离，约相似度 0.5 / 夹角 60°），不是配置项。
- 第 25 步答案级引用校验增量已执行并通过 `go test ./...`、`go vet ./...`、gofmt 与 `git diff HEAD --check`；新增 `TestValidateSearchAnswer`（6 项）、`TestOpenAIRunnerAppendsAnswerValidationNote` 与 `TestOpenAIRunnerRejectsAnswerValidationWithoutExecute`。校验通过 `Tool.ValidateAnswer` 钩子实现，知识域逻辑留在 `knowledge` 包，`agent` 包不反向依赖。未执行真实模型浏览器验收，不能标记为已通过。
- 第 26 步检索重排序增量已执行并通过 `go test ./...`、`go vet ./...`、gofmt 与 `git diff HEAD --check`；新增 `rerank.go` 规则重排序（`SearchRecallFactor=4` 放大召回 + 关键词覆盖率加权）与 7 项测试（含 `TestRerankPromotesKeywordMatch`、`TestSearchReranksByKeywordCoverage`）。重排序为纯规则实现，未引入交叉编码器等外部依赖。未执行真实模型浏览器验收，不能标记为已通过。
- 第 27 步答案级事实核验增量已执行并通过 `go test ./...`、`go vet ./...`、gofmt 与 `git diff HEAD --check`；新增 `factcheck.go` 逐句字面支撑校验（bigram 重叠启发式）与 10 项测试。校验为字面重叠启发式而非语义事实核验，未引入 NLI/LLM 判定。未执行真实模型浏览器验收，不能标记为已通过。
- 第 28 步 PostgreSQL + pgvector 实库集成测试已执行并通过：用 Docker Compose 起 pg17+pgvector 真实库，`go test ./... -count=1` 全绿，三个 DSN 门控测试（`TestSearchPipelineAgainstPostgres`、`TestKnowledgeStorePostgres`、`TestEntStorePersistsApprovalAndTicketAcrossReopen`）真正运行并通过。新增 `internal/testpg` 为每个用例创建独立临时数据库，解决跨包共享库的检索串扰。未执行真实嵌入模型与真实供应商浏览器回归，不能标记为已通过。
- 第 29 步语义级事实核验增量已执行并通过 `go test ./...`、`go vet ./...` 与 gofmt；新增 `support_judge.go`（`OpenAISupportJudge` 单次非流式批量判定）与 5 项测试，`factcheck_test.go` 新增 3 项 judge 覆盖测试。语义判定由 `knowledge.SupportJudge` 接口定义、`agent` 实现、`providers` 在 `NEXUS_KNOWLEDGE_SEMANTIC_CHECK` 开启时接线，默认关闭以控制额外模型调用成本与延迟；未开启时回退到字面 bigram 启发式。判定只经假服务（httptest）验证，未对真实供应商验证语义判定质量，不能标记为已通过。
- 第 30 步 BM25 关键词打分融合增量已执行并通过 `go test ./...`、`go vet ./...` 与 gofmt；用 BM25（bigram 词元、候选池内局部 IDF、长度归一化，`k1=1.2`、`b=0.75`）替换第 26 步的关键词覆盖率，归一化到 `[0,1]` 后与余弦距离融合，新增 4 项 BM25 测试（词频、IDF 罕见词、归一化、空查询/空池）并重命名集成测试。概念说明见 `docs/bm25.md`。未重跑真实 pgvector 集成测试（DSN 门控），不能标记为已通过。
- 第 23 步未执行真实浏览器检索验收，不能标记为已通过。
- 第 22 步未执行真实浏览器删除验收，不能标记为已通过。
- PostgreSQL + pgvector 实库的余弦检索、距离阈值过滤、重排序与答案校验已通过 Docker Compose 真实库集成测试验证；真实嵌入模型语义检索已通过单样本观测跑通，真实供应商浏览器回归仍未执行，不能标记为已通过。
- 未执行完整的真实供应商 Web 与 Admin 浏览器回归，单次手工模型回答不能替代检索事件、引用内容和多页面闭环验收。
- 真实后台认证、租户隔离、生产级授权与审计仍未接入；语义级事实核验已实现但默认关闭（`NEXUS_KNOWLEDGE_SEMANTIC_CHECK` 开启后改用 NLI/LLM 判定，未开启时沿用字面 bigram 启发式），且仅对真实供应商做过单样本判定质量观测，未做量化验证。
- 第 29/30 步真实供应商后端回归：新增门控测试 `TestRealModelKnowledgePipeline`（`NEXUS_TEST_REAL_MODEL=1`，默认跳过），用真实嵌入模型（DashScope text-embedding-v4）与对话模型（DeepSeek deepseek-v4-flash）端到端跑通入库/检索/语义核验：查询「蓝雀缓存雪崩如何应对？」命中正确分块、余弦距离 0.1133，语义判定只把无关句标为无支撑。这是单样本观测而非量化质量/延迟/成本基准，未覆盖两段式流式回答与浏览器闭环，不能标记为已通过。
- 真实供应商 AG-UI 端到端回归（2026-11-03）：经 go-zero `/ag-ui` 流式端点用真实模型实测，RUN_STARTED → search_knowledge 工具调用 → pgvector 检索 → 两段式流式回答 → 语义核验 → RUN_FINISHED 完整跑通。查询「蓝雀缓存雪崩如何应对？」命中《故障排查》《雪崩与击穿应对》《服务架构》等正确文档，余弦距离 0.12～0.26；回答准确覆盖随机过期时间、熔断限流、TTL、LRU 淘汰、主从异步并带引用。观测到语义核验误报：《配置与运维》检索结果已含「告警应覆盖内存接近上限、命中率骤降和主从同步延迟」、《服务架构》已含「主从复制默认异步」，LLM 判定仍把答案对应句标为无支撑（precision 问题，未量化）。另注意 Windows Git Bash 下 `curl -d` 内联中文会乱码（模型误读为「冰雪天气」），改用 node 标准 UTF-8 重发正常，浏览器不受影响。仍未做真浏览器 UI 验收（检索卡片渲染、多页面闭环），不能标记为已通过。
- 语义核验误报修复（2026-11-03）：`answerSupportNote` 改为「字面 bigram 先行 + LLM 语义兜底」——字面重叠 ≥0.3 的句子直接判支撑（跳过 LLM），只有字面重叠不足的句子才交 LLM 判定同义改写或编造，消除「原文照抄/高度重叠句被 LLM 误判无支撑」的误报。新增 `TestAnswerSupportNoteSkipsJudgeForLiteralMatches` 并更新 2 个受影响测试，`go test ./...`、`go vet ./...`、gofmt、`git diff HEAD --check` 全部通过。真实模型复测确认原误报句（「告警应覆盖内存接近上限…」「主从复制默认异步…」）不再告警；但发现新问题：模型用 markdown 组织回答时，标题/连接语等组织性句子被纳入核验并标无支撑，降低告警信噪比，待单独过滤。
- 语义核验组织语言过滤（2026-11-03）：新增 `isOrganizationalSentence` 过滤 markdown 标题（`#` 开头）与冒号结尾的引导/连接语，避免模型组织性语言被纳入核验产生噪音。新增 `TestAnswerSupportNoteSkipsOrganizationalSentences`，`go test ./...`、`go vet ./...`、gofmt、`git diff HEAD --check` 全部通过。真实模型复测确认标题/连接语不再告警；但结语/免责声明/提醒类元语言（如「以上内容来自知识库检索结果」「如需…建议…」）仍被标记，需句子级分类才能稳定过滤，暂记为已知局限。
- 结语过滤 LLM 方案尝试失败（2026-11-03，二分类与三分类均失败）：① 二分类 prompt 引导 LLM 把「组织性/结语/免责/建议」句判为 supported，门控测试实测 `note=""`——编造句「数据库主从复制采用半同步复制模式」也被判 supported，引入漏报；② 显式三分类（supported/unsupported/not-a-claim）同样实测 `note=""`——编造句仍未被标记。两次均已回退。结论：LLM 无法可靠区分「组织语言」与「编造知识」，无论二分类还是三分类；结语/提醒类元语言过滤需非 LLM 的确定性方案（或独立句子分类模型），暂缓。
- 真浏览器 UI 验收（2026-11-03，Playwright + Chromium 自动化）：Admin AI 助手检索事件卡片（「知识检索结果 5 个分块」）与引用回答、知识库 5 篇文档列表、审批记录与批准（Popconfirm 确认）、工单显示、多页面闭环（AI 创建工单 → 审批批准 → 工单）全部实测通过。Web 首页加载与内嵌 Runtime info 正常，但 Web 端 CopilotChat 发送按钮 disabled（Enter 换行、发送按钮不可点），未能实测 Web 流式回答，待查（疑似 CopilotKit v2 useSingleEndpoint 模式下的 UI 状态或 web 内嵌 Runtime 配置）。另 Web page.tsx 的「Mock Agent / MOCK · STREAMING」文案已过时，与真实模型不符。
- 交叉编码器重排序评估（2026-11-04）：评估结论为当前不引入，详见 `docs/reranker-evaluation.md`。现有 pgvector 余弦召回 + BM25 关键词融合已零成本命中正确文档（距离 0.12～0.26），交叉编码器（reranker）的 15～25% 排序精度提升不足以覆盖其 100～500ms 延迟与每次查询的 API 成本；且 reranker 只能精排候选池、不能救回漏召回。重新评估条件：知识库规模显著上升 + 有量化数据证明排序精度成为瓶颈。若引入：保留召回 + 规则重排，加交叉编码器对 Top N 精排，优先用 API reranker（复用 DashScope）或本地 bge-reranker。
- 知识库检索量化基准（2026-11-04）：新增门控测试 `TestRealModelRetrievalBenchmark`（`NEXUS_TEST_REAL_MODEL=1`），10 题覆盖多文档，命中率 10/10=100%、平均检索延迟 470ms；排序质量有提升空间（3 题正确答案非首位：HIT@4×2、HIT@2×1），印证交叉编码器评估里「精排是排序精度问题、非召回问题」的判断。另发现现有库文档为 7 篇（比预期 5 篇多 2 篇，疑为回归测试重复入库「故障排查」）。
- 知识库端到端问答量化基准（2026-11-04）：新增门控测试 `TestRealModelE2EBenchmark`（`NEXUS_TEST_REAL_MODEL=1`），5 题走完整两段式问答（模型决定检索 → 工具执行 → 生成回答 → 语义核验），平均端到端延迟 8.04s（范围 2.4～18.4s，语义核验开启），每问模型调用 search_knowledge 2 次。token 成本尚未量化（需在 OpenAIRunner 记录 usage）。
- 知识库问答 token 成本量化（2026-11-04）：`OpenAIRunner` 与 `OpenAISupportJudge` 新增 `Usage` 记录（请求开 `stream_options.include_usage`），基准每问输出对话模型 token（2 次调用 1460～2654）+ 语义核验 token（745～1935），合计约 2200～4500 token/问；嵌入 token 未计（查询约十几 token，可忽略）。成本需按实际模型单价折算。
- 数据层租户隔离（2026-11-04/05）：`tenant_id` 覆盖 `users`、`token_usages`、`documents`、`chunks`、`approvals`、`tickets` 六张表（迁移 `007_tenant_isolation.sql` 幂等加列与索引），知识库 CreateDocument/List/Get/Delete/SearchSimilar 与审批 propose/List/ListTickets/UpdateTicketStatus/Approve/Reject 均按租户过滤、跨租户操作返回 not found。新增 `TestKnowledgeStoreTenantIsolation`、`TestEntStoreTenantIsolation`。`go build ./...`、`go vet ./...`、`go test ./...`（含 `NEXUS_TEST_POSTGRES_DSN` 门控的 `TestSearchPipelineAgainstPostgres`、`TestKnowledgeStorePostgres`、`TestEntStorePersistsApprovalAndTicketAcrossReopen` 三个真实 pgvector 集成测试实际运行通过）、gofmt、`git diff HEAD --check`、Admin lint（Biome+tsc）均通过；分支已合并 main（`02385e1`）并推送。

### 环境注意事项

- Go 已安装在 `C:\Program Files\Go\bin`，但当前 Shell 的 PATH 可能未包含它。
- Docker Desktop 可用；需要时将 `C:\Program Files\Docker\Docker\resources\bin` 加入 PATH，以便 Compose 和凭据助手正常工作。
- pnpm 未全局安装，可通过 `npx --yes pnpm@11.15.1 <命令>` 执行；Web 和 Admin 依赖均已安装。
- 当前 npm 默认源为 `https://registry.npmmirror.com`，依赖解析可使用官方源 `https://registry.npmjs.org`。
- goctl 通过根脚本中的 `go run github.com/zeromicro/go-zero/tools/goctl@v1.10.2` 执行，不再采用手写 Handler、Logic 和 Routes 结构。
- 不遗留临时开发服务器、无头浏览器、验证脚本或后台进程；用户自行启动的开发服务不在自动清理范围内。

## 下一步

- 排查 Web 端 CopilotChat 发送按钮 disabled 并补测 Web 流式回答（版本组合已记录，后期修复）。
- 过滤语义核验中的结语/提醒类元语言（需非 LLM 确定性方案）。
- 三方登录（OAuth）：需先选平台（GitHub / 微信 / 企业微信 / 钉钉等）并到对方平台申请 App Key/Secret；未接入前仅账号密码登录。认证、token 记账与数据层租户隔离（六张表 tenant_id）均已合并 main。
