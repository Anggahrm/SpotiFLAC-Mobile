import type { MetadataResult, TrackMetadata } from "./api/types"

export function isSupportedUrl(value: string): boolean {
  return /(spotify|deezer)\.com/i.test(value)
}

export function buildTrackKey(track: TrackMetadata): string {
  return (
    track.spotify_id ||
    track.deezer_id ||
    track.isrc ||
    `${track.title}::${track.artist}::${track.album ?? "unknown-album"}::${track.duration_ms ?? 0}`
  )
}

export function metadataToTracks(metadata: MetadataResult | null): TrackMetadata[] {
  if (!metadata) return []
  if (metadata.type === "track" && metadata.track) return [metadata.track]
  if (metadata.type === "album" && metadata.album) return metadata.album.tracks
  if (metadata.type === "playlist" && metadata.playlist) return metadata.playlist.tracks
  return []
}

export function getCollectionLabel(metadata: MetadataResult | null): string {
  if (!metadata) return "Idle workspace"
  if (metadata.type === "track") return "Single capture"
  if (metadata.type === "album") return "Album spread"
  return "Playlist drift"
}

export function getCollectionTitle(metadata: MetadataResult | null): string {
  if (!metadata) return "No selection yet"
  if (metadata.type === "track") return metadata.track?.title ?? "Unknown track"
  if (metadata.type === "album") return metadata.album?.name ?? "Unknown album"
  return metadata.playlist?.name ?? "Unknown playlist"
}

export function getCollectionSubtitle(metadata: MetadataResult | null): string {
  if (!metadata) return "Start with a link or a search query."
  if (metadata.type === "track") {
    const artist = metadata.track?.artist ?? "Unknown artist"
    const album = metadata.track?.album ? ` · ${metadata.track.album}` : ""
    return `${artist}${album}`
  }
  if (metadata.type === "album") {
    return `${metadata.album?.artist ?? "Unknown artist"} · ${metadata.album?.total_tracks ?? 0} tracks`
  }
  return `${metadata.playlist?.owner ?? "Unknown curator"} · ${metadata.playlist?.total_tracks ?? 0} tracks`
}

export function getCollectionArtwork(metadata: MetadataResult | null): string | undefined {
  if (!metadata) return undefined
  if (metadata.type === "track") return metadata.track?.cover_url
  if (metadata.type === "album") return metadata.album?.cover_url
  return metadata.playlist?.cover_url
}
