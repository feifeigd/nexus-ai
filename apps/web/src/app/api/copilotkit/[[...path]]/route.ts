import { createCopilotRuntimeHandler } from "@copilotkit/runtime/v2";

import { createNexusRuntime } from "@/lib/copilot-runtime";

const handler = createCopilotRuntimeHandler({
  runtime: createNexusRuntime(),
  basePath: "/api/copilotkit",
});

export const GET = handler;
export const POST = handler;
export const PATCH = handler;
export const DELETE = handler;
