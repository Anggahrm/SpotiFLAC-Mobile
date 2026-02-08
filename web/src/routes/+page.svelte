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
	import { Download, Loader2, Music, Disc3, ListMusic, Check, X, Sparkles, Link2, Search } from 'lucide-svelte';
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

	function getInputHint() {
		const trimmed = query.trim();
		if (loading) {
			return isUrl(trimmed) ? 'Resolving link and fetching metadata…' : `Searching ${searchSource} catalog…`;
		}
		if (!trimmed) {
			return 'Paste a Spotify/Deezer URL or type a song title to start.';
		}
		if (isUrl(trimmed)) {
			return 'URL detected. Metadata is fetched automatically.';
		}
		if (trimmed.length < 2) {
			return 'Keep typing at least 2 characters to run search.';
		}
		return `Search mode active on ${searchSource}.`;
	}

	function getInputBadge() {
		const trimmed = query.trim();
		if (!trimmed) return 'IDLE';
		if (loading) return 'LIVE';
		return isUrl(trimmed) ? 'URL' : 'SEARCH';
	}

	function getInputIconMode() {
		const trimmed = query.trim();
		if (loading) return 'loading';
		if (isUrl(trimmed)) return 'url';
		if (trimmed.length >= 2) return 'search';
		return 'idle';
	}
</script>

