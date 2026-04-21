import { afterEach, describe, expect, it, vi } from "vitest";
import { downloadTrack } from "../../lib/api/client";

describe("flow-download-single", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("posts download request and returns file path", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: { get: () => "application/json" },
      json: async () => ({
        success: true,
        data: {
          success: true,
          file_path: "/tmp/spotiflac_downloads/song.flac",
          service: "tidal",
        },
      }),
    });
    vi.stubGlobal("fetch", fetchMock);

    const result = await downloadTrack({
      track_name: "Song",
      artist_name: "Artist",
      service: "tidal",
    });

    expect(result.success).toBe(true);
    expect(result.file_path).toContain("song.flac");
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("throws on non-OK download response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        headers: { get: () => "application/json" },
        json: async () => ({ success: false, error: "provider failed" }),
      }),
    );

    await expect(downloadTrack({ track_name: "Song", artist_name: "Artist" })).rejects.toThrow("provider failed");
  });
});
