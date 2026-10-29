import { createServer } from "node:http";
import { createNexusRuntimeListener } from "./runtime.js";

const host = process.env.NEXUS_COPILOT_RUNTIME_HOST ?? "127.0.0.1";
const port = Number(process.env.NEXUS_COPILOT_RUNTIME_PORT ?? "4000");

if (!Number.isInteger(port) || port < 1 || port > 65535) {
  throw new Error("NEXUS_COPILOT_RUNTIME_PORT must be a valid TCP port");
}

const server = createServer(createNexusRuntimeListener());

server.listen(port, host, () => {
  console.log(`NexusAI Copilot Runtime listening on http://${host}:${port}`);
});

function shutdown() {
  server.close(() => process.exit(0));
}

process.once("SIGINT", shutdown);
process.once("SIGTERM", shutdown);
