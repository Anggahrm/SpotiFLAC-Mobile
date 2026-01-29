<script lang="ts">
	import {
		getMetadata,
		downloadTrack,
		getFileUrl,
		formatDuration,
		checkAvailability,
		search,
		type MetadataResult,
		type TrackMetadata
	} from '$lib/api';
	import { toasts } from '$lib/stores/toasts';
	import { Download, Loader2, Music, Disc3, ListMusic, Check, X, Terminal } from 'lucide-svelte';

	type Provider = 'auto' | 'tidal' | 'qobuz' | 'amazon';
	type SearchSource = 'spotify' | 'deezer';

	let query = $state('');
	let metadata: MetadataResult | null = $state(null);
	let searchResults: TrackMetadata[] = $state([]);
	let loading = $state(false);
	let provider: Provider = $state('auto');
	let searchSource: SearchSource = $state('deezer');
	let isSearchMode = $state(false);
	let debounceTimer: ReturnType<typeof setTimeout> | null = $state(null);

	// Download states
	let downloading = $state(false);
	let batchDownloading = $state(false);
	let batchProgress = $state({ current: 0, total: 0, success: 0, failed: 0 });
	let downloadResults: Map<string, { status: 'pending' | 'downloading' | 'success' | 'error'; fileName?: string }> = $state(new Map());

	const providers: { id: Provider; name: string }[] = [
		{ id: 'auto', name: 'Auto' },
		{ id: 'tidal', name: 'Tidal' },
		{ id: 'qobuz', name: 'Qobuz' },
		{ id: 'amazon', name: 'Amazon' }
	];

	function isUrl(str: string): boolean {
		return str.includes('spotify.com') || str.includes('deezer.com') || str.startsWith('http');
	}

	// Auto search/fetch on input change with debounce
	function handleInput() {
		const trimmed = query.trim();

		// Clear previous timer
		if (debounceTimer) {
			clearTimeout(debounceTimer);
		}

		// Don't search if empty or too short
		if (!trimmed || trimmed.length < 2) {
			if (!isUrl(trimmed)) {
				searchResults = [];
				isSearchMode = false;
			}
			return;
		}

		// For URLs, fetch immediately
		if (isUrl(trimmed)) {
			isSearchMode = false;
			searchResults = [];
			fetchMetadata();
			return;
		}

		// For search queries, debounce 500ms
		isSearchMode = true;
		metadata = null;
		debounceTimer = setTimeout(() => {
			handleSearch();
		}, 500);
	}

	async function handleSearch() {
		const trimmed = query.trim();
		if (!trimmed) return;

		loading = true;
		searchResults = [];
		downloadResults = new Map();

		try {
			const result = await search(trimmed, searchSource, 20);
			searchResults = result.tracks;
			if (searchResults.length > 0) {
				toasts.success(`Found ${searchResults.length} tracks`);
			} else {
				toasts.error('No tracks found');
			}
		} catch (err) {
			toasts.error(err instanceof Error ? err.message : 'Search failed');
		} finally {
			loading = false;
		}
	}

	async function selectTrack(track: TrackMetadata) {
		// If track has spotify_id, fetch full metadata to get ISRC
		if (track.spotify_id && !track.isrc) {
			loading = true;
			searchResults = [];
			isSearchMode = false;
			try {
				const url = `https://open.spotify.com/track/${track.spotify_id}`;
				metadata = await getMetadata(url);
				toasts.success('Track loaded!');
			} catch (err) {
				// Fallback to search result data if metadata fetch fails
				metadata = {
					type: 'track',
					track
				};
				toasts.error('Using limited track info');
			} finally {
				loading = false;
			}
		} else {
			metadata = {
				type: 'track',
				track
			};
			searchResults = [];
			isSearchMode = false;
		}
	}

	async function fetchMetadata() {
		const trimmedUrl = query.trim();
		if (!trimmedUrl) return;

		if (!trimmedUrl.includes('spotify.com') && !trimmedUrl.includes('deezer.com')) {
			toasts.error('Please enter a valid Spotify or Deezer URL');
			return;
		}

		loading = true;
		metadata = null;
		downloadResults = new Map();

		try {
			metadata = await getMetadata(trimmedUrl);
			if (metadata.type === 'track') {
				toasts.success('Track found!');
			} else if (metadata.type === 'album') {
				toasts.success(`Album found: ${metadata.album?.tracks.length} tracks`);
			} else if (metadata.type === 'playlist') {
				toasts.success(`Playlist found: ${metadata.playlist?.tracks.length} tracks`);
			}
		} catch (err) {
			toasts.error(err instanceof Error ? err.message : 'Failed to fetch metadata');
		} finally {
			loading = false;
		}
	}

	async function downloadSingleTrack(track: TrackMetadata) {
		const trackKey = track.spotify_id || track.isrc || track.title;
		downloadResults.set(trackKey, { status: 'downloading' });
		downloadResults = new Map(downloadResults);

		try {
			// Step 1: If we don't have ISRC, fetch full metadata first (like Telegram bot does)
			let trackData = track;
			if (!track.isrc && track.spotify_id) {
				try {
					const isDeezerTrack = track.spotify_id.startsWith('deezer:');
					let url: string;
					if (isDeezerTrack) {
						const deezerId = track.spotify_id.replace('deezer:', '');
						url = `https://www.deezer.com/track/${deezerId}`;
					} else {
						url = `https://open.spotify.com/track/${track.spotify_id}`;
					}
					const fullMetadata = await getMetadata(url);
					if (fullMetadata.track) {
						trackData = fullMetadata.track;
					}
				} catch {
					// Continue with original track data if metadata fetch fails
				}
			}

			// Step 2: Check availability (like Telegram bot)
			let service = provider;
			if (provider === 'auto') {
				const isDeezerTrack = trackData.spotify_id?.startsWith('deezer:');
				const deezerId = isDeezerTrack ? trackData.spotify_id?.replace('deezer:', '') : undefined;
				const spotifyId = isDeezerTrack ? undefined : trackData.spotify_id;
				
				const avail = await checkAvailability(spotifyId, trackData.isrc, deezerId);
				if (avail.tidal) service = 'tidal';
				else if (avail.qobuz) service = 'qobuz';
				else if (avail.amazon) service = 'amazon';
				else throw new Error('Not available on any service');
			}

			// Step 3: Download with full metadata including ISRC
			const result = await downloadTrack({
				track_name: trackData.title,
				artist_name: trackData.artist,
				album_name: trackData.album,
				album_artist: trackData.album_artist,
				cover_url: trackData.cover_url,
				spotify_id: trackData.spotify_id,
				isrc: trackData.isrc,
				service: service === 'auto' ? undefined : service,
				item_id: crypto.randomUUID(),
				duration_ms: trackData.duration_ms
			});

			if (result.success && result.file_path) {
				// Extract filename from path
				const fileName = result.file_path.split('/').pop() || result.file_path;
				downloadResults.set(trackKey, { status: 'success', fileName });
				downloadResults = new Map(downloadResults);

				const link = document.createElement('a');
				link.href = getFileUrl(fileName);
				link.download = fileName;
				link.click();

				return { success: true };
			} else {
				throw new Error(result.error || 'Download failed');
			}
		} catch (err) {
			downloadResults.set(trackKey, { status: 'error' });
			downloadResults = new Map(downloadResults);
			return { success: false, error: err instanceof Error ? err.message : 'Unknown error' };
		}
	}

	async function handleDownloadTrack() {
		if (!metadata?.track) return;
		downloading = true;

		const result = await downloadSingleTrack(metadata.track);
		if (result.success) {
			toasts.success('Download complete!');
		} else {
			toasts.error(result.error || 'Download failed');
		}

		downloading = false;
	}

	async function handleDownloadAll() {
		const tracks = metadata?.album?.tracks || metadata?.playlist?.tracks;
		if (!tracks || tracks.length === 0) return;

		batchDownloading = true;
		batchProgress = { current: 0, total: tracks.length, success: 0, failed: 0 };

		for (let i = 0; i < tracks.length; i++) {
			batchProgress.current = i + 1;
			const result = await downloadSingleTrack(tracks[i]);

			if (result.success) {
				batchProgress.success++;
			} else {
				batchProgress.failed++;
			}
		}

		batchDownloading = false;
		toasts.success(`Download complete: ${batchProgress.success}/${batchProgress.total} tracks`);
	}

	function getTrackStatus(track: TrackMetadata) {
		const key = track.spotify_id || track.isrc || track.title;
		return downloadResults.get(key);
	}

	function reset() {
		query = '';
		metadata = null;
		searchResults = [];
		downloadResults = new Map();
		isSearchMode = false;
		if (debounceTimer) {
			clearTimeout(debounceTimer);
		}
	}
