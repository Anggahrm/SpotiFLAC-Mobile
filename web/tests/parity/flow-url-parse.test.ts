import { afterEach, describe, expect, it, vi } from "vitest";
import { parseUrl } from "../../lib/api/client";

describe("flow-url-parse", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("returns parsed type and id for valid URL", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: { get: () => "application/json" },
        json: async () => ({ success: true, data: { type: "track", id: "abc123" } }),
      }),
    );

    const result = await parseUrl("https://open.spotify.com/track/abc123");
    expect(result).toEqual({ type: "track", id: "abc123" });
  });

  it("surfaces API error for non-OK JSON responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 400,
        headers: { get: () => "application/json" },
        json: async () => ({ success: false, error: "URL parameter is required" }),
      }),
    );

    await expect(parseUrl("bad")).rejects.toThrow("URL parameter is required");
  });
});
