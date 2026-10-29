import assert from "node:assert/strict";
import { createServer, type Server } from "node:http";
import { after, before, describe, it } from "node:test";
import { createNexusRuntimeListener } from "./runtime.js";

const AGENT_EVENTS = [
  { type: "RUN_STARTED", threadId: "thread-test", runId: "run-test" },
  {
    type: "TEXT_MESSAGE_START",
    messageId: "message-assistant",
    role: "assistant",
  },
  {
    type: "TEXT_MESSAGE_CONTENT",
    messageId: "message-assistant",
    delta: "独立 Runtime 已连接",
  },
  { type: "TEXT_MESSAGE_END", messageId: "message-assistant" },
  { type: "RUN_FINISHED", threadId: "thread-test", runId: "run-test" },
];

async function listen(server: Server) {
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  assert(address && typeof address === "object");
  return address.port;
}

async function close(server: Server) {
  await new Promise<void>((resolve, reject) => {
    server.close((error) => (error ? reject(error) : resolve()));
  });
}

describe("standalone Copilot Runtime", () => {
  let agentServer: Server;
  let runtimeServer: Server;
  let runtimeUrl: string;
  let upstreamBody = "";
  let upstreamAuthorization = "";
  let upstreamTenant = "";

  before(async () => {
    agentServer = createServer((request, response) => {
      let body = "";
      request.setEncoding("utf8");
      request.on("data", (chunk) => {
        body += chunk;
      });
      request.on("end", () => {
        upstreamBody = body;
        upstreamAuthorization = String(request.headers.authorization ?? "");
        upstreamTenant = String(request.headers["x-tenant-id"] ?? "");
        response.writeHead(200, {
          "Content-Type": "text/event-stream; charset=utf-8",
        });
        for (const event of AGENT_EVENTS) {
          response.write(`data: ${JSON.stringify(event)}\n\n`);
        }
        response.end();
      });
    });
    const agentPort = await listen(agentServer);

    runtimeServer = createServer(
      createNexusRuntimeListener(`http://127.0.0.1:${agentPort}/ag-ui`),
    );
    const runtimePort = await listen(runtimeServer);
    runtimeUrl = `http://127.0.0.1:${runtimePort}/api/copilotkit`;
  });

  after(async () => {
    await close(runtimeServer);
    await close(agentServer);
  });

  it("discovers the default agent", async () => {
    const response = await fetch(`${runtimeUrl}/info`);

    assert.equal(response.status, 200);
    assert.match(await response.text(), /default/);
  });

  it("forwards allowed headers and streams the AG-UI response", async () => {
    const response = await fetch(`${runtimeUrl}/agent/default/run`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "text/event-stream",
        Authorization: "Bearer test-token",
        "X-Tenant-Id": "tenant-test",
      },
      body: JSON.stringify({
        threadId: "thread-test",
        runId: "run-test",
        messages: [
          { id: "message-user", role: "user", content: "你好，Runtime" },
        ],
      }),
    });

    assert.equal(response.status, 200);
    assert.match(response.headers.get("content-type") ?? "", /text\/event-stream/);
    const stream = await response.text();
    assert.match(stream, /独立 Runtime 已连接/);
    assert.match(stream, /RUN_FINISHED/);
    assert.match(upstreamBody, /你好，Runtime/);
    assert.equal(upstreamAuthorization, "Bearer test-token");
    assert.equal(upstreamTenant, "tenant-test");
  });
});