</script>

<div class="min-h-screen flex items-center justify-center font-mono p-2 sm:p-4 selection:bg-violet-500 selection:text-white bg-gradient-to-b from-slate-50 to-white relative overflow-hidden">
	<!-- Gradient blobs - hidden on mobile for performance -->
	<div class="hidden sm:block fixed bottom-0 left-0 w-[400px] h-[400px] bg-gradient-to-tr from-sky-400/20 to-violet-500/20 blur-[100px] rounded-full pointer-events-none -z-10 translate-y-1/3 -translate-x-1/4"></div>
	<div class="hidden sm:block fixed top-0 right-0 w-[300px] h-[300px] bg-gradient-to-bl from-violet-300/20 to-sky-400/20 blur-[80px] rounded-full pointer-events-none -z-10 -translate-y-1/3 translate-x-1/4"></div>

	<!-- Main Card -->
	<div class="bg-white border-2 border-violet-500 shadow-[4px_4px_0px_0px_#c4b5fd] max-w-lg w-full relative group transition-all duration-300 hover:shadow-[6px_6px_0px_0px_#8b5cf6] rounded-lg overflow-hidden sm:my-4">

		<!-- Terminal Header -->
		<div class="bg-gradient-to-r from-violet-500 to-sky-500 text-white px-3 sm:px-4 py-2.5 sm:py-3 flex justify-between items-center border-b-2 border-violet-500">
			<div class="flex gap-2">
				<div class="w-3 h-3 rounded-full bg-red-400 border border-white/30"></div>
				<div class="w-3 h-3 rounded-full bg-yellow-400 border border-white/30"></div>
				<div class="w-3 h-3 rounded-full bg-green-400 border border-white/30"></div>
			</div>
			<div class="text-xs font-bold tracking-widest opacity-90 flex items-center gap-2">
				<Terminal class="w-3 h-3" />
				<span>zFlac</span>
			</div>
		</div>

		<div class="p-4 sm:p-6">
			<!-- Provider Selector -->
			<div class="grid grid-cols-4 gap-1.5 sm:gap-2 mb-3 sm:mb-4">
				{#each providers as p}
					<button
						class="px-2 sm:px-3 py-1.5 sm:py-2 text-[10px] sm:text-xs font-bold border-2 transition-all duration-200 rounded {provider === p.id
							? 'bg-gradient-to-r from-violet-500 to-sky-500 text-white border-violet-500 shadow-[2px_2px_0px_0px_#c4b5fd]'
							: 'bg-white text-slate-600 border-violet-200 hover:border-violet-400 hover:bg-violet-50'}"
						onclick={() => provider = p.id}
					>
						{p.name}
					</button>
				{/each}
			</div>

			<!-- Search Source Toggle -->
			<div class="grid grid-cols-2 gap-1.5 sm:gap-2 mb-3 sm:mb-4">
				<button
					class="px-2 sm:px-3 py-1.5 text-[10px] sm:text-xs font-bold border-2 transition-all duration-200 rounded {searchSource === 'deezer'
						? 'bg-gradient-to-r from-amber-500 to-orange-500 text-white border-amber-500 shadow-[2px_2px_0px_0px_#fcd34d]'
						: 'bg-white text-slate-600 border-amber-200 hover:border-amber-400 hover:bg-amber-50'}"
					onclick={() => searchSource = 'deezer'}
				>
					Deezer
				</button>
				<button
					class="px-2 sm:px-3 py-1.5 text-[10px] sm:text-xs font-bold border-2 transition-all duration-200 rounded {searchSource === 'spotify'
						? 'bg-gradient-to-r from-emerald-500 to-teal-500 text-white border-emerald-500 shadow-[2px_2px_0px_0px_#6ee7b7]'
						: 'bg-white text-slate-600 border-emerald-200 hover:border-emerald-400 hover:bg-emerald-50'}"
					onclick={() => searchSource = 'spotify'}
				>
					Spotify
				</button>
			</div>

			<!-- URL/Search Input -->
			<div class="mb-3 sm:mb-4">
				<div class="relative">
					<input
						type="text"
						placeholder="Paste URL or type song name..."
						class="w-full h-11 sm:h-12 px-3 sm:px-4 text-xs sm:text-sm border-2 border-violet-300 bg-gradient-to-r from-violet-50/50 to-sky-50/50 rounded-lg outline-none transition-all focus:border-violet-500 focus:shadow-[0_0_10px_rgba(139,92,246,0.2)] placeholder:text-slate-400"
						bind:value={query}
						oninput={handleInput}
					/>
					{#if loading}
						<div class="absolute right-3 top-1/2 -translate-y-1/2">
							<Loader2 class="w-4 h-4 animate-spin text-violet-500" />
						</div>
					{/if}
				</div>
			</div>

			<!-- Search Results -->
			{#if searchResults.length > 0}
				<div class="border-2 border-violet-200 rounded-lg overflow-hidden bg-gradient-to-r from-violet-50/30 to-sky-50/30 mb-3 sm:mb-4">
					<div class="px-3 sm:px-4 py-2 border-b-2 border-violet-200 bg-gradient-to-r from-violet-100/50 to-sky-100/50">
						<span class="text-[10px] sm:text-xs font-bold text-violet-600">RESULTS ({searchResults.length})</span>
					</div>
					<div class="max-h-64 sm:max-h-80 overflow-y-auto">
						{#each searchResults as track, i}
							<button
								class="w-full flex items-center gap-2 sm:gap-3 px-3 sm:px-4 py-2.5 sm:py-3 border-b border-violet-100 last:border-b-0 hover:bg-violet-50/70 active:bg-violet-100/70 transition-colors text-left"
								onclick={() => selectTrack(track)}
							>
								{#if track.cover_url}
									<img src={track.cover_url} alt="" class="w-10 h-10 sm:w-12 sm:h-12 rounded border-2 border-violet-200 object-cover shadow-[2px_2px_0px_0px_#c4b5fd]" />
								{:else}
									<div class="w-10 h-10 sm:w-12 sm:h-12 rounded border-2 border-violet-200 bg-violet-50 flex items-center justify-center">
										<Music class="w-4 h-4 sm:w-5 sm:h-5 text-violet-400" />
									</div>
								{/if}
								<div class="flex-1 min-w-0">
									<p class="text-xs sm:text-sm font-medium text-slate-700 truncate">{track.title}</p>
									<p class="text-[10px] sm:text-xs text-slate-500 truncate">{track.artist}</p>
								</div>
								{#if track.duration_ms}
									<span class="text-[10px] font-mono text-slate-400 hidden sm:inline">{formatDuration(track.duration_ms)}</span>
								{/if}
								<Download class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-violet-400 flex-shrink-0" />
							</button>
						{/each}
					</div>
				</div>
			{/if}

			<!-- Content -->
			{#if metadata}
				<!-- Track View -->
				{#if metadata.type === 'track' && metadata.track}
					{@const track = metadata.track}
					{@const trackStatus = getTrackStatus(track)}
					<div class="border-2 border-violet-200 rounded-lg overflow-hidden bg-gradient-to-r from-violet-50/30 to-sky-50/30">
						<div class="flex gap-3 sm:gap-4 p-3 sm:p-4">
							{#if track.cover_url}
								<img
									src={track.cover_url}
									alt={track.title}
									class="w-20 h-20 sm:w-24 sm:h-24 flex-shrink-0 rounded border-2 border-violet-200 object-cover shadow-[2px_2px_0px_0px_#c4b5fd]"
								/>
							{:else}
								<div class="w-20 h-20 sm:w-24 sm:h-24 flex-shrink-0 rounded border-2 border-violet-200 bg-gradient-to-br from-violet-100 to-sky-100 flex items-center justify-center">
									<Music class="w-6 h-6 sm:w-8 sm:h-8 text-violet-400" />
								</div>
							{/if}

							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2 mb-1">
									<span class="px-1.5 sm:px-2 py-0.5 text-[8px] sm:text-[10px] font-bold bg-gradient-to-r from-violet-500 to-sky-500 text-white rounded-full">TRACK</span>
								</div>
								<h3 class="font-bold text-sm sm:text-base text-slate-800 truncate">{track.title}</h3>
								<p class="text-xs sm:text-sm text-slate-500 truncate">{track.artist}</p>
								{#if track.album}
									<p class="text-[10px] sm:text-xs text-slate-400 truncate mt-0.5">{track.album}</p>
								{/if}
								<div class="flex gap-1.5 sm:gap-2 mt-1.5 sm:mt-2 flex-wrap">
									{#if track.duration_ms}
										<span class="px-1.5 sm:px-2 py-0.5 text-[8px] sm:text-[10px] font-mono bg-violet-100 text-violet-600 rounded">{formatDuration(track.duration_ms)}</span>
									{/if}
									{#if track.release_date}
										<span class="px-1.5 sm:px-2 py-0.5 text-[8px] sm:text-[10px] font-mono bg-sky-100 text-sky-600 rounded">{track.release_date.split('-')[0]}</span>
									{/if}
								</div>
							</div>
						</div>

						<div class="border-t-2 border-violet-200 p-2.5 sm:p-3">
							{#if trackStatus?.status === 'success'}
								<a
									href={getFileUrl(trackStatus.fileName || '')}
									download
									class="flex items-center justify-center gap-2 w-full py-2 sm:py-2.5 text-xs sm:text-sm font-bold bg-gradient-to-r from-emerald-500 to-teal-500 text-white border-2 border-emerald-500 rounded shadow-[2px_2px_0px_0px_#6ee7b7] hover:shadow-none hover:translate-x-[2px] hover:translate-y-[2px] transition-all active:scale-[0.98]"
								>
									<Download class="w-3.5 h-3.5 sm:w-4 sm:h-4" />
									SAVE FILE
								</a>
							{:else}
								<button
									onclick={handleDownloadTrack}
									disabled={downloading}
									class="flex items-center justify-center gap-2 w-full py-2 sm:py-2.5 text-xs sm:text-sm font-bold bg-gradient-to-r from-violet-500 to-sky-500 text-white border-2 border-violet-500 rounded shadow-[2px_2px_0px_0px_#c4b5fd] hover:shadow-none hover:translate-x-[2px] hover:translate-y-[2px] transition-all disabled:opacity-50 disabled:cursor-not-allowed active:scale-[0.98]"
								>
									{#if downloading}
										<Loader2 class="w-3.5 h-3.5 sm:w-4 sm:h-4 animate-spin" />
										DOWNLOADING...
									{:else}
										<Download class="w-3.5 h-3.5 sm:w-4 sm:h-4" />
										DOWNLOAD FLAC
									{/if}
								</button>
							{/if}
						</div>
					</div>

				<!-- Album View -->
				{:else if metadata.type === 'album' && metadata.album}
					{@const album = metadata.album}
					<div class="border-2 border-violet-200 rounded-lg overflow-hidden bg-gradient-to-r from-violet-50/30 to-sky-50/30">
						<div class="flex gap-3 sm:gap-4 p-3 sm:p-4">
							{#if album.cover_url}
								<img
									src={album.cover_url}
									alt={album.name}
									class="w-20 h-20 sm:w-24 sm:h-24 flex-shrink-0 rounded border-2 border-violet-200 object-cover shadow-[2px_2px_0px_0px_#c4b5fd]"
								/>
							{:else}
								<div class="w-20 h-20 sm:w-24 sm:h-24 flex-shrink-0 rounded border-2 border-violet-200 bg-gradient-to-br from-violet-100 to-sky-100 flex items-center justify-center">
									<Disc3 class="w-6 h-6 sm:w-8 sm:h-8 text-violet-400" />
								</div>
							{/if}

							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2 mb-1">
									<span class="px-1.5 sm:px-2 py-0.5 text-[8px] sm:text-[10px] font-bold bg-gradient-to-r from-violet-500 to-sky-500 text-white rounded-full">ALBUM</span>
								</div>
								<h3 class="font-bold text-sm sm:text-base text-slate-800 truncate">{album.name}</h3>
								<p class="text-xs sm:text-sm text-slate-500 truncate">{album.artist}</p>
								<div class="flex gap-1.5 sm:gap-2 mt-1.5 sm:mt-2 flex-wrap">
									<span class="px-1.5 sm:px-2 py-0.5 text-[8px] sm:text-[10px] font-mono bg-violet-100 text-violet-600 rounded">{album.total_tracks} tracks</span>
									{#if album.release_date}
										<span class="px-1.5 sm:px-2 py-0.5 text-[8px] sm:text-[10px] font-mono bg-sky-100 text-sky-600 rounded">{album.release_date.split('-')[0]}</span>
									{/if}
								</div>
							</div>
						</div>

						<!-- Download All Button -->
						<div class="border-t-2 border-violet-200 p-2.5 sm:p-3">
							{#if batchDownloading}
								<div class="space-y-2">
									<div class="flex items-center justify-between text-[10px] sm:text-xs font-mono">
										<span class="text-slate-600">{batchProgress.current}/{batchProgress.total}</span>
										<span>
											<span class="text-emerald-600">{batchProgress.success}</span> /
											<span class="text-red-500">{batchProgress.failed}</span>
										</span>
									</div>
									<div class="h-1.5 sm:h-2 bg-violet-100 rounded-full overflow-hidden border border-violet-200">
										<div
											class="h-full bg-gradient-to-r from-violet-500 to-sky-500 transition-all"
											style="width: {(batchProgress.current / batchProgress.total) * 100}%"
										></div>
									</div>
								</div>
							{:else}
								<button
									onclick={handleDownloadAll}
									class="flex items-center justify-center gap-2 w-full py-2 sm:py-2.5 text-xs sm:text-sm font-bold bg-gradient-to-r from-violet-500 to-sky-500 text-white border-2 border-violet-500 rounded shadow-[2px_2px_0px_0px_#c4b5fd] hover:shadow-none hover:translate-x-[2px] hover:translate-y-[2px] transition-all active:scale-[0.98]"
								>
									<Download class="w-3.5 h-3.5 sm:w-4 sm:h-4" />
									DOWNLOAD ALL ({album.total_tracks})
								</button>
							{/if}
						</div>

						<!-- Track List -->
						<div class="border-t-2 border-violet-200 max-h-48 sm:max-h-64 overflow-y-auto">
							{#each album.tracks as track, i}
								{@const status = getTrackStatus(track)}
								<div class="flex items-center gap-2 sm:gap-3 px-3 sm:px-4 py-1.5 sm:py-2 border-b border-violet-100 last:border-b-0 hover:bg-violet-50/50 transition-colors">
									<span class="w-4 sm:w-5 text-center text-[10px] sm:text-xs font-mono text-violet-400">{i + 1}</span>
									{#if track.cover_url}
										<img src={track.cover_url} alt="" class="w-7 h-7 sm:w-8 sm:h-8 rounded border border-violet-200 object-cover" />
									{:else}
										<div class="w-7 h-7 sm:w-8 sm:h-8 rounded border border-violet-200 bg-violet-50 flex items-center justify-center">
											<Music class="w-3 h-3 text-violet-400" />
										</div>
									{/if}
									<div class="flex-1 min-w-0">
										<p class="text-xs sm:text-sm text-slate-700 truncate">{track.title}</p>
										<p class="text-[10px] sm:text-xs text-slate-400 truncate">{track.artist}</p>
									</div>
									{#if track.duration_ms}
										<span class="text-[10px] font-mono text-slate-400 hidden sm:inline">{formatDuration(track.duration_ms)}</span>
									{/if}
									<div class="w-4 sm:w-5">
										{#if status?.status === 'downloading'}
											<Loader2 class="w-3.5 h-3.5 sm:w-4 sm:h-4 animate-spin text-violet-500" />
										{:else if status?.status === 'success'}
											<Check class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-emerald-500" />
										{:else if status?.status === 'error'}
											<X class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-red-500" />
										{/if}
									</div>
								</div>
							{/each}
						</div>
					</div>

				<!-- Playlist View -->
				{:else if metadata.type === 'playlist' && metadata.playlist}
					{@const playlist = metadata.playlist}
					<div class="border-2 border-violet-200 rounded-lg overflow-hidden bg-gradient-to-r from-violet-50/30 to-sky-50/30">
						<div class="flex gap-3 sm:gap-4 p-3 sm:p-4">
							{#if playlist.cover_url}
								<img
									src={playlist.cover_url}
									alt={playlist.name}
									class="w-20 h-20 sm:w-24 sm:h-24 flex-shrink-0 rounded border-2 border-violet-200 object-cover shadow-[2px_2px_0px_0px_#c4b5fd]"
								/>
							{:else}
								<div class="w-20 h-20 sm:w-24 sm:h-24 flex-shrink-0 rounded border-2 border-violet-200 bg-gradient-to-br from-violet-100 to-sky-100 flex items-center justify-center">
									<ListMusic class="w-6 h-6 sm:w-8 sm:h-8 text-violet-400" />
								</div>
							{/if}

							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2 mb-1">
									<span class="px-1.5 sm:px-2 py-0.5 text-[8px] sm:text-[10px] font-bold bg-gradient-to-r from-violet-500 to-sky-500 text-white rounded-full">PLAYLIST</span>
								</div>
								<h3 class="font-bold text-sm sm:text-base text-slate-800 truncate">{playlist.name}</h3>
								{#if playlist.owner}
									<p class="text-xs sm:text-sm text-slate-500 truncate">by {playlist.owner}</p>
								{/if}
								<div class="flex gap-1.5 sm:gap-2 mt-1.5 sm:mt-2">
									<span class="px-1.5 sm:px-2 py-0.5 text-[8px] sm:text-[10px] font-mono bg-violet-100 text-violet-600 rounded">{playlist.total_tracks} tracks</span>
								</div>
							</div>
						</div>

						<!-- Download All Button -->
						<div class="border-t-2 border-violet-200 p-2.5 sm:p-3">
							{#if batchDownloading}
								<div class="space-y-2">
									<div class="flex items-center justify-between text-[10px] sm:text-xs font-mono">
										<span class="text-slate-600">{batchProgress.current}/{batchProgress.total}</span>
										<span>
											<span class="text-emerald-600">{batchProgress.success}</span> /
											<span class="text-red-500">{batchProgress.failed}</span>
										</span>
									</div>
									<div class="h-1.5 sm:h-2 bg-violet-100 rounded-full overflow-hidden border border-violet-200">
										<div
											class="h-full bg-gradient-to-r from-violet-500 to-sky-500 transition-all"
											style="width: {(batchProgress.current / batchProgress.total) * 100}%"
										></div>
									</div>
								</div>
							{:else}
								<button
									onclick={handleDownloadAll}
									class="flex items-center justify-center gap-2 w-full py-2 sm:py-2.5 text-xs sm:text-sm font-bold bg-gradient-to-r from-violet-500 to-sky-500 text-white border-2 border-violet-500 rounded shadow-[2px_2px_0px_0px_#c4b5fd] hover:shadow-none hover:translate-x-[2px] hover:translate-y-[2px] transition-all active:scale-[0.98]"
								>
									<Download class="w-3.5 h-3.5 sm:w-4 sm:h-4" />
									DOWNLOAD ALL ({playlist.total_tracks})
								</button>
							{/if}
						</div>

						<!-- Track List -->
						<div class="border-t-2 border-violet-200 max-h-48 sm:max-h-64 overflow-y-auto">
							{#each playlist.tracks as track, i}
								{@const status = getTrackStatus(track)}
								<div class="flex items-center gap-2 sm:gap-3 px-3 sm:px-4 py-1.5 sm:py-2 border-b border-violet-100 last:border-b-0 hover:bg-violet-50/50 transition-colors">
									<span class="w-4 sm:w-5 text-center text-[10px] sm:text-xs font-mono text-violet-400">{i + 1}</span>
									{#if track.cover_url}
										<img src={track.cover_url} alt="" class="w-7 h-7 sm:w-8 sm:h-8 rounded border border-violet-200 object-cover" />
									{:else}
										<div class="w-7 h-7 sm:w-8 sm:h-8 rounded border border-violet-200 bg-violet-50 flex items-center justify-center">
											<Music class="w-3 h-3 text-violet-400" />
										</div>
									{/if}
									<div class="flex-1 min-w-0">
										<p class="text-xs sm:text-sm text-slate-700 truncate">{track.title}</p>
										<p class="text-[10px] sm:text-xs text-slate-400 truncate">{track.artist}</p>
									</div>
									{#if track.duration_ms}
										<span class="text-[10px] font-mono text-slate-400 hidden sm:inline">{formatDuration(track.duration_ms)}</span>
									{/if}
									<div class="w-4 sm:w-5">
										{#if status?.status === 'downloading'}
											<Loader2 class="w-3.5 h-3.5 sm:w-4 sm:h-4 animate-spin text-violet-500" />
										{:else if status?.status === 'success'}
											<Check class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-emerald-500" />
										{:else if status?.status === 'error'}
											<X class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-red-500" />
										{/if}
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Reset Button -->
				<button
					onclick={reset}
					class="mt-3 sm:mt-4 w-full py-1.5 sm:py-2 text-[10px] sm:text-xs font-bold text-violet-500 border-2 border-violet-200 rounded hover:border-violet-400 hover:bg-violet-50 transition-all active:scale-[0.98]"
				>
					START OVER
				</button>

			{:else if !loading && searchResults.length === 0}
				<!-- Minimal hint when nothing shown -->
				<div class="text-center py-4">
					<p class="text-[10px] sm:text-xs text-slate-400">
						Supports Tidal, Qobuz, Amazon
					</p>
				</div>
			{/if}
		</div>

		<!-- Footer -->
		<div class="border-t-2 border-violet-200 px-4 sm:px-6 py-2 sm:py-3 flex justify-between items-center text-[8px] sm:text-[10px] font-bold text-violet-400 uppercase tracking-wider bg-gradient-to-r from-violet-50/50 to-sky-50/50">
			<div class="flex items-center gap-1.5 sm:gap-2">
				<div class="relative">
					<div class="w-1.5 h-1.5 sm:w-2 sm:h-2 rounded-full bg-emerald-500 animate-ping absolute"></div>
					<div class="w-1.5 h-1.5 sm:w-2 sm:h-2 rounded-full bg-emerald-500 relative"></div>
				</div>
				<span class="font-mono">v1.0</span>
			</div>
			<span class="italic">FLAC Downloader</span>
		</div>
	</div>
</div>
