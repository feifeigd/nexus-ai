import { describe, expect, it } from "vitest";

import { GET } from "./route";

describe("Copilot Runtime", () => {
  it("advertises the default agent through the discovery endpoint", async () => {
    const response = await GET(
      new Request("http://localhost/api/copilotkit/info"),
    );

    expect(response.status).toBe(200);
    expect(await response.text()).toContain("default");
  });
});