<div class="relative min-h-screen bg-[var(--background)] text-[var(--foreground)]">
	<header class="sticky top-0 z-30 border-b border-[var(--border)] bg-[color:color-mix(in_srgb,var(--background)_78%,transparent)] backdrop-blur-xl">
		<div class="mx-auto flex h-16 max-w-[min(1800px,96vw)] items-center justify-between px-4 lg:px-8">
			<div class="flex items-center gap-3">
				<div class="waveform" aria-hidden="true">
					<span class="waveform-bar"></span><span class="waveform-bar"></span><span class="waveform-bar"></span><span class="waveform-bar"></span><span class="waveform-bar"></span>
				</div>
				<div>
					<p class="font-display text-xl leading-none text-gradient">zFlac</p>
					<p class="font-mono text-[10px] uppercase tracking-[0.24em] text-[var(--muted-foreground)]">by anggahrm</p>
				</div>
			</div>
			<a href="/docs" class="rounded-full border border-[var(--border)] px-3 py-1.5 text-xs font-medium text-[var(--muted-foreground)] transition-all hover:border-[var(--border-glow)] hover:text-[var(--foreground)]">
				API Docs
			</a>
		</div>
	</header>

	<main class="relative z-10 mx-auto max-w-[min(1800px,96vw)] px-4 py-8 lg:px-8">
		<section class="mb-6 rounded-2xl glass-card p-4 sm:p-6 fade-up">
			<div class="flex items-start justify-between gap-4">
				<div>
					<p class="mb-2 inline-flex items-center gap-1 rounded-full border border-[var(--border)] px-2.5 py-1 font-mono text-[10px] uppercase tracking-[0.18em] text-[var(--muted-foreground)]">
						<Sparkles class="h-3 w-3" /> High-Fidelity Downloader
					</p>
					<h1 class="font-display text-2xl sm:text-3xl leading-tight">Download tracks like a <span class="text-gradient">studio session</span>.</h1>
					<p class="mt-2 max-w-2xl text-sm text-[var(--muted-foreground)]">Paste a Spotify/Deezer URL or search tracks instantly. Batch downloads keep progress visible in real time.</p>
				</div>
				<div class="hidden sm:flex items-center gap-2 rounded-xl border border-[var(--border)] bg-[var(--secondary)] px-3 py-2 font-mono text-xs text-[var(--muted-foreground)]">
					<div class="relative h-2 w-2"><span class="status-dot absolute"></span></div>
					LIVE
				</div>
			</div>
		</section>

		<div class="grid gap-6 xl:grid-cols-[minmax(0,1.15fr)_minmax(520px,0.85fr)]">
			<div>
				<p class="mb-2 font-mono text-[10px] uppercase tracking-[0.16em] text-[var(--muted-foreground)]">Download Provider</p>
				<div class="grid grid-cols-4 gap-1.5 sm:gap-2 mb-4 sm:mb-5">
					{#each providers as p}
						<button
							class="px-2 sm:px-3 py-1.5 sm:py-2 text-[10px] sm:text-xs font-semibold border transition-all duration-200 rounded-md {provider === p.id
								? 'btn-primary border-transparent text-[var(--primary-foreground)]'
								: 'btn-secondary text-[var(--muted-foreground)]'}"
							onclick={() => provider = p.id}
						>
							{p.name}
						</button>
					{/each}
				</div>

				<p class="mb-2 font-mono text-[10px] uppercase tracking-[0.16em] text-[var(--muted-foreground)]">Search Source</p>
				<div class="grid grid-cols-2 gap-1.5 sm:gap-2 mb-4 sm:mb-5">
					<button
						class="px-2 sm:px-3 py-1.5 text-[10px] sm:text-xs font-semibold border transition-all duration-200 rounded-md {searchSource === 'deezer'
							? 'btn-primary border-transparent text-[var(--primary-foreground)]'
							: 'btn-secondary text-[var(--muted-foreground)]'}"
						onclick={() => searchSource = 'deezer'}
					>
						Deezer
					</button>
					<button
						class="px-2 sm:px-3 py-1.5 text-[10px] sm:text-xs font-semibold border transition-all duration-200 rounded-md {searchSource === 'spotify'
							? 'btn-primary border-transparent text-[var(--primary-foreground)]'
							: 'btn-secondary text-[var(--muted-foreground)]'}"
						onclick={() => searchSource = 'spotify'}
					>
						Spotify
					</button>
				</div>

				<div class="mb-4 sm:mb-5">
					<div class="relative">
						<input
							type="text"
							placeholder="Paste URL or type song name..."
							class="input-glow w-full h-11 sm:h-12 px-3 sm:px-4 text-xs sm:text-sm rounded-xl outline-none placeholder:text-[var(--muted-foreground)]"
							bind:value={query}
							oninput={handleInput}
						/>
						{#if loading}
							<div class="absolute right-3 top-1/2 -translate-y-1/2">
								<Loader2 class="w-4 h-4 animate-spin text-[var(--primary)]" />
							</div>
						{/if}
					</div>
					<div class="mt-2 flex items-center justify-between gap-3 rounded-lg border border-[var(--border)] bg-[var(--secondary)]/60 px-3 py-2">
						<div class="flex min-w-0 items-center gap-2">
							{#if getInputIconMode() === 'loading'}
								<Loader2 class="h-3.5 w-3.5 shrink-0 animate-spin text-[var(--primary)]" />
							{:else if getInputIconMode() === 'url'}
								<Link2 class="h-3.5 w-3.5 shrink-0 text-[var(--primary)]" />
							{:else if getInputIconMode() === 'search'}
								<Search class="h-3.5 w-3.5 shrink-0 text-[var(--primary)]" />
							{:else}
								<Sparkles class="h-3.5 w-3.5 shrink-0 text-[var(--muted-foreground)]" />
							{/if}
							<p class="truncate text-[11px] text-[var(--muted-foreground)]">{getInputHint()}</p>
						</div>
						<span class="rounded-full border border-[var(--border)] px-2 py-0.5 font-mono text-[9px] uppercase tracking-[0.14em] text-[var(--muted-foreground)]">{getInputBadge()}</span>
					</div>
				</div>
			</div>

			<div class="space-y-4">
				{#if searchResults.length > 0}
					<div class="glass-card border rounded-xl overflow-hidden mb-3 sm:mb-4 fade-up">
						<div class="px-3 sm:px-4 py-2 border-b border-[var(--border)] bg-[var(--secondary)]">
							<span class="font-mono text-[10px] sm:text-xs font-medium text-[var(--muted-foreground)]">RESULTS ({searchResults.length})</span>
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

				{#if metadata}
					{#if metadata.type === 'track' && metadata.track}
						{@const track = metadata.track}
						{@const trackStatus = getTrackStatus(track)}
						<div class="glass-card border rounded-xl overflow-hidden fade-up">
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
										class="btn-primary flex items-center justify-center gap-2 w-full py-2 sm:py-2.5 text-xs sm:text-sm rounded-lg"
									>
										<Download class="w-3.5 h-3.5 sm:w-4 sm:h-4" />
										SAVE FILE
									</a>
								{:else}
									<button
										onclick={handleDownloadTrack}
										disabled={downloading}
										class="btn-primary flex items-center justify-center gap-2 w-full py-2 sm:py-2.5 text-xs sm:text-sm rounded-lg disabled:opacity-50 disabled:cursor-not-allowed"
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

					{:else if metadata.type === 'album' && metadata.album}
						{@const album = metadata.album}
						<div class="glass-card border rounded-xl overflow-hidden fade-up">
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
										class="btn-primary flex items-center justify-center gap-2 w-full py-2 sm:py-2.5 text-xs sm:text-sm rounded-lg"
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

					{:else if metadata.type === 'playlist' && metadata.playlist}
						{@const playlist = metadata.playlist}
						<div class="glass-card border rounded-xl overflow-hidden fade-up">
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
										class="btn-primary flex items-center justify-center gap-2 w-full py-2 sm:py-2.5 text-xs sm:text-sm rounded-lg"
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

					<button
						onclick={reset}
						class="btn-secondary mt-3 sm:mt-4 w-full py-1.5 sm:py-2 text-[10px] sm:text-xs rounded-lg"
					>
						START OVER
					</button>
				{:else if !loading && searchResults.length === 0}
					<div class="glass-card rounded-xl p-4 sm:p-5 fade-up">
						<p class="font-display text-base sm:text-lg">Ready for your next track.</p>
						<p class="mt-1 text-xs sm:text-sm text-[var(--muted-foreground)]">Choose one of these quick starts:</p>
						<div class="mt-3 grid gap-2 sm:grid-cols-3">
							<div class="rounded-lg border border-[var(--border)] bg-[var(--secondary)]/60 p-2.5">
								<p class="font-mono text-[10px] uppercase tracking-[0.12em] text-[var(--muted-foreground)]">Track</p>
								<p class="mt-1 text-xs text-[var(--foreground)]">Paste a Spotify/Deezer track link for single download.</p>
							</div>
							<div class="rounded-lg border border-[var(--border)] bg-[var(--secondary)]/60 p-2.5">
								<p class="font-mono text-[10px] uppercase tracking-[0.12em] text-[var(--muted-foreground)]">Album</p>
								<p class="mt-1 text-xs text-[var(--foreground)]">Paste an album URL and pull full-track FLAC batches.</p>
							</div>
							<div class="rounded-lg border border-[var(--border)] bg-[var(--secondary)]/60 p-2.5">
								<p class="font-mono text-[10px] uppercase tracking-[0.12em] text-[var(--muted-foreground)]">Search</p>
								<p class="mt-1 text-xs text-[var(--foreground)]">Type a title/artist and select the result you want.</p>
							</div>
						</div>
						<p class="mt-3 text-[10px] sm:text-xs text-[var(--muted-foreground)]">Supports Tidal, Qobuz, and Amazon source resolution.</p>
					</div>
				{/if}
			</div>
		</div>
	</main>

	<div class="relative z-10 border-t border-[var(--border)] px-4 sm:px-6 py-3 flex justify-between items-center text-[9px] sm:text-[10px] font-medium uppercase tracking-[0.14em] bg-[var(--card)] backdrop-blur">
		<div class="flex items-center gap-2">
			<div class="relative h-2 w-2">
				<span class="status-dot absolute"></span>
			</div>
			<span class="font-mono">v1.0</span>
		</div>
		<span class="text-[var(--muted-foreground)]">FLAC Downloader</span>
	</div>
</div>
