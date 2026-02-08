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
	import BatchProgress from '$lib/components/BatchProgress.svelte';

	type Provider = 'auto' | 'tidal' | 'qobuz' | 'amazon';
	type SearchSource = 'spotify' | 'deezer';

	let query = $state('');
	let metadata: MetadataResult | null = $state(null);
	let searchResults: TrackMetadata[] = $state([]);
	let loading = $state(false);
	let provider: Provider = $state('auto');
	let searchSource: SearchSource = $state('spotify');
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

	async function handleDownloadSingleFromList(track: TrackMetadata) {
		const result = await downloadSingleTrack(track);
		if (result.success) {
			toasts.success(`Downloaded: ${track.title}`);
		} else {
			toasts.error(result.error || 'Download failed');
		}
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

<div class="min-h-screen font-mono bg-[var(--background)] text-[var(--foreground)]">
	<!-- Header -->
	<header class="sticky top-0 z-30 border-b border-[var(--border)] bg-[var(--background)]/80 backdrop-blur-sm">
		<div class="mx-auto flex h-14 max-w-6xl items-center justify-between px-4 lg:px-8">
			<div class="flex items-center gap-2">
				<Terminal class="h-5 w-5" />
				<span class="text-lg font-bold">zFlac</span>
			</div>
			<a href="/docs" class="text-sm text-[var(--muted-foreground)] hover:text-[var(--foreground)] transition-colors">
				API Docs
			</a>
		</div>
	</header>

	<!-- Main Content -->
	<main class="mx-auto max-w-6xl px-4 lg:px-8 py-8">
			<!-- Provider Selector -->
			<div class="grid grid-cols-4 gap-1.5 sm:gap-2 mb-3 sm:mb-4">
				{#each providers as p}
					<button
						class="px-2 sm:px-3 py-1.5 sm:py-2 text-[10px] sm:text-xs font-medium border transition-all duration-200 rounded-md {provider === p.id
							? 'bg-[var(--foreground)] text-[var(--background)] border-[var(--foreground)]'
							: 'bg-[var(--card)] text-[var(--muted-foreground)] border-[var(--border)] hover:border-[var(--foreground)] hover:text-[var(--foreground)]'}"
						onclick={() => provider = p.id}
					>
						{p.name}
					</button>
				{/each}
			</div>

			<!-- Search Source Toggle -->
			<div class="grid grid-cols-2 gap-1.5 sm:gap-2 mb-3 sm:mb-4">
				<button
					class="px-2 sm:px-3 py-1.5 text-[10px] sm:text-xs font-medium border transition-all duration-200 rounded-md {searchSource === 'deezer'
						? 'bg-[var(--foreground)] text-[var(--background)] border-[var(--foreground)]'
						: 'bg-[var(--card)] text-[var(--muted-foreground)] border-[var(--border)] hover:border-[var(--foreground)] hover:text-[var(--foreground)]'}"
					onclick={() => searchSource = 'deezer'}
				>
					Deezer
				</button>
				<button
					class="px-2 sm:px-3 py-1.5 text-[10px] sm:text-xs font-medium border transition-all duration-200 rounded-md {searchSource === 'spotify'
						? 'bg-[var(--foreground)] text-[var(--background)] border-[var(--foreground)]'
						: 'bg-[var(--card)] text-[var(--muted-foreground)] border-[var(--border)] hover:border-[var(--foreground)] hover:text-[var(--foreground)]'}"
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
						class="w-full h-11 sm:h-12 px-3 sm:px-4 text-xs sm:text-sm border border-[var(--border)] bg-[var(--card)] rounded-lg outline-none transition-all focus:border-[var(--foreground)] placeholder:text-[var(--muted-foreground)]"
						bind:value={query}
						oninput={handleInput}
					/>
					{#if loading}
						<div class="absolute right-3 top-1/2 -translate-y-1/2">
							<Loader2 class="w-4 h-4 animate-spin text-[var(--muted-foreground)]" />
						</div>
					{/if}
				</div>
			</div>

			<!-- Search Results -->
			{#if searchResults.length > 0}
				<div class="border border-[var(--border)] rounded-lg overflow-hidden bg-[var(--card)] mb-3 sm:mb-4">
					<div class="px-3 sm:px-4 py-2 border-b border-[var(--border)] bg-[var(--secondary)]">
						<span class="text-[10px] sm:text-xs font-medium text-[var(--muted-foreground)]">RESULTS ({searchResults.length})</span>
					</div>
					<div class="max-h-64 sm:max-h-80 overflow-y-auto">
						{#each searchResults as track, i}
							<button
								class="w-full flex items-center gap-2 sm:gap-3 px-3 sm:px-4 py-2.5 sm:py-3 border-b border-[var(--border)] last:border-b-0 hover:bg-[var(--secondary)] transition-colors text-left"
								onclick={() => selectTrack(track)}
							>
								{#if track.cover_url}
									<img src={track.cover_url} alt="" class="w-10 h-10 sm:w-12 sm:h-12 rounded border border-[var(--border)] object-cover" />
								{:else}
									<div class="w-10 h-10 sm:w-12 sm:h-12 rounded border border-[var(--border)] bg-[var(--secondary)] flex items-center justify-center">
										<Music class="w-4 h-4 sm:w-5 sm:h-5 text-[var(--muted-foreground)]" />
									</div>
								{/if}
								<div class="flex-1 min-w-0">
									<p class="text-xs sm:text-sm font-medium text-[var(--foreground)] truncate">{track.title}</p>
									<p class="text-[10px] sm:text-xs text-[var(--muted-foreground)] truncate">{track.artist}</p>
								</div>
								{#if track.duration_ms}
									<span class="text-[10px] font-mono text-[var(--muted-foreground)] hidden sm:inline">{formatDuration(track.duration_ms)}</span>
								{/if}
								<Download class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-[var(--muted-foreground)] flex-shrink-0" />
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
					<div class="border border-[var(--border)] rounded-lg overflow-hidden bg-[var(--card)]">
						<div class="flex gap-3 sm:gap-4 p-3 sm:p-4">
							{#if track.cover_url}
								<img
									src={track.cover_url}
									alt={track.title}
									class="w-20 h-20 sm:w-24 sm:h-24 flex-shrink-0 rounded border border-[var(--border)] object-cover"
								/>
							{:else}
								<div class="w-20 h-20 sm:w-24 sm:h-24 flex-shrink-0 rounded border border-[var(--border)] bg-[var(--secondary)] flex items-center justify-center">
									<Music class="w-6 h-6 sm:w-8 sm:h-8 text-[var(--muted-foreground)]" />
								</div>
							{/if}

							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2 mb-1">
									<span class="px-1.5 sm:px-2 py-0.5 text-[8px] sm:text-[10px] font-medium bg-[var(--foreground)] text-[var(--background)] rounded-full">TRACK</span>
								</div>
								<h3 class="font-bold text-sm sm:text-base text-[var(--foreground)] truncate">{track.title}</h3>
								<p class="text-xs sm:text-sm text-[var(--muted-foreground)] truncate">{track.artist}</p>
								{#if track.album}
									<p class="text-[10px] sm:text-xs text-[var(--muted-foreground)] truncate mt-0.5">{track.album}</p>
								{/if}
								<div class="flex gap-1.5 sm:gap-2 mt-1.5 sm:mt-2 flex-wrap">
									{#if track.duration_ms}
										<span class="px-1.5 sm:px-2 py-0.5 text-[8px] sm:text-[10px] font-mono bg-[var(--secondary)] text-[var(--muted-foreground)] rounded">{formatDuration(track.duration_ms)}</span>
									{/if}
									{#if track.release_date}
										<span class="px-1.5 sm:px-2 py-0.5 text-[8px] sm:text-[10px] font-mono bg-[var(--secondary)] text-[var(--muted-foreground)] rounded">{track.release_date.split('-')[0]}</span>
									{/if}
								</div>
							</div>
						</div>

						<div class="border-t border-[var(--border)] p-2.5 sm:p-3">
							{#if trackStatus?.status === 'success'}
								<a
									href={getFileUrl(trackStatus.fileName || '')}
									download
									class="flex items-center justify-center gap-2 w-full py-2 sm:py-2.5 text-xs sm:text-sm font-medium bg-emerald-600 text-white border border-emerald-600 rounded-md hover:bg-emerald-700 transition-all active:scale-[0.98]"
								>
									<Download class="w-3.5 h-3.5 sm:w-4 sm:h-4" />
									SAVE FILE
								</a>
							{:else}
								<button
									onclick={handleDownloadTrack}
									disabled={downloading}
									class="flex items-center justify-center gap-2 w-full py-2 sm:py-2.5 text-xs sm:text-sm font-medium bg-[var(--foreground)] text-[var(--background)] border border-[var(--foreground)] rounded-md hover:opacity-90 transition-all disabled:opacity-50 disabled:cursor-not-allowed active:scale-[0.98]"
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
					<div class="border border-[var(--border)] rounded-lg overflow-hidden bg-[var(--card)]">
						<div class="flex gap-3 sm:gap-4 p-3 sm:p-4">
							{#if album.cover_url}
								<img
									src={album.cover_url}
									alt={album.name}
									class="w-20 h-20 sm:w-24 sm:h-24 flex-shrink-0 rounded border border-[var(--border)] object-cover"
								/>
							{:else}
								<div class="w-20 h-20 sm:w-24 sm:h-24 flex-shrink-0 rounded border border-[var(--border)] bg-[var(--secondary)] flex items-center justify-center">
									<Disc3 class="w-6 h-6 sm:w-8 sm:h-8 text-[var(--muted-foreground)]" />
								</div>
							{/if}

							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2 mb-1">
									<span class="px-1.5 sm:px-2 py-0.5 text-[8px] sm:text-[10px] font-medium bg-[var(--foreground)] text-[var(--background)] rounded-full">ALBUM</span>
								</div>
								<h3 class="font-bold text-sm sm:text-base text-[var(--foreground)] truncate">{album.name}</h3>
								<p class="text-xs sm:text-sm text-[var(--muted-foreground)] truncate">{album.artist}</p>
								<div class="flex gap-1.5 sm:gap-2 mt-1.5 sm:mt-2 flex-wrap">
									<span class="px-1.5 sm:px-2 py-0.5 text-[8px] sm:text-[10px] font-mono bg-[var(--secondary)] text-[var(--muted-foreground)] rounded">{album.total_tracks} tracks</span>
									{#if album.release_date}
										<span class="px-1.5 sm:px-2 py-0.5 text-[8px] sm:text-[10px] font-mono bg-[var(--secondary)] text-[var(--muted-foreground)] rounded">{album.release_date.split('-')[0]}</span>
									{/if}
								</div>
							</div>
						</div>

						<!-- Download All Button -->
						<div class="border-t border-[var(--border)] p-2.5 sm:p-3">
							{#if batchDownloading}
								<BatchProgress
									current={batchProgress.current}
									total={batchProgress.total}
									success={batchProgress.success}
									failed={batchProgress.failed}
								/>
							{:else}
								<button
									onclick={handleDownloadAll}
									class="flex items-center justify-center gap-2 w-full py-2 sm:py-2.5 text-xs sm:text-sm font-medium bg-[var(--foreground)] text-[var(--background)] border border-[var(--foreground)] rounded-md hover:opacity-90 transition-all active:scale-[0.98]"
								>
									<Download class="w-3.5 h-3.5 sm:w-4 sm:h-4" />
									DOWNLOAD ALL ({album.total_tracks})
								</button>
							{/if}
						</div>

						<!-- Track List -->
						<div class="border-t border-[var(--border)] max-h-48 sm:max-h-64 overflow-y-auto">
							{#each album.tracks as track, i}
								{@const status = getTrackStatus(track)}
								<div class="flex items-center gap-2 sm:gap-3 px-3 sm:px-4 py-1.5 sm:py-2 border-b border-[var(--border)] last:border-b-0 hover:bg-[var(--secondary)] transition-colors">
									<span class="w-4 sm:w-5 text-center text-[10px] sm:text-xs font-mono text-[var(--muted-foreground)]">{i + 1}</span>
									{#if track.cover_url}
										<img src={track.cover_url} alt="" class="w-7 h-7 sm:w-8 sm:h-8 rounded border border-[var(--border)] object-cover" />
									{:else}
										<div class="w-7 h-7 sm:w-8 sm:h-8 rounded border border-[var(--border)] bg-[var(--secondary)] flex items-center justify-center">
											<Music class="w-3 h-3 text-[var(--muted-foreground)]" />
										</div>
									{/if}
									<div class="flex-1 min-w-0">
										<p class="text-xs sm:text-sm text-[var(--foreground)] truncate">{track.title}</p>
										<p class="text-[10px] sm:text-xs text-[var(--muted-foreground)] truncate">{track.artist}</p>
									</div>
									{#if track.duration_ms}
										<span class="text-[10px] font-mono text-[var(--muted-foreground)] hidden sm:inline">{formatDuration(track.duration_ms)}</span>
									{/if}
									<div class="w-4 sm:w-5">
										{#if status?.status === 'downloading'}
											<Loader2 class="w-3.5 h-3.5 sm:w-4 sm:h-4 animate-spin text-[var(--muted-foreground)]" />
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
					<div class="border border-[var(--border)] rounded-lg overflow-hidden bg-[var(--card)]">
						<div class="flex gap-3 sm:gap-4 p-3 sm:p-4">
							{#if playlist.cover_url}
								<img
									src={playlist.cover_url}
									alt={playlist.name}
									class="w-20 h-20 sm:w-24 sm:h-24 flex-shrink-0 rounded border border-[var(--border)] object-cover"
								/>
							{:else}
								<div class="w-20 h-20 sm:w-24 sm:h-24 flex-shrink-0 rounded border border-[var(--border)] bg-[var(--secondary)] flex items-center justify-center">
									<ListMusic class="w-6 h-6 sm:w-8 sm:h-8 text-[var(--muted-foreground)]" />
								</div>
							{/if}

							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2 mb-1">
									<span class="px-1.5 sm:px-2 py-0.5 text-[8px] sm:text-[10px] font-medium bg-[var(--foreground)] text-[var(--background)] rounded-full">PLAYLIST</span>
								</div>
								<h3 class="font-bold text-sm sm:text-base text-[var(--foreground)] truncate">{playlist.name}</h3>
								{#if playlist.owner}
									<p class="text-xs sm:text-sm text-[var(--muted-foreground)] truncate">by {playlist.owner}</p>
								{/if}
								<div class="flex gap-1.5 sm:gap-2 mt-1.5 sm:mt-2">
									<span class="px-1.5 sm:px-2 py-0.5 text-[8px] sm:text-[10px] font-mono bg-[var(--secondary)] text-[var(--muted-foreground)] rounded">{playlist.total_tracks} tracks</span>
								</div>
							</div>
						</div>

						<!-- Download All Button -->
						<div class="border-t border-[var(--border)] p-2.5 sm:p-3">
							{#if batchDownloading}
								<BatchProgress
									current={batchProgress.current}
									total={batchProgress.total}
									success={batchProgress.success}
									failed={batchProgress.failed}
								/>
							{:else}
								<button
									onclick={handleDownloadAll}
									class="flex items-center justify-center gap-2 w-full py-2 sm:py-2.5 text-xs sm:text-sm font-medium bg-[var(--foreground)] text-[var(--background)] border border-[var(--foreground)] rounded-md hover:opacity-90 transition-all active:scale-[0.98]"
								>
									<Download class="w-3.5 h-3.5 sm:w-4 sm:h-4" />
									DOWNLOAD ALL ({playlist.total_tracks})
								</button>
							{/if}
						</div>

						<!-- Track List -->
						<div class="border-t border-[var(--border)] max-h-48 sm:max-h-64 overflow-y-auto">
							{#each playlist.tracks as track, i}
								{@const status = getTrackStatus(track)}
								<button
									class="w-full flex items-center gap-2 sm:gap-3 px-3 sm:px-4 py-1.5 sm:py-2 border-b border-[var(--border)] last:border-b-0 hover:bg-[var(--secondary)] transition-colors text-left disabled:opacity-50"
									onclick={() => handleDownloadSingleFromList(track)}
									disabled={status?.status === 'downloading' || status?.status === 'success'}
								>
									<span class="w-4 sm:w-5 text-center text-[10px] sm:text-xs font-mono text-[var(--muted-foreground)]">{i + 1}</span>
									{#if track.cover_url}
										<img src={track.cover_url} alt="" class="w-7 h-7 sm:w-8 sm:h-8 rounded border border-[var(--border)] object-cover" />
									{:else}
										<div class="w-7 h-7 sm:w-8 sm:h-8 rounded border border-[var(--border)] bg-[var(--secondary)] flex items-center justify-center">
											<Music class="w-3 h-3 text-[var(--muted-foreground)]" />
										</div>
									{/if}
									<div class="flex-1 min-w-0">
										<p class="text-xs sm:text-sm text-[var(--foreground)] truncate">{track.title}</p>
										<p class="text-[10px] sm:text-xs text-[var(--muted-foreground)] truncate">{track.artist}</p>
									</div>
									{#if track.duration_ms}
										<span class="text-[10px] font-mono text-[var(--muted-foreground)] hidden sm:inline">{formatDuration(track.duration_ms)}</span>
									{/if}
									<div class="w-5 sm:w-6 flex justify-center">
										{#if status?.status === 'downloading'}
											<Loader2 class="w-3.5 h-3.5 sm:w-4 sm:h-4 animate-spin text-[var(--muted-foreground)]" />
										{:else if status?.status === 'success'}
											<Check class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-emerald-500" />
										{:else if status?.status === 'error'}
											<X class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-red-500" />
										{:else}
											<Download class="w-3.5 h-3.5 sm:w-4 sm:h-4 text-[var(--muted-foreground)]" />
										{/if}
									</div>
								</button>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Reset Button -->
				<button
					onclick={reset}
					class="mt-3 sm:mt-4 w-full py-1.5 sm:py-2 text-[10px] sm:text-xs font-medium text-[var(--muted-foreground)] border border-[var(--border)] rounded-md hover:border-[var(--foreground)] hover:text-[var(--foreground)] transition-all active:scale-[0.98]"
				>
					START OVER
				</button>

			{:else if !loading && searchResults.length === 0}
				<!-- Minimal hint when nothing shown -->
				<div class="text-center py-4">
					<p class="text-[10px] sm:text-xs text-[var(--muted-foreground)]">
						Supports Tidal, Qobuz, Amazon
					</p>
				</div>
			{/if}
	</main>

	<!-- Footer -->
	<div class="border-t border-[var(--border)] px-4 sm:px-6 py-2 sm:py-3 flex justify-between items-center text-[8px] sm:text-[10px] font-medium text-[var(--muted-foreground)] uppercase tracking-wider bg-[var(--card)]">
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
