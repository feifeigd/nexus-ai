# NexusAI 项目上下文

## 项目目标

NexusAI 是暂定名称。项目旨在构建企业级 AI 应用开发框架，并内置多租户“企业知识库与智能工单”SaaS 参考应用。项目不是 MaaS；长期方向是 AI 应用框架 / AI PaaS。

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

当前分支：`feat/04-web-api-rest`。

截至 2026-10-28，第 1 至 3 步已提交；第 4 步已完成并等待用户 review，尚未提交。

### 已完成并提交

1. 第 1 步提交 `87788d5 feat: add go-zero health service`：pnpm Monorepo 根结构、go-zero 最小服务、`GET /healthz`、`.api` 契约和 Handler 单元测试。已验证 Go 测试、静态检查、格式和真实 HTTP 响应。
2. 第 2 步提交 `30e763e feat: add postgres and redis infrastructure`：Docker Compose 配置 PostgreSQL 17、pgvector 和 Redis 7.4。已验证容器健康检查、PostgreSQL 17.11、pgvector 0.8.7、真实向量运算及 Redis 读写。
3. 第 3 步提交 `302f82e feat: add nextjs web skeleton`：Next.js 16.4.3、React 19.3.0、TypeScript、Tailwind CSS、ESLint、NexusAI 最小首页、页面测试和根级 Web 脚本。

### 当前待 review 增量

第 4 步：前后端 REST 连通。

- Next.js 首页通过服务端 `fetch` 调用 go-zero `GET /healthz`，使用 `Suspense` 保留静态页面外壳。
- 健康请求使用 `NEXUS_API_INTERNAL_URL`，默认地址为 `http://127.0.0.1:8080`，超时为 3 秒。
- 页面区分 API 已连接和未连接状态，不在 TypeScript 层重复实现业务逻辑。
- 已验证 3 个用户端测试、ESLint、Go 测试与静态检查、TypeScript 和生产构建。
- 已同时启动前后端完成真实联调：后端响应符合契约，首页返回 HTTP 200，并显示“go-zero API 已连接”和“nexus-api · ok”。
- 临时前后端服务已停止，端口 7100 和 8080 均已释放。

### 环境注意事项

- Go 已安装在 `C:\Program Files\Go\bin`，但当前 Shell 的 PATH 可能未包含它。
- Docker Desktop 可用；需要时将 `C:\Program Files\Docker\Docker\resources\bin` 加入 PATH，以便 Compose 和凭据助手正常工作。
- pnpm 未全局安装，可通过 `npx --yes pnpm@11.15.1 <命令>` 执行；Web 依赖已经安装。
- goctl 因依赖下载超时未用于第 1 步代码生成，当前采用与 go-zero 标准结构一致的手写最小实现，保留 `.api` 契约。
- 第 2 步验证容器已停止并删除，命名数据卷保留；当前无项目相关临时服务或后台进程。

## 下一步

第五步：Ant Design Pro 管理后台骨架。

后续步骤依次为：CopilotKit Runtime、go-zero Mock AG-UI 流、完整 AI 流式闭环。
