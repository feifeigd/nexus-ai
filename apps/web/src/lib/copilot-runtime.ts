import { HttpAgent } from "@ag-ui/client";
import {
  CopilotRuntime,
  InMemoryAgentRunner,
} from "@copilotkit/runtime/v2";

// 发给 services/api 进程
// 指定 CopilotKit Runtime 转发 AG-UI 请求的 go-zero 地址。
export const NEXUS_AG_UI_AGENT_URL =
  process.env.NEXUS_AG_UI_AGENT_URL ?? "http://127.0.0.1:8080/ag-ui";

// 为什么中间加 Runtime
// 1. 适配 CopilotKit 前端
//   go-zero 只提供 AG-UI 协议；CopilotKit 前端还需要 Agent 发现、运行 Agent 等 Runtime 契约，例如：
//     /api/copilotkit/info
//     /api/copilotkit/agent/default/run
// 2. 浏览器保持同源
//   浏览器只访问 Next.js, 不需要直接访问 8080，可以减少 CORS 配置。
// 3. 统一注册 Agent
// 4. 控制转发信息
//   当前只允许转发：
//     authorization
//     x-tenant-id
//   而不是把浏览器请求头全部交给 Agent 后端。

// Runtime 不做什么
//   它不负责：
//     企业业务逻辑。
//     权限与租户隔离的最终校验。
//     参数校验和审计。
//     模拟或调用大模型。
//     这些仍然由 go-zero 负责。
//   所以，直接调用 go-zero 完全可以，适合测试和排查问题；Runtime 是为了让 CopilotKit 前端以统一、同源的方式接入。它类似一个翻译和转发层，不是重复开发一套后端。
export function createNexusRuntime() {
  return new CopilotRuntime({
    // 3. 统一注册 Agent
    agents: {
      // 当前注册了 default
      default: new HttpAgent({ url: NEXUS_AG_UI_AGENT_URL }),
      // 后续增加知识库、工单等 Agent 时，前端仍通过统一入口选择。
    },
    runner: new InMemoryAgentRunner(),
    forwardHeaders: {
      allow: ["authorization", "x-tenant-id"],
    },
  });
}
