import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import ChatPanel from "./chat-panel";

vi.mock("@copilotkit/react-core/v2", () => ({
  CopilotChat: ({ agentId }: { agentId?: string }) => (
    <div data-testid="copilot-chat" data-agent-id={agentId} />
  ),
}));

describe("ChatPanel", () => {
  it("connects the chat interface to the default agent", () => {
    const html = renderToStaticMarkup(<ChatPanel />);

    expect(html).toContain('data-testid="copilot-chat"');
    expect(html).toContain('data-agent-id="default"');
  });
});
