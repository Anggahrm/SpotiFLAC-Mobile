import { afterEach, describe, expect, it, vi } from "vitest";
import { downloadTrack } from "../../lib/api/client";

describe("flow-download-batch", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("supports sequential calls for batch workflow", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: { get: () => "application/json" },
        json: async () => ({ success: true, data: { success: true, file_path: "/tmp/a.flac" } }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: { get: () => "application/json" },
        json: async () => ({ success: true, data: { success: true, file_path: "/tmp/b.flac" } }),
      });

    vi.stubGlobal("fetch", fetchMock);

    const first = await downloadTrack({ track_name: "A", artist_name: "Artist" });
    const second = await downloadTrack({ track_name: "B", artist_name: "Artist" });

    expect(first.success).toBe(true);
    expect(second.success).toBe(true);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
