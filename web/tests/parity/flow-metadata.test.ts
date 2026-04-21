import { afterEach, describe, expect, it, vi } from "vitest";
import { getMetadata } from "../../lib/api/client";

describe("flow-metadata", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("maps track metadata response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: { get: () => "application/json" },
        json: async () => ({
          success: true,
          data: {
            track: {
              name: "Wildflower",
              artists: "Billie Eilish",
              album_name: "Hit Me Hard and Soft",
              isrc: "USUM72401967",
              spotify_id: "4iV5W9uYEdYUVa79Axb7Rh",
            },
          },
        }),
      }),
    );

    const result = await getMetadata("https://open.spotify.com/track/4iV5W9uYEdYUVa79Axb7Rh");
    expect(result.type).toBe("track");
    expect(result.track?.title).toBe("Wildflower");
    expect(result.track?.artist).toBe("Billie Eilish");
  });

  it("throws when backend returns malformed envelope", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: { get: () => "application/json" },
        json: async () => ({ foo: "bar" }),
      }),
    );

    await expect(getMetadata("https://open.spotify.com/track/4iV5W9uYEdYUVa79Axb7Rh")).rejects.toThrow(
      "Unknown error",
    );
  });
});
