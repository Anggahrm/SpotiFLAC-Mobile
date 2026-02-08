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
			return isUrl(trimmed) ? 'Resolving link...' : `Searching ${searchSource}...`;
		}
		if (!trimmed) {
			return 'Paste a Spotify or Deezer URL, or type a song name';
		}
		if (isUrl(trimmed)) {
			return 'URL detected — fetching metadata';
		}
		if (trimmed.length < 2) {
			return 'Type at least 2 characters to search';
		}
		return `Searching on ${searchSource}`;
	}

	function getInputBadge() {
		const trimmed = query.trim();
		if (!trimmed) return 'Ready';
		if (loading) return 'Loading';
		return isUrl(trimmed) ? 'URL' : 'Search';
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
	<!-- Elegant Header -->
	<header class="sticky top-0 z-30 border-b border-[var(--border)] bg-[color:color-mix(in_srgb,var(--background)_85%,transparent)] backdrop-blur-2xl">
		<div class="mx-auto flex h-20 max-w-[min(1400px,94vw)] items-center justify-between px-6 lg:px-12">
			<div class="flex items-center gap-4">
				<div class="flex items-center gap-3">
					<div class="waveform" aria-hidden="true">
						<span class="waveform-bar"></span><span class="waveform-bar"></span><span class="waveform-bar"></span><span class="waveform-bar"></span><span class="waveform-bar"></span>
					</div>
				</div>
				<div>
					<p class="font-display text-2xl tracking-tight text-gradient">zFlac</p>
				</div>
			</div>
			<div class="flex items-center gap-6">
				<div class="hidden sm:flex items-center gap-2 text-[11px] font-mono uppercase tracking-[0.2em] text-[var(--muted-foreground)]">
					<span class="relative h-1.5 w-1.5">
						<span class="status-dot absolute"></span>
					</span>
					Live
				</div>
				<a href="/docs" class="text-xs font-medium tracking-wide text-[var(--muted-foreground)] hover:text-[var(--foreground)] transition-colors">
					Documentation
				</a>
			</div>
		</div>
	</header>

	<!-- Hero Section -->
	<main class="relative z-10 mx-auto max-w-[min(1400px,94vw)] px-6 lg:px-12">
		<!-- Hero -->
		<section class="pt-16 pb-12 lg:pt-24 lg:pb-16">
			<div class="max-w-2xl">
				<p class="mb-4 font-mono text-[11px] uppercase tracking-[0.25em] text-[var(--primary)]">
					High-Fidelity Audio
				</p>
				<h1 class="font-display text-4xl sm:text-5xl lg:text-6xl leading-[1.1] tracking-tight">
					Download music in <span class="text-gradient italic">studio quality</span>
				</h1>
				<p class="mt-6 text-base sm:text-lg text-[var(--muted-foreground)] leading-relaxed max-w-xl">
					Lossless FLAC downloads from Tidal, Qobuz, and Amazon. Paste a link or search directly — your music, uncompromised.
				</p>
			</div>
		</section>

		<!-- Main Interface -->
		<section class="pb-16 lg:pb-24">
			<div class="grid gap-8 lg:grid-cols-[1fr_1.2fr] lg:gap-12">
				<!-- Left Column: Controls -->
				<div class="space-y-8">
					<!-- Provider Selection -->
					<div class="glass-card rounded-2xl p-6 lg:p-8">
						<p class="mb-4 font-mono text-[10px] uppercase tracking-[0.2em] text-[var(--muted-foreground)]">Source Provider</p>
						<div class="grid grid-cols-4 gap-2">
							{#each providers as p}
								<button
									class="px-3 py-2.5 text-[11px] font-semibold tracking-wide border transition-all duration-300 rounded-lg {provider === p.id
										? 'btn-primary border-transparent'
										: 'btn-secondary text-[var(--muted-foreground)]'}"
									onclick={() => provider = p.id}
								>
									{p.name}
								</button>
							{/each}
						</div>
					</div>

					<!-- Search Source -->
					<div class="glass-card rounded-2xl p-6 lg:p-8">
						<p class="mb-4 font-mono text-[10px] uppercase tracking-[0.2em] text-[var(--muted-foreground)]">Search Source</p>
						<div class="grid grid-cols-2 gap-3">
							<button
								class="px-4 py-3 text-xs font-medium tracking-wide border transition-all duration-300 rounded-lg {searchSource === 'deezer'
									? 'btn-primary border-transparent'
									: 'btn-secondary text-[var(--muted-foreground)]'}"
								onclick={() => searchSource = 'deezer'}
							>
								Deezer
							</button>
							<button
								class="px-4 py-3 text-xs font-medium tracking-wide border transition-all duration-300 rounded-lg {searchSource === 'spotify'
									? 'btn-primary border-transparent'
									: 'btn-secondary text-[var(--muted-foreground)]'}"
								onclick={() => searchSource = 'spotify'}
							>
								Spotify
							</button>
						</div>
					</div>

					<!-- Input -->
					<div class="glass-card rounded-2xl p-6 lg:p-8">
						<p class="mb-4 font-mono text-[10px] uppercase tracking-[0.2em] text-[var(--muted-foreground)]">Input</p>
						<div class="relative">
							<input
								type="text"
								placeholder="Paste URL or type song name..."
								class="input-glow w-full h-14 px-5 text-sm rounded-xl outline-none placeholder:text-[var(--muted-foreground)] bg-[var(--secondary)]/50"
								bind:value={query}
								oninput={handleInput}
							/>
							{#if loading}
								<div class="absolute right-5 top-1/2 -translate-y-1/2">
									<Loader2 class="w-5 h-5 animate-spin text-[var(--primary)]" />
								</div>
							{/if}
						</div>
						<div class="mt-4 flex items-center justify-between">
							<div class="flex items-center gap-2 text-[11px] text-[var(--muted-foreground)]">
								{#if getInputIconMode() === 'loading'}
									<Loader2 class="h-3.5 w-3.5 animate-spin text-[var(--primary)]" />
								{:else if getInputIconMode() === 'url'}
									<Link2 class="h-3.5 w-3.5 text-[var(--primary)]" />
								{:else if getInputIconMode() === 'search'}
									<Search class="h-3.5 w-3.5 text-[var(--primary)]" />
								{:else}
									<Sparkles class="h-3.5 w-3.5" />
								{/if}
								<span>{getInputHint()}</span>
							</div>
							<span class="font-mono text-[10px] uppercase tracking-[0.15em] text-[var(--muted-foreground)]">{getInputBadge()}</span>
						</div>
					</div>

					<!-- Quick Info -->
					<div class="grid grid-cols-3 gap-4">
						<div class="text-center py-4">
							<p class="font-display text-2xl text-gradient">24-bit</p>
							<p class="mt-1 text-[10px] font-mono uppercase tracking-[0.15em] text-[var(--muted-foreground)]">Hi-Res</p>
						</div>
						<div class="text-center py-4 border-x border-[var(--border)]">
							<p class="font-display text-2xl text-gradient">192</p>
							<p class="mt-1 text-[10px] font-mono uppercase tracking-[0.15em] text-[var(--muted-foreground)]">kHz</p>
						</div>
						<div class="text-center py-4">
							<p class="font-display text-2xl text-gradient">3</p>
							<p class="mt-1 text-[10px] font-mono uppercase tracking-[0.15em] text-[var(--muted-foreground)]">Sources</p>
						</div>
					</div>
				</div>

				<!-- Right Column: Results -->
				<div class="space-y-6">
					<!-- Search Results -->
					{#if searchResults.length > 0}
						<div class="glass-card rounded-2xl overflow-hidden fade-up">
							<div class="px-6 py-4 border-b border-[var(--border)] bg-[var(--secondary)]/30">
								<p class="font-mono text-[10px] uppercase tracking-[0.2em] text-[var(--muted-foreground)]">Search Results <span class="text-[var(--foreground)]">({searchResults.length})</span></p>
							</div>
							<div class="max-h-80 overflow-y-auto">
								{#each searchResults as track}
									<button
										class="w-full flex items-center gap-4 px-6 py-4 border-b border-[var(--border)] last:border-b-0 hover:bg-[var(--secondary)]/50 transition-all text-left group"
										onclick={() => selectTrack(track)}
									>
										{#if track.cover_url}
											<img src={track.cover_url} alt="" class="w-12 h-12 rounded-lg border border-[var(--border)] object-cover group-hover:scale-105 transition-transform" />
										{:else}
											<div class="w-12 h-12 rounded-lg border border-[var(--border)] bg-[var(--secondary)] flex items-center justify-center">
												<Music class="w-5 h-5 text-[var(--muted-foreground)]" />
											</div>
										{/if}
										<div class="flex-1 min-w-0">
											<p class="text-sm font-medium text-[var(--foreground)] truncate group-hover:text-[var(--primary)] transition-colors">{track.title}</p>
											<p class="text-xs text-[var(--muted-foreground)] truncate">{track.artist}</p>
										</div>
										{#if track.duration_ms}
											<span class="text-[11px] font-mono text-[var(--muted-foreground)]">{formatDuration(track.duration_ms)}</span>
										{/if}
										<Download class="w-4 h-4 text-[var(--muted-foreground)] opacity-0 group-hover:opacity-100 transition-opacity" />
									</button>
								{/each}
							</div>
						</div>
					{/if}

					<!-- Track/Album/Playlist Display -->
					{#if metadata}
						{#if metadata.type === 'track' && metadata.track}
							{@const track = metadata.track}
							{@const trackStatus = getTrackStatus(track)}
							<div class="glass-card rounded-2xl overflow-hidden fade-up luxury-hover">
								<div class="p-6 lg:p-8">
									<div class="flex gap-6">
										{#if track.cover_url}
											<img
												src={track.cover_url}
												alt={track.title}
												class="w-28 h-28 lg:w-36 lg:h-36 flex-shrink-0 rounded-xl border border-[var(--border)] object-cover shadow-2xl"
											/>
										{:else}
											<div class="w-28 h-28 lg:w-36 lg:h-36 flex-shrink-0 rounded-xl border border-[var(--border)] bg-[var(--secondary)] flex items-center justify-center">
												<Music class="w-10 h-10 text-[var(--muted-foreground)]" />
											</div>
										{/if}

										<div class="flex-1 min-w-0">
											<span class="inline-block px-2.5 py-1 text-[9px] font-semibold uppercase tracking-[0.15em] bg-[var(--foreground)] text-[var(--background)] rounded-full mb-3">Track</span>
											<h3 class="font-display text-xl lg:text-2xl text-[var(--foreground)] leading-tight">{track.title}</h3>
											<p class="mt-1 text-sm text-[var(--muted-foreground)]">{track.artist}</p>
											{#if track.album}
												<p class="text-xs text-[var(--muted-foreground)] mt-1">{track.album}</p>
											{/if}
											<div class="flex gap-2 mt-4">
												{#if track.duration_ms}
													<span class="px-2 py-1 text-[10px] font-mono bg-[var(--secondary)] text-[var(--muted-foreground)] rounded-md">{formatDuration(track.duration_ms)}</span>
												{/if}
												{#if track.release_date}
													<span class="px-2 py-1 text-[10px] font-mono bg-[var(--secondary)] text-[var(--muted-foreground)] rounded-md">{track.release_date.split('-')[0]}</span>
												{/if}
											</div>
										</div>
									</div>
								</div>

								<div class="border-t border-[var(--border)] p-6">
									{#if trackStatus?.status === 'success'}
										<a
											href={getFileUrl(trackStatus.fileName || '')}
											download
											class="btn-primary flex items-center justify-center gap-3 w-full py-3.5 text-sm font-semibold tracking-wide rounded-xl"
										>
											<Download class="w-4 h-4" />
											Save File
										</a>
									{:else}
										<button
											onclick={handleDownloadTrack}
											disabled={downloading}
											class="btn-primary flex items-center justify-center gap-3 w-full py-3.5 text-sm font-semibold tracking-wide rounded-xl disabled:opacity-50 disabled:cursor-not-allowed"
										>
											{#if downloading}
												<Loader2 class="w-4 h-4 animate-spin" />
												Processing...
											{:else}
												<Download class="w-4 h-4" />
												Download FLAC
											{/if}
										</button>
									{/if}
								</div>
							</div>

						{:else if metadata.type === 'album' && metadata.album}
							{@const album = metadata.album}
							<div class="glass-card rounded-2xl overflow-hidden fade-up luxury-hover">
								<div class="p-6 lg:p-8">
									<div class="flex gap-6">
										{#if album.cover_url}
											<img
												src={album.cover_url}
												alt={album.name}
												class="w-28 h-28 lg:w-36 lg:h-36 flex-shrink-0 rounded-xl border border-[var(--border)] object-cover shadow-2xl"
											/>
										{:else}
											<div class="w-28 h-28 lg:w-36 lg:h-36 flex-shrink-0 rounded-xl border border-[var(--border)] bg-[var(--secondary)] flex items-center justify-center">
												<Disc3 class="w-10 h-10 text-[var(--muted-foreground)]" />
											</div>
										{/if}

										<div class="flex-1 min-w-0">
											<span class="inline-block px-2.5 py-1 text-[9px] font-semibold uppercase tracking-[0.15em] bg-[var(--foreground)] text-[var(--background)] rounded-full mb-3">Album</span>
											<h3 class="font-display text-xl lg:text-2xl text-[var(--foreground)] leading-tight">{album.name}</h3>
											<p class="mt-1 text-sm text-[var(--muted-foreground)]">{album.artist}</p>
											<div class="flex gap-2 mt-4">
												<span class="px-2 py-1 text-[10px] font-mono bg-[var(--secondary)] text-[var(--muted-foreground)] rounded-md">{album.total_tracks} tracks</span>
												{#if album.release_date}
													<span class="px-2 py-1 text-[10px] font-mono bg-[var(--secondary)] text-[var(--muted-foreground)] rounded-md">{album.release_date.split('-')[0]}</span>
												{/if}
											</div>
										</div>
									</div>
								</div>

								<div class="border-t border-[var(--border)] p-6">
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
											class="btn-primary flex items-center justify-center gap-3 w-full py-3.5 text-sm font-semibold tracking-wide rounded-xl"
										>
											<Download class="w-4 h-4" />
											Download All ({album.total_tracks})
										</button>
									{/if}
								</div>

								<div class="border-t border-[var(--border)] max-h-64 overflow-y-auto">
									{#each album.tracks as track, i}
										{@const status = getTrackStatus(track)}
										<div class="flex items-center gap-4 px-6 py-3 border-b border-[var(--border)] last:border-b-0 hover:bg-[var(--secondary)]/30 transition-colors">
											<span class="w-5 text-center text-[11px] font-mono text-[var(--muted-foreground)]">{i + 1}</span>
											{#if track.cover_url}
												<img src={track.cover_url} alt="" class="w-9 h-9 rounded-md border border-[var(--border)] object-cover" />
											{:else}
												<div class="w-9 h-9 rounded-md border border-[var(--border)] bg-[var(--secondary)] flex items-center justify-center">
													<Music class="w-3.5 h-3.5 text-[var(--muted-foreground)]" />
												</div>
											{/if}
											<div class="flex-1 min-w-0">
												<p class="text-sm text-[var(--foreground)] truncate">{track.title}</p>
												<p class="text-[11px] text-[var(--muted-foreground)] truncate">{track.artist}</p>
											</div>
											{#if track.duration_ms}
												<span class="text-[11px] font-mono text-[var(--muted-foreground)]">{formatDuration(track.duration_ms)}</span>
											{/if}
											<div class="w-5">
												{#if status?.status === 'downloading'}
													<Loader2 class="w-4 h-4 animate-spin text-[var(--muted-foreground)]" />
												{:else if status?.status === 'success'}
													<Check class="w-4 h-4 text-emerald-500" />
												{:else if status?.status === 'error'}
													<X class="w-4 h-4 text-red-500" />
												{/if}
											</div>
										</div>
									{/each}
								</div>
							</div>

						{:else if metadata.type === 'playlist' && metadata.playlist}
							{@const playlist = metadata.playlist}
							<div class="glass-card rounded-2xl overflow-hidden fade-up luxury-hover">
								<div class="p-6 lg:p-8">
									<div class="flex gap-6">
										{#if playlist.cover_url}
											<img
												src={playlist.cover_url}
												alt={playlist.name}
												class="w-28 h-28 lg:w-36 lg:h-36 flex-shrink-0 rounded-xl border border-[var(--border)] object-cover shadow-2xl"
											/>
										{:else}
											<div class="w-28 h-28 lg:w-36 lg:h-36 flex-shrink-0 rounded-xl border border-[var(--border)] bg-[var(--secondary)] flex items-center justify-center">
												<ListMusic class="w-10 h-10 text-[var(--muted-foreground)]" />
											</div>
										{/if}

										<div class="flex-1 min-w-0">
											<span class="inline-block px-2.5 py-1 text-[9px] font-semibold uppercase tracking-[0.15em] bg-[var(--foreground)] text-[var(--background)] rounded-full mb-3">Playlist</span>
											<h3 class="font-display text-xl lg:text-2xl text-[var(--foreground)] leading-tight">{playlist.name}</h3>
											{#if playlist.owner}
												<p class="mt-1 text-sm text-[var(--muted-foreground)]">by {playlist.owner}</p>
											{/if}
											<div class="flex gap-2 mt-4">
												<span class="px-2 py-1 text-[10px] font-mono bg-[var(--secondary)] text-[var(--muted-foreground)] rounded-md">{playlist.total_tracks} tracks</span>
											</div>
										</div>
									</div>
								</div>

								<div class="border-t border-[var(--border)] p-6">
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
											class="btn-primary flex items-center justify-center gap-3 w-full py-3.5 text-sm font-semibold tracking-wide rounded-xl"
										>
											<Download class="w-4 h-4" />
											Download All ({playlist.total_tracks})
										</button>
									{/if}
								</div>

								<div class="border-t border-[var(--border)] max-h-64 overflow-y-auto">
									{#each playlist.tracks as track, i}
										{@const status = getTrackStatus(track)}
										<button
											class="w-full flex items-center gap-4 px-6 py-3 border-b border-[var(--border)] last:border-b-0 hover:bg-[var(--secondary)]/30 transition-colors text-left disabled:opacity-50"
											onclick={() => handleDownloadSingleFromList(track)}
											disabled={status?.status === 'downloading' || status?.status === 'success'}
										>
											<span class="w-5 text-center text-[11px] font-mono text-[var(--muted-foreground)]">{i + 1}</span>
											{#if track.cover_url}
												<img src={track.cover_url} alt="" class="w-9 h-9 rounded-md border border-[var(--border)] object-cover" />
											{:else}
												<div class="w-9 h-9 rounded-md border border-[var(--border)] bg-[var(--secondary)] flex items-center justify-center">
													<Music class="w-3.5 h-3.5 text-[var(--muted-foreground)]" />
												</div>
											{/if}
											<div class="flex-1 min-w-0">
												<p class="text-sm text-[var(--foreground)] truncate">{track.title}</p>
												<p class="text-[11px] text-[var(--muted-foreground)] truncate">{track.artist}</p>
											</div>
											{#if track.duration_ms}
												<span class="text-[11px] font-mono text-[var(--muted-foreground)]">{formatDuration(track.duration_ms)}</span>
											{/if}
											<div class="w-6 flex justify-center">
												{#if status?.status === 'downloading'}
													<Loader2 class="w-4 h-4 animate-spin text-[var(--muted-foreground)]" />
												{:else if status?.status === 'success'}
													<Check class="w-4 h-4 text-emerald-500" />
												{:else if status?.status === 'error'}
													<X class="w-4 h-4 text-red-500" />
												{:else}
													<Download class="w-4 h-4 text-[var(--muted-foreground)]" />
												{/if}
											</div>
										</button>
									{/each}
								</div>
							</div>
						{/if}

						<button
							onclick={reset}
							class="btn-secondary w-full py-3 text-xs font-medium tracking-wide rounded-xl"
						>
							Start Over
						</button>
					{:else if !loading && searchResults.length === 0}
						<div class="glass-card rounded-2xl p-8 lg:p-12 text-center">
							<div class="w-16 h-16 mx-auto mb-6 rounded-2xl bg-[var(--secondary)] flex items-center justify-center">
								<Music class="w-8 h-8 text-[var(--primary)]" />
							</div>
							<h3 class="font-display text-xl text-[var(--foreground)]">Ready to download</h3>
							<p class="mt-2 text-sm text-[var(--muted-foreground)] max-w-sm mx-auto">
								Paste a Spotify or Deezer URL, or search for tracks by name. Supports single tracks, albums, and playlists.
							</p>
							<div class="mt-8 flex justify-center gap-6 text-[11px] font-mono uppercase tracking-[0.15em] text-[var(--muted-foreground)]">
								<span>Track</span>
								<span class="text-[var(--border)]">|</span>
								<span>Album</span>
								<span class="text-[var(--border)]">|</span>
								<span>Playlist</span>
							</div>
						</div>
					{/if}
				</div>
			</div>
		</section>
	</main>

	<!-- Elegant Footer -->
	<footer class="relative z-10 border-t border-[var(--border)]">
		<div class="mx-auto max-w-[min(1400px,94vw)] px-6 lg:px-12 py-8">
			<div class="flex flex-col sm:flex-row justify-between items-center gap-4">
				<div class="flex items-center gap-3">
					<p class="font-display text-lg text-gradient">zFlac</p>
					<span class="text-[var(--border)]">|</span>
					<p class="text-xs text-[var(--muted-foreground)]">Lossless audio downloader</p>
				</div>
				<div class="flex items-center gap-2 text-[11px] font-mono uppercase tracking-[0.15em] text-[var(--muted-foreground)]">
					<span class="relative h-1.5 w-1.5">
						<span class="status-dot absolute"></span>
					</span>
					<span>v1.0</span>
				</div>
			</div>
		</div>
	</footer>
</div>
