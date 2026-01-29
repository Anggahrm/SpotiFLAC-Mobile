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

export interface SearchResult {
	tracks: TrackMetadata[];
	artists?: { name: string; id: string; image_url?: string }[];
}

export interface AvailabilityResult {
	tidal?: { available: boolean; quality?: string };
	qobuz?: { available: boolean; quality?: string };
	amazon?: { available: boolean; quality?: string };
	deezer?: { available: boolean; quality?: string };
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
	quality?: 'LOSSLESS' | 'HIGH' | 'NORMAL';
	embed_lyrics?: boolean;
	item_id?: string;
	duration_ms?: number;
}

export interface DownloadResult {
	success: boolean;
	file_path?: string;
	file_name?: string;
	error?: string;
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

export async function getMetadata(url: string): Promise<TrackMetadata> {
	return apiRequest<TrackMetadata>(`/api/metadata?url=${encodeURIComponent(url)}`);
}

export async function search(
	query: string,
	source: 'deezer' | 'spotify' = 'deezer',
	trackLimit = 20
): Promise<SearchResult> {
	return apiRequest<SearchResult>(
		`/api/search?q=${encodeURIComponent(query)}&source=${source}&track_limit=${trackLimit}`
	);
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
		body: JSON.stringify(request)
	});
}

export async function getProgress(itemId?: string): Promise<ProgressData> {
	const params = itemId ? `?item_id=${encodeURIComponent(itemId)}` : '';
	return apiRequest<ProgressData>(`/api/progress${params}`);
}

export function getFileUrl(filename: string): string {
	return `${API_BASE}/api/files/${encodeURIComponent(filename)}`;
}
