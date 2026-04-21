export interface TrackMetadata {
  title: string;
  artist: string;
  album?: string;
  album_artist?: string;
  cover_url?: string;
  preview_url?: string;
  external_url?: string;
  provider_id?: string;
  duration_ms?: number;
  release_date?: string;
  isrc?: string;
  spotify_id?: string;
  deezer_id?: string;
  track_number?: number;
  disc_number?: number;
  total_tracks?: number;
  total_discs?: number;
  source?: string;
}

export interface AlbumMetadata {
  name: string;
  artist: string;
  cover_url?: string;
  release_date?: string;
  total_tracks: number;
  tracks: TrackMetadata[];
}

export interface PlaylistMetadata {
  name: string;
  owner: string;
  cover_url?: string;
  total_tracks: number;
  tracks: TrackMetadata[];
}

export interface URLParseResult {
  type: "track" | "album" | "playlist";
  id: string;
}

export interface MetadataResult {
  type: "track" | "album" | "playlist";
  track?: TrackMetadata;
  album?: AlbumMetadata;
  playlist?: PlaylistMetadata;
  availability?: AvailabilityResult;
}

export interface SearchResult {
  tracks: TrackMetadata[];
  artists?: { name: string; id: string; image_url?: string }[];
}

export interface AvailabilityResult {
  tidal?: boolean;
  qobuz?: boolean;
  amazon?: boolean;
}

export interface DownloadRequest {
  isrc?: string;
  service?: string;
  spotify_id?: string;
  deezer_id?: string;
  tidal_id?: string;
  qobuz_id?: string;
  track_name: string;
  artist_name: string;
  album_name?: string;
  album_artist?: string;
  cover_url?: string;
  quality?: "HI_RES_LOSSLESS" | "LOSSLESS" | "HIGH";
  embed_lyrics?: boolean;
  track_number?: number;
  disc_number?: number;
  total_tracks?: number;
  total_discs?: number;
  release_date?: string;
  item_id?: string;
  duration_ms?: number;
  source?: string;
  genre?: string;
  label?: string;
  copyright?: string;
  composer?: string;
  lyrics_mode?: string;
  use_extensions?: boolean;
  use_fallback?: boolean;
}

export interface DownloadResult {
  success: boolean;
  file_path?: string;
  file_name?: string;
  error?: string;
  service?: string;
  actual_bit_depth?: number;
  actual_sample_rate?: number;
}

export interface ProgressData {
  [itemId: string]: {
    progress: number;
    status: "pending" | "downloading" | "finalizing" | "completed" | "error";
    message?: string;
    file_path?: string;
  };
}
