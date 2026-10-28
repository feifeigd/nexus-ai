import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import Home from "./page";

describe("Home", () => {
  it("renders the NexusAI ready state", () => {
    const html = renderToStaticMarkup(<Home />);

    expect(html).toContain("NexusAI");
    expect(html).toContain("用户端 Web 骨架已就绪。");
  });
});
