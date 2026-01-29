const API_BASE = '';

export interface TrackMetadata {
	title: string;
	artist: string;
	album?: string;
	album_artist?: string;
	cover_url?: string;
	duration_ms?: number;
	release_date?: string;
	isrc?: string;
	spotify_id?: string;
	deezer_id?: string;
	track_number?: number;
	disc_number?: number;
	total_tracks?: number;
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
	type: 'track' | 'album' | 'playlist';
	id: string;
}

export interface MetadataResult {
	type: 'track' | 'album' | 'playlist';
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
	track_name: string;
	artist_name: string;
	album_name?: string;
	album_artist?: string;
	cover_url?: string;
	quality?: 'HI_RES_LOSSLESS' | 'LOSSLESS' | 'HIGH';
	embed_lyrics?: boolean;
	item_id?: string;
	duration_ms?: number;
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
		status: 'pending' | 'downloading' | 'completed' | 'error';
		message?: string;
		file_path?: string;
	};
}

interface ApiResponse<T> {
	success: boolean;
	data?: T;
	error?: string;
}

async function apiRequest<T>(
	endpoint: string,
	options: RequestInit = {}
): Promise<T> {
	const response = await fetch(`${API_BASE}${endpoint}`, {
		...options,
		headers: {
			'Content-Type': 'application/json',
			...options.headers
		}
	});

	const json: ApiResponse<T> = await response.json();

	if (!json.success) {
		throw new Error(json.error || 'Unknown error');
	}

	return json.data as T;
}

export async function parseURL(url: string): Promise<URLParseResult> {
	return apiRequest<URLParseResult>(`/api/parse-url?url=${encodeURIComponent(url)}`);
}

export async function getMetadata(url: string): Promise<MetadataResult> {
	const raw = await apiRequest<any>(`/api/metadata?url=${encodeURIComponent(url)}`);

	// Parse response based on structure
	if (raw.track) {
		const track = raw.track;
		return {
			type: 'track',
			track: {
				title: track.name,
				artist: track.artists,
				album: track.album_name,
				album_artist: track.album_artist,
				cover_url: track.images,
				duration_ms: track.duration_ms,
				release_date: track.release_date,
				isrc: track.isrc,
				spotify_id: track.spotify_id,
				track_number: track.track_number,
				disc_number: track.disc_number,
				total_tracks: track.total_tracks
			}
		};
	} else if (raw.album_info && raw.track_list) {
		const tracks = raw.track_list.map((t: any) => ({
			title: t.name,
			artist: t.artists,
			album: raw.album_info.name,
			cover_url: t.images || raw.album_info.images,
			duration_ms: t.duration_ms,
			isrc: t.isrc,
			spotify_id: t.spotify_id,
			track_number: t.track_number
		}));
		return {
			type: 'album',
			album: {
				name: raw.album_info.name,
				artist: raw.album_info.artists,
				cover_url: raw.album_info.images,
				release_date: raw.album_info.release_date,
				total_tracks: raw.album_info.total_tracks || tracks.length,
				tracks
			}
		};
	} else if (raw.playlist_info && raw.tracks) {
		const tracks = raw.tracks.map((t: any) => ({
			title: t.name,
			artist: t.artists,
			album: t.album_name,
			cover_url: t.images,
			duration_ms: t.duration_ms,
			isrc: t.isrc,
			spotify_id: t.spotify_id || t.id,
			track_number: t.track_number
		}));
		return {
			type: 'playlist',
			playlist: {
				name: raw.playlist_info.name,
				owner: raw.playlist_info.owner,
				cover_url: raw.playlist_info.images,
				total_tracks: raw.playlist_info.total_tracks || tracks.length,
				tracks
			}
		};
	}

	// Fallback: try to interpret as single track
	return {
		type: 'track',
		track: {
			title: raw.name || raw.title || 'Unknown',
			artist: raw.artists || raw.artist || 'Unknown',
			album: raw.album_name || raw.album,
			cover_url: raw.images || raw.cover_url,
			duration_ms: raw.duration_ms,
			isrc: raw.isrc,
			spotify_id: raw.spotify_id
		}
	};
}

export async function search(
	query: string,
	source: 'deezer' | 'spotify' = 'spotify',
	trackLimit = 20
): Promise<SearchResult> {
	const raw = await apiRequest<any>(
		`/api/search?q=${encodeURIComponent(query)}&source=${source}&track_limit=${trackLimit}`
	);

	// Map response to our format
	const tracks = (raw.tracks || []).map((t: any) => ({
		title: t.name,
		artist: t.artists,
		album: t.album_name,
		cover_url: t.images,
		duration_ms: t.duration_ms,
		isrc: t.isrc,
		spotify_id: t.spotify_id || t.id,
		track_number: t.track_number
	}));

	return { tracks };
}

export async function checkAvailability(
	spotifyId?: string,
	isrc?: string,
	deezerId?: string
): Promise<AvailabilityResult> {
	const params = new URLSearchParams();
	if (spotifyId) params.set('spotify_id', spotifyId);
	if (isrc) params.set('isrc', isrc);
	if (deezerId) params.set('deezer_id', deezerId);

	return apiRequest<AvailabilityResult>(`/api/availability?${params}`);
}

export async function downloadTrack(request: DownloadRequest): Promise<DownloadResult> {
	return apiRequest<DownloadResult>('/api/download', {
		method: 'POST',
		body: JSON.stringify({
			...request,
			embed_lyrics: true,
			embed_max_quality_cover: true,
			quality: request.quality || 'HI_RES_LOSSLESS'
		})
	});
}

export async function getProgress(itemId?: string): Promise<ProgressData> {
	const params = itemId ? `?item_id=${encodeURIComponent(itemId)}` : '';
	return apiRequest<ProgressData>(`/api/progress${params}`);
}

export function getFileUrl(filename: string): string {
	return `${API_BASE}/api/files/${encodeURIComponent(filename)}`;
}

export function formatDuration(ms: number): string {
	const seconds = Math.floor(ms / 1000);
	const minutes = Math.floor(seconds / 60);
	const remainingSeconds = seconds % 60;
	return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
}
