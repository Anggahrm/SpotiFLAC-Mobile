export interface ApiParameter {
	name: string;
	type: string;
	required: boolean;
	description: string;
	example?: string;
}

export interface ApiEndpoint {
	id: string;
	method: 'GET' | 'POST';
	path: string;
	title: string;
	description: string;
	parameters?: ApiParameter[];
	requestBody?: {
		description: string;
		example: Record<string, unknown>;
	};
	responseExample: Record<string, unknown>;
}

export const apiEndpoints: ApiEndpoint[] = [
	{
		id: 'health',
		method: 'GET',
		path: '/health',
		title: 'Health Check',
		description: 'Check if the API server is running and healthy.',
		responseExample: {
			status: 'ok',
			version: '1.0.0'
		}
	},
	{
		id: 'search',
		method: 'GET',
		path: '/api/search',
		title: 'Search Tracks',
		description: 'Search for tracks on Spotify or Deezer by query string.',
		parameters: [
			{ name: 'q', type: 'string', required: true, description: 'Search query (song name, artist, etc.)', example: 'bohemian rhapsody' },
			{ name: 'source', type: 'string', required: false, description: 'Search source: "spotify" or "deezer"', example: 'spotify' },
			{ name: 'track_limit', type: 'number', required: false, description: 'Maximum number of tracks to return (default: 20)', example: '10' }
		],
		responseExample: {
			success: true,
			data: {
				tracks: [
					{
						name: 'Bohemian Rhapsody',
						artists: 'Queen',
						album_name: 'A Night at the Opera',
						images: 'https://i.scdn.co/image/...',
						duration_ms: 354320,
						spotify_id: '7tFiyTwD0nx5a1eklYtX2J'
					}
				]
			}
		}
	},
	{
		id: 'metadata',
		method: 'GET',
		path: '/api/metadata',
		title: 'Get Metadata',
		description: 'Fetch detailed metadata for a Spotify or Deezer URL (track, album, or playlist).',
		parameters: [
			{ name: 'url', type: 'string', required: true, description: 'Spotify or Deezer URL', example: 'https://open.spotify.com/track/7tFiyTwD0nx5a1eklYtX2J' }
		],
		responseExample: {
			success: true,
			data: {
				track: {
					name: 'Bohemian Rhapsody',
					artists: 'Queen',
					album_name: 'A Night at the Opera',
					album_artist: 'Queen',
					images: 'https://i.scdn.co/image/...',
					duration_ms: 354320,
					release_date: '1975-10-31',
					isrc: 'GBUM71029604',
					spotify_id: '7tFiyTwD0nx5a1eklYtX2J'
				}
			}
		}
	},
	{
		id: 'parse-url',
		method: 'GET',
		path: '/api/parse-url',
		title: 'Parse URL',
		description: 'Parse a Spotify or Deezer URL to extract type and ID.',
		parameters: [
			{ name: 'url', type: 'string', required: true, description: 'Spotify or Deezer URL to parse', example: 'https://open.spotify.com/track/7tFiyTwD0nx5a1eklYtX2J' }
		],
		responseExample: {
			success: true,
			data: {
				type: 'track',
				id: '7tFiyTwD0nx5a1eklYtX2J'
			}
		}
	},
	{
		id: 'availability',
		method: 'GET',
		path: '/api/availability',
		title: 'Check Availability',
		description: 'Check if a track is available for download on Tidal, Qobuz, or Amazon.',
		parameters: [
			{ name: 'spotify_id', type: 'string', required: false, description: 'Spotify track ID', example: '7tFiyTwD0nx5a1eklYtX2J' },
			{ name: 'isrc', type: 'string', required: false, description: 'ISRC code', example: 'GBUM71029604' },
			{ name: 'deezer_id', type: 'string', required: false, description: 'Deezer track ID', example: '3157895' }
		],
		responseExample: {
			success: true,
			data: {
				tidal: true,
				qobuz: true,
				amazon: false
			}
		}
	},
	{
		id: 'download',
		method: 'POST',
		path: '/api/download',
		title: 'Download Track',
		description: 'Download a track in FLAC format from Tidal, Qobuz, or Amazon.',
		requestBody: {
			description: 'Track information and download options',
			example: {
				track_name: 'Bohemian Rhapsody',
				artist_name: 'Queen',
				album_name: 'A Night at the Opera',
				isrc: 'GBUM71029604',
				service: 'tidal',
				quality: 'HI_RES_LOSSLESS',
				embed_lyrics: true
			}
		},
		responseExample: {
			success: true,
			data: {
				success: true,
				file_path: '/downloads/Queen - Bohemian Rhapsody.flac',
				service: 'tidal',
				actual_bit_depth: 24,
				actual_sample_rate: 96000
			}
		}
	},
	{
		id: 'progress',
		method: 'GET',
		path: '/api/progress',
		title: 'Get Download Progress',
		description: 'Get the progress of ongoing downloads.',
		parameters: [
			{ name: 'item_id', type: 'string', required: false, description: 'Specific item ID to get progress for', example: 'uuid-here' }
		],
		responseExample: {
			success: true,
			data: {
				'uuid-123': {
					progress: 75,
					status: 'downloading',
					message: 'Downloading...'
				}
			}
		}
	},
	{
		id: 'files',
		method: 'GET',
		path: '/api/files/{filename}',
		title: 'Get File',
		description: 'Download a completed file by filename.',
		parameters: [
			{ name: 'filename', type: 'string', required: true, description: 'Name of the file to download', example: 'Queen - Bohemian Rhapsody.flac' }
		],
		responseExample: {
			note: 'Returns the file binary data with appropriate Content-Type header'
		}
	},
	{
		id: 'lyrics',
		method: 'GET',
		path: '/api/lyrics',
		title: 'Get Lyrics',
		description: 'Fetch lyrics for a track.',
		parameters: [
			{ name: 'isrc', type: 'string', required: false, description: 'ISRC code', example: 'GBUM71029604' },
			{ name: 'track_name', type: 'string', required: false, description: 'Track name', example: 'Bohemian Rhapsody' },
			{ name: 'artist_name', type: 'string', required: false, description: 'Artist name', example: 'Queen' },
			{ name: 'duration_ms', type: 'number', required: false, description: 'Track duration in milliseconds', example: '354320' }
		],
		responseExample: {
			success: true,
			data: {
				lyrics: '[00:00.00] Is this the real life?\n[00:03.50] Is this just fantasy?',
				synced: true
			}
		}
	}
];

