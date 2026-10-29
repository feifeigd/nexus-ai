import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";

import Home from "./page";

vi.mock("./chat-panel", () => ({
  default: () => <div data-testid="chat-panel" />,
}));

describe("Home", () => {
  it("renders the NexusAI chat experience", () => {
    const html = renderToStaticMarkup(<Home />);

    expect(html).toContain("NexusAI");
    expect(html).toContain("流式聊天链路已接通");
    expect(html).toContain('data-testid="chat-panel"');
  });
});
