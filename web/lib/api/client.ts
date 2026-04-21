import type {
  AvailabilityResult,
  DownloadRequest,
  DownloadResult,
  MetadataResult,
  ProgressData,
  SearchResult,
  URLParseResult,
} from "./types";

const API_BASE = "";

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}

function isJsonResponse(response: Response): boolean {
  const contentType = response.headers.get("content-type") || "";
  return contentType.toLowerCase().includes("application/json");
}

function mapTrackLike(track: Record<string, unknown>) {
  return {
    title: String(track.name ?? "Unknown"),
    artist: String(track.artists ?? "Unknown"),
    album: track.album_name ? String(track.album_name) : undefined,
    album_artist: track.album_artist ? String(track.album_artist) : undefined,
    cover_url: track.images ? String(track.images) : undefined,
    preview_url: track.preview_url ? String(track.preview_url) : undefined,
    external_url: track.external_urls ? String(track.external_urls) : undefined,
    provider_id: track.provider_id ? String(track.provider_id) : undefined,
    duration_ms: typeof track.duration_ms === "number" ? track.duration_ms : undefined,
    release_date: track.release_date ? String(track.release_date) : undefined,
    isrc: track.isrc ? String(track.isrc) : undefined,
    spotify_id: track.spotify_id ? String(track.spotify_id) : track.id ? String(track.id) : undefined,
    deezer_id: track.deezer_id ? String(track.deezer_id) : undefined,
    track_number: typeof track.track_number === "number" ? track.track_number : undefined,
    disc_number: typeof track.disc_number === "number" ? track.disc_number : undefined,
    total_tracks: typeof track.total_tracks === "number" ? track.total_tracks : undefined,
    total_discs: typeof track.total_discs === "number" ? track.total_discs : undefined,
    source: track.source ? String(track.source) : undefined,
  };
}

async function apiRequest<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
  });

  if (!response.ok) {
    if (isJsonResponse(response)) {
      const errorJson = (await response.json()) as ApiResponse<unknown>;
      throw new Error(errorJson.error || `Request failed (${response.status})`);
    }
    throw new Error(`Request failed (${response.status})`);
  }

  if (!isJsonResponse(response)) {
    throw new Error("Unexpected non-JSON response from API");
  }

  const json = (await response.json()) as ApiResponse<T>;

  if (!json.success) {
    throw new Error(json.error || "Unknown error");
  }

  return json.data as T;
}

export async function parseUrl(url: string): Promise<URLParseResult> {
  return apiRequest<URLParseResult>(`/api/parse-url?url=${encodeURIComponent(url)}`);
}

export async function getMetadata(url: string): Promise<MetadataResult> {
  const raw = await apiRequest<Record<string, unknown>>(`/api/metadata?url=${encodeURIComponent(url)}`);

  if (raw.track && typeof raw.track === "object") {
    const track = raw.track as Record<string, unknown>;
    const mappedTrack = mapTrackLike(track);
    return {
      type: "track",
      track: {
        ...mappedTrack,
      },
    };
  }

  if (raw.album_info && raw.track_list && Array.isArray(raw.track_list)) {
    const albumInfo = raw.album_info as Record<string, unknown>;
    const tracks = raw.track_list.map((item) => {
      const track = item as Record<string, unknown>;
      return {
        ...mapTrackLike(track),
        album: track.album_name ? String(track.album_name) : albumInfo.name ? String(albumInfo.name) : undefined,
        cover_url: track.images
          ? String(track.images)
          : albumInfo.images
            ? String(albumInfo.images)
            : undefined,
      };
    });

    return {
      type: "album",
      album: {
        name: String(albumInfo.name ?? "Unknown"),
        artist: String(albumInfo.artists ?? "Unknown"),
        cover_url: albumInfo.images ? String(albumInfo.images) : undefined,
        release_date: albumInfo.release_date ? String(albumInfo.release_date) : undefined,
        total_tracks: typeof albumInfo.total_tracks === "number" ? albumInfo.total_tracks : tracks.length,
        tracks,
      },
    };
  }

  if (raw.playlist_info && raw.tracks && Array.isArray(raw.tracks)) {
    const playlistInfo = raw.playlist_info as Record<string, unknown>;
    const tracks = raw.tracks.map((item) => {
      const track = item as Record<string, unknown>;
      return mapTrackLike(track);
    });

    return {
      type: "playlist",
      playlist: {
        name: String(playlistInfo.name ?? "Unknown"),
        owner: String(playlistInfo.owner ?? "Unknown"),
        cover_url: playlistInfo.images ? String(playlistInfo.images) : undefined,
        total_tracks:
          typeof playlistInfo.total_tracks === "number" ? playlistInfo.total_tracks : tracks.length,
        tracks,
      },
    };
  }

  return {
    type: "track",
      track: {
        title: String(raw.name ?? raw.title ?? "Unknown"),
        artist: String(raw.artists ?? raw.artist ?? "Unknown"),
        album: raw.album_name ? String(raw.album_name) : raw.album ? String(raw.album) : undefined,
        cover_url: raw.images ? String(raw.images) : raw.cover_url ? String(raw.cover_url) : undefined,
        preview_url: raw.preview_url ? String(raw.preview_url) : undefined,
        external_url: raw.external_urls ? String(raw.external_urls) : undefined,
        provider_id: raw.provider_id ? String(raw.provider_id) : undefined,
        duration_ms: typeof raw.duration_ms === "number" ? raw.duration_ms : undefined,
        isrc: raw.isrc ? String(raw.isrc) : undefined,
      spotify_id: raw.spotify_id ? String(raw.spotify_id) : undefined,
    },
  };
}

export async function searchTracks(
  query: string,
  source: "deezer" | "spotify" = "spotify",
  trackLimit = 20,
): Promise<SearchResult> {
  const raw = await apiRequest<Record<string, unknown>>(
    `/api/search?q=${encodeURIComponent(query)}&source=${source}&track_limit=${trackLimit}`,
  );

  const sourceTracks = Array.isArray(raw.tracks) ? raw.tracks : [];
  const tracks = sourceTracks.map((item) => {
    const track = item as Record<string, unknown>;
    return mapTrackLike(track);
  });

  return { tracks };
}

export async function checkAvailability(
  spotifyId?: string,
  isrc?: string,
  deezerId?: string,
): Promise<AvailabilityResult> {
  const params = new URLSearchParams();
  if (spotifyId) params.set("spotify_id", spotifyId);
  if (isrc) params.set("isrc", isrc);
  if (deezerId) params.set("deezer_id", deezerId);

  return apiRequest<AvailabilityResult>(`/api/availability?${params.toString()}`);
}

export async function downloadTrack(request: DownloadRequest): Promise<DownloadResult> {
  return apiRequest<DownloadResult>("/api/download", {
    method: "POST",
    body: JSON.stringify({
      ...request,
      embed_lyrics: true,
      embed_max_quality_cover: true,
      quality: request.quality || "HI_RES_LOSSLESS",
    }),
  });
}

export async function getProgress(itemId?: string): Promise<ProgressData> {
  const params = itemId ? `?item_id=${encodeURIComponent(itemId)}` : "";
  return apiRequest<ProgressData>(`/api/progress${params}`);
}

export function getFileUrl(filename: string): string {
  return `${API_BASE}/api/files/${encodeURIComponent(filename)}`;
}

export function formatDuration(milliseconds: number): string {
  const seconds = Math.floor(milliseconds / 1000);
  const minutes = Math.floor(seconds / 60);
  const remainder = seconds % 60;
  return `${minutes}:${remainder.toString().padStart(2, "0")}`;
}