export function generateCurl(endpoint: ApiEndpoint, baseUrl: string = ''): string {
	const url = `${baseUrl}${endpoint.path}`;

	if (endpoint.method === 'GET') {
		if (endpoint.parameters?.length) {
			const params = endpoint.parameters
				.filter(p => p.example)
				.map(p => `${p.name}=${encodeURIComponent(p.example || '')}`)
				.join('&');
			return `curl "${url}?${params}"`;
		}
		return `curl "${url}"`;
	}

	if (endpoint.method === 'POST' && endpoint.requestBody) {
		const body = JSON.stringify(endpoint.requestBody.example, null, 2);
		return `curl -X POST "${url}" \\
  -H "Content-Type: application/json" \\
  -d '${body}'`;
	}

	return `curl "${url}"`;
}

export function generateFetch(endpoint: ApiEndpoint, baseUrl: string = ''): string {
	const url = `${baseUrl}${endpoint.path}`;

	if (endpoint.method === 'GET') {
		if (endpoint.parameters?.length) {
			const params = endpoint.parameters
				.filter(p => p.example)
				.map(p => `  params.set('${p.name}', '${p.example}');`)
				.join('\n');
			return `const params = new URLSearchParams();
${params}

const response = await fetch(\`${url}?\${params}\`);
const data = await response.json();`;
		}
		return `const response = await fetch('${url}');
const data = await response.json();`;
	}

	if (endpoint.method === 'POST' && endpoint.requestBody) {
		const body = JSON.stringify(endpoint.requestBody.example, null, 2);
		return `const response = await fetch('${url}', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify(${body}),
});
const data = await response.json();`;
	}

	return `const response = await fetch('${url}');
const data = await response.json();`;
}

export function generateAxios(endpoint: ApiEndpoint, baseUrl: string = ''): string {
	const url = `${baseUrl}${endpoint.path}`;

	if (endpoint.method === 'GET') {
		if (endpoint.parameters?.length) {
			const params = endpoint.parameters
				.filter(p => p.example)
				.map(p => `    ${p.name}: '${p.example}'`)
				.join(',\n');
			return `const { data } = await axios.get('${url}', {
  params: {
${params}
  }
});`;
		}
		return `const { data } = await axios.get('${url}');`;
	}

	if (endpoint.method === 'POST' && endpoint.requestBody) {
		const body = JSON.stringify(endpoint.requestBody.example, null, 2);
		return `const { data } = await axios.post('${url}', ${body});`;
	}

	return `const { data } = await axios.get('${url}');`;
}

export function generatePython(endpoint: ApiEndpoint, baseUrl: string = ''): string {
	const url = `${baseUrl}${endpoint.path}`;

	if (endpoint.method === 'GET') {
		if (endpoint.parameters?.length) {
			const params = endpoint.parameters
				.filter(p => p.example)
				.map(p => `    '${p.name}': '${p.example}'`)
				.join(',\n');
			return `import requests

params = {
${params}
}
response = requests.get('${url}', params=params)
data = response.json()`;
		}
		return `import requests

response = requests.get('${url}')
data = response.json()`;
	}

	if (endpoint.method === 'POST' && endpoint.requestBody) {
		const body = JSON.stringify(endpoint.requestBody.example, null, 2)
			.replace(/"/g, "'")
			.replace(/: true/g, ': True')
			.replace(/: false/g, ': False')
			.replace(/: null/g, ': None');
		return `import requests

payload = ${body}
response = requests.post('${url}', json=payload)
data = response.json()`;
	}

	return `import requests

response = requests.get('${url}')
data = response.json()`;
}
