# NexusAI 项目上下文

## 项目目标

NexusAI 是暂定名称。项目首先用于全栈面试作品，随后逐步演进为企业级 AI 应用开发框架，并内置多租户“企业知识库与智能工单”SaaS 参考应用。项目不是 MaaS；长期方向是 AI 应用框架 / AI PaaS。

## 已确定的技术边界

- go-zero 是唯一企业业务与 Agent 后端，第一阶段采用模块化单体，不提前拆微服务。
- Next.js 用于用户端 Web；Ant Design Pro 用于独立管理后台。
- CopilotKit 负责 AI 前端交互；Copilot Runtime 仅是 Next.js 内的轻量 TypeScript 协议适配层，不重复实现企业业务。
- AG-UI 负责 Agent 事件通信，A2UI 负责受控动态 UI；普通业务仍使用 REST。
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

当前分支：`feat/01-go-zero-health`。

截至 2026-10-28，尚无 Git 提交，第一步正在等待用户 review。

已完成：

1. pnpm Monorepo 根结构与根级 API 脚本。
2. go-zero 最小服务及 `GET /healthz`。
3. `services/api/api/health.api` 契约。
4. Handler 单元测试。
5. 已验证 `go test ./...`、`go vet ./...`、gofmt 和真实 HTTP 请求。
6. 真实响应为 `{"status":"ok","service":"nexus-api"}`。

goctl 因依赖下载超时未用于第一步代码生成，当前采用与 go-zero 标准结构一致的手写最小实现，保留 `.api` 契约。

项目已从原任务工作区移动到 `H:\git\nexus-ai`，未携带工具链缓存、编译产物或 `node_modules`。

## 下一步

第二步：配置并验证 PostgreSQL、pgvector、Redis。

当前环境曾检测到 Docker 不在 PATH，因此容器步骤需要先确认 Docker 是否已安装；如果无法运行容器，必须明确标记验证受阻，不能声称通过。

后续步骤依次为：Next.js 骨架、前后端 REST 连通、Ant Design Pro、CopilotKit Runtime、go-zero Mock AG-UI 流、完整 AI 流式闭环。
