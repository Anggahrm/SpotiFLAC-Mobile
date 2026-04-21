import { describe, expect, it } from "vitest"
import {
  buildTrackKey,
  getCollectionLabel,
  getCollectionSubtitle,
  getCollectionTitle,
  isSupportedUrl,
  metadataToTracks,
} from "../../lib/downloader"

describe("downloader utils", () => {
  it("detects supported URLs", () => {
    expect(isSupportedUrl("https://open.spotify.com/track/abc")).toBe(true)
    expect(isSupportedUrl("https://www.deezer.com/track/abc")).toBe(true)
    expect(isSupportedUrl("hello world")).toBe(false)
  })

  it("creates stable keys for tracks", () => {
    expect(buildTrackKey({ title: "Song", artist: "Artist", spotify_id: "sp-1" })).toBe("sp-1")
  })

  it("maps track metadata into track arrays", () => {
    const metadata = {
      type: "album" as const,
      album: {
        name: "Album",
        artist: "Artist",
        total_tracks: 1,
        tracks: [{ title: "Song", artist: "Artist" }],
      },
    }

    expect(metadataToTracks(metadata)).toHaveLength(1)
    expect(getCollectionLabel(metadata)).toBe("Album spread")
    expect(getCollectionTitle(metadata)).toBe("Album")
    expect(getCollectionSubtitle(metadata)).toContain("Artist")
  })
})
