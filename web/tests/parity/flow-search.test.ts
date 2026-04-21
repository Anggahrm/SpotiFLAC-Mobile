import { afterEach, describe, expect, it, vi } from "vitest";
import { searchTracks } from "../../lib/api/client";

describe("flow-search", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("maps backend search response into track list", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: { get: () => "application/json" },
        json: async () => ({
          success: true,
          data: {
            tracks: [
              {
                name: "Song A",
                artists: "Artist A",
                album_name: "Album A",
                duration_ms: 210000,
                spotify_id: "spid-1",
              },
            ],
          },
        }),
      }),
    );

    const result = await searchTracks("song a", "spotify", 10);
    expect(result.tracks).toHaveLength(1);
    const [track] = result.tracks;
    expect(track?.title).toBe("Song A");
    expect(track?.artist).toBe("Artist A");
  });

  it("rejects on non-JSON server response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: { get: () => "text/html" },
        json: async () => ({})
      }),
    );

    await expect(searchTracks("song", "spotify")).rejects.toThrow("Unexpected non-JSON response");
  });
});
