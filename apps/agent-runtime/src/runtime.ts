import { HttpAgent } from "@ag-ui/client";
import {
  CopilotRuntime,
  InMemoryAgentRunner,
} from "@copilotkit/runtime/v2";
import { createCopilotNodeListener } from "@copilotkit/runtime/v2/node";

// go-zero
export const COPILOT_BASE_PATH = "/api/copilotkit";
export const DEFAULT_AGENT_URL = "http://127.0.0.1:8080/ag-ui";

export function createNexusRuntime(
  agentUrl = process.env.NEXUS_AG_UI_AGENT_URL ?? DEFAULT_AGENT_URL,
) {
  return new CopilotRuntime({
    agents: {
      default: new HttpAgent({ url: agentUrl }),
    },
    runner: new InMemoryAgentRunner(),
    forwardHeaders: {
      allow: ["authorization", "x-tenant-id"],
    },
  });
}

export function createNexusRuntimeListener(agentUrl?: string) {
  return createCopilotNodeListener({
    runtime: createNexusRuntime(agentUrl),
    basePath: COPILOT_BASE_PATH,
    activateChannels: false,
  });
}
