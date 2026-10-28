import { afterEach, describe, expect, it, vi } from "vitest";

import { getApiHealth } from "./api";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("getApiHealth", () => {
  it("returns the go-zero health response when the API is available", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ status: "ok", service: "nexus-api" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(getApiHealth()).resolves.toEqual({
      status: "ok",
      service: "nexus-api",
    });
    expect(fetchMock).toHaveBeenCalledWith("http://127.0.0.1:8080/healthz", {
      signal: expect.any(AbortSignal),
    });
  });

  it("returns null when the API is unavailable", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("connection refused")));

    await expect(getApiHealth()).resolves.toBeNull();
  });
});
