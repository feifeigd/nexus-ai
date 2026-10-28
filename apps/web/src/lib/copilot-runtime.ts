import { HttpAgent } from "@ag-ui/client";
import {
  CopilotRuntime,
  InMemoryAgentRunner,
} from "@copilotkit/runtime/v2";

// 发给 services/api 进程
export const NEXUS_AG_UI_AGENT_URL =
  process.env.NEXUS_AG_UI_AGENT_URL ?? "http://127.0.0.1:8080/ag-ui";

export function createNexusRuntime() {
  return new CopilotRuntime({
    agents: {
      default: new HttpAgent({ url: NEXUS_AG_UI_AGENT_URL }),
    },
    runner: new InMemoryAgentRunner(),
    forwardHeaders: {
      allow: ["authorization", "x-tenant-id"],
    },
  });
}
