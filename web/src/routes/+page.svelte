<script lang="ts">
	import {
		getMetadata,
		downloadTrack,
		getFileUrl,
		formatDuration,
		checkAvailability,
		type MetadataResult,
		type TrackMetadata
	} from '$lib/api';
	import { toasts } from '$lib/stores/toasts';
	import { Download, Loader2, Music, Disc3, ListMusic, Check, X, ChevronDown } from 'lucide-svelte';

	type Provider = 'auto' | 'tidal' | 'qobuz' | 'amazon';

	let url = $state('');
	let metadata: MetadataResult | null = $state(null);
	let loading = $state(false);
	let provider: Provider = $state('auto');
	let showProviderMenu = $state(false);

	// Download states
	let downloading = $state(false);
	let batchDownloading = $state(false);
	let batchProgress = $state({ current: 0, total: 0, success: 0, failed: 0 });
	let downloadResults: Map<string, { status: 'pending' | 'downloading' | 'success' | 'error'; fileName?: string }> = $state(new Map());

	const providers: { id: Provider; name: string }[] = [
		{ id: 'auto', name: 'Auto (Best Available)' },
		{ id: 'tidal', name: 'Tidal' },
		{ id: 'qobuz', name: 'Qobuz' },
		{ id: 'amazon', name: 'Amazon Music' }
	];

	async function handleSubmit(e: Event) {
		e.preventDefault();
		await fetchMetadata();
	}

	async function fetchMetadata() {
		const trimmedUrl = url.trim();
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
			let service = provider;
			if (provider === 'auto') {
				const avail = await checkAvailability(track.spotify_id, track.isrc);
				if (avail.tidal) service = 'tidal';
				else if (avail.qobuz) service = 'qobuz';
				else if (avail.amazon) service = 'amazon';
				else throw new Error('Not available on any service');
			}

			const result = await downloadTrack({
				track_name: track.title,
				artist_name: track.artist,
				album_name: track.album,
				album_artist: track.album_artist,
				cover_url: track.cover_url,
				spotify_id: track.spotify_id,
				isrc: track.isrc,
				service: service === 'auto' ? undefined : service,
				item_id: crypto.randomUUID(),
				duration_ms: track.duration_ms
			});

			if (result.success && result.file_name) {
				downloadResults.set(trackKey, { status: 'success', fileName: result.file_name });
				downloadResults = new Map(downloadResults);

				const link = document.createElement('a');
				link.href = getFileUrl(result.file_name);
				link.download = result.file_name;
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
		url = '';
		metadata = null;
		downloadResults = new Map();
	}
</script>

<div class="min-h-screen bg-black">
	<!-- Header -->
	<header class="border-b border-zinc-800 bg-zinc-950">
		<div class="mx-auto flex h-16 max-w-4xl items-center justify-between px-4">
			<div class="flex items-center gap-3">
				<div class="flex h-9 w-9 items-center justify-center rounded-lg bg-emerald-500">
					<Music class="h-5 w-5 text-black" />
				</div>
				<span class="text-lg font-semibold text-white">zFlac Downloader</span>
			</div>

			<!-- Provider selector -->
			<div class="relative">
				<button
					class="flex items-center gap-2 rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-zinc-300 transition-colors hover:bg-zinc-800"
					onclick={() => showProviderMenu = !showProviderMenu}
				>
					{providers.find(p => p.id === provider)?.name}
					<ChevronDown class="h-4 w-4" />
				</button>

				{#if showProviderMenu}
					<div class="absolute right-0 top-full z-50 mt-1 w-48 rounded-lg border border-zinc-700 bg-zinc-900 py-1 shadow-xl">
						{#each providers as p}
							<button
								class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-zinc-300 hover:bg-zinc-800"
								onclick={() => { provider = p.id; showProviderMenu = false; }}
							>
								{#if provider === p.id}
									<Check class="h-4 w-4 text-emerald-500" />
								{:else}
									<span class="w-4"></span>
								{/if}
								{p.name}
							</button>
						{/each}
					</div>
				{/if}
			</div>
		</div>
	</header>

	<main class="mx-auto max-w-4xl px-4 py-8">
		<!-- URL Input -->
		<form onsubmit={handleSubmit} class="mb-8">
			<div class="relative">
				<input
					type="url"
					placeholder="Paste Spotify or Deezer URL (track, album, or playlist)"
					class="h-14 w-full rounded-xl border border-zinc-700 bg-zinc-900 px-4 pr-24 text-white placeholder-zinc-500 outline-none transition-colors focus:border-emerald-500"
					bind:value={url}
				/>
				<button
					type="submit"
					disabled={loading || !url.trim()}
					class="absolute right-2 top-1/2 -translate-y-1/2 rounded-lg bg-emerald-500 px-4 py-2 font-medium text-black transition-all hover:bg-emerald-400 disabled:cursor-not-allowed disabled:opacity-50"
				>
					{#if loading}
						<Loader2 class="h-5 w-5 animate-spin" />
					{:else}
						Fetch
					{/if}
				</button>
			</div>
		</form>

		<!-- Content -->
		{#if metadata}
			<!-- Track View -->
			{#if metadata.type === 'track' && metadata.track}
				{@const track = metadata.track}
				{@const trackStatus = getTrackStatus(track)}
				<div class="overflow-hidden rounded-xl border border-zinc-800 bg-zinc-900">
					<div class="flex gap-4 p-4 sm:p-6">
						{#if track.cover_url}
							<img
								src={track.cover_url}
								alt={track.title}
								class="h-32 w-32 flex-shrink-0 rounded-lg object-cover shadow-lg sm:h-40 sm:w-40"
							/>
						{:else}
							<div class="flex h-32 w-32 flex-shrink-0 items-center justify-center rounded-lg bg-zinc-800 sm:h-40 sm:w-40">
								<Music class="h-12 w-12 text-zinc-600" />
							</div>
						{/if}

						<div class="flex min-w-0 flex-1 flex-col justify-between">
							<div>
								<h2 class="truncate text-xl font-bold text-white sm:text-2xl">{track.title}</h2>
								<p class="mt-1 truncate text-zinc-400">{track.artist}</p>
								{#if track.album}
									<p class="mt-0.5 truncate text-sm text-zinc-500">{track.album}</p>
								{/if}
							</div>

							<div class="mt-4 flex flex-wrap items-center gap-2 text-xs text-zinc-500">
								{#if track.duration_ms}
									<span class="rounded bg-zinc-800 px-2 py-1">{formatDuration(track.duration_ms)}</span>
								{/if}
								{#if track.release_date}
									<span class="rounded bg-zinc-800 px-2 py-1">{track.release_date.split('-')[0]}</span>
								{/if}
							</div>
						</div>
					</div>

					<div class="border-t border-zinc-800 p-4">
						{#if trackStatus?.status === 'success'}
							<a
								href={getFileUrl(trackStatus.fileName || '')}
								download
								class="flex w-full items-center justify-center gap-2 rounded-lg bg-emerald-500 py-3 font-medium text-black transition-all hover:bg-emerald-400"
							>
								<Download class="h-5 w-5" />
								Save File
							</a>
						{:else}
							<button
								onclick={handleDownloadTrack}
								disabled={downloading}
								class="flex w-full items-center justify-center gap-2 rounded-lg bg-emerald-500 py-3 font-medium text-black transition-all hover:bg-emerald-400 disabled:cursor-not-allowed disabled:opacity-50"
							>
								{#if downloading}
									<Loader2 class="h-5 w-5 animate-spin" />
									Downloading...
								{:else}
									<Download class="h-5 w-5" />
									Download FLAC
								{/if}
							</button>
						{/if}
					</div>
				</div>

			<!-- Album View -->
			{:else if metadata.type === 'album' && metadata.album}
				{@const album = metadata.album}
				<div class="overflow-hidden rounded-xl border border-zinc-800 bg-zinc-900">
					<div class="flex gap-4 p-4 sm:p-6">
						{#if album.cover_url}
							<img
								src={album.cover_url}
								alt={album.name}
								class="h-32 w-32 flex-shrink-0 rounded-lg object-cover shadow-lg sm:h-40 sm:w-40"
							/>
						{:else}
							<div class="flex h-32 w-32 flex-shrink-0 items-center justify-center rounded-lg bg-zinc-800 sm:h-40 sm:w-40">
								<Disc3 class="h-12 w-12 text-zinc-600" />
							</div>
						{/if}

						<div class="flex min-w-0 flex-1 flex-col justify-between">
							<div>
								<div class="mb-1 flex items-center gap-2">
									<span class="rounded bg-zinc-800 px-2 py-0.5 text-xs font-medium text-zinc-400">ALBUM</span>
								</div>
								<h2 class="truncate text-xl font-bold text-white sm:text-2xl">{album.name}</h2>
								<p class="mt-1 truncate text-zinc-400">{album.artist}</p>
							</div>

							<div class="mt-4 flex flex-wrap items-center gap-2 text-xs text-zinc-500">
								<span class="rounded bg-zinc-800 px-2 py-1">{album.total_tracks} tracks</span>
								{#if album.release_date}
									<span class="rounded bg-zinc-800 px-2 py-1">{album.release_date.split('-')[0]}</span>
								{/if}
							</div>
						</div>
					</div>

					<!-- Download All Button -->
					<div class="border-t border-zinc-800 p-4">
						{#if batchDownloading}
							<div class="space-y-2">
								<div class="flex items-center justify-between text-sm">
									<span class="text-zinc-400">Downloading {batchProgress.current}/{batchProgress.total}</span>
									<span class="text-zinc-500">
										<span class="text-emerald-500">{batchProgress.success}</span> /
										<span class="text-red-500">{batchProgress.failed}</span>
									</span>
								</div>
								<div class="h-2 overflow-hidden rounded-full bg-zinc-800">
									<div
										class="h-full bg-emerald-500 transition-all"
										style="width: {(batchProgress.current / batchProgress.total) * 100}%"
									></div>
								</div>
							</div>
						{:else}
							<button
								onclick={handleDownloadAll}
								class="flex w-full items-center justify-center gap-2 rounded-lg bg-emerald-500 py-3 font-medium text-black transition-all hover:bg-emerald-400"
							>
								<Download class="h-5 w-5" />
								Download All ({album.total_tracks} tracks)
							</button>
						{/if}
					</div>

					<!-- Track List -->
					<div class="border-t border-zinc-800">
						{#each album.tracks as track, i}
							{@const status = getTrackStatus(track)}
							<div class="flex items-center gap-3 border-b border-zinc-800/50 px-4 py-3 last:border-b-0">
								<span class="w-6 text-center text-sm text-zinc-600">{i + 1}</span>

								<div class="min-w-0 flex-1">
									<p class="truncate text-sm text-white">{track.title}</p>
									<p class="truncate text-xs text-zinc-500">{track.artist}</p>
								</div>

								{#if track.duration_ms}
									<span class="text-xs text-zinc-600">{formatDuration(track.duration_ms)}</span>
								{/if}

								<div class="w-6">
									{#if status?.status === 'downloading'}
										<Loader2 class="h-4 w-4 animate-spin text-emerald-500" />
									{:else if status?.status === 'success'}
										<Check class="h-4 w-4 text-emerald-500" />
									{:else if status?.status === 'error'}
										<X class="h-4 w-4 text-red-500" />
									{/if}
								</div>
							</div>
						{/each}
					</div>
				</div>

			<!-- Playlist View -->
			{:else if metadata.type === 'playlist' && metadata.playlist}
				{@const playlist = metadata.playlist}
				<div class="overflow-hidden rounded-xl border border-zinc-800 bg-zinc-900">
					<div class="flex gap-4 p-4 sm:p-6">
						{#if playlist.cover_url}
							<img
								src={playlist.cover_url}
								alt={playlist.name}
								class="h-32 w-32 flex-shrink-0 rounded-lg object-cover shadow-lg sm:h-40 sm:w-40"
							/>
						{:else}
							<div class="flex h-32 w-32 flex-shrink-0 items-center justify-center rounded-lg bg-zinc-800 sm:h-40 sm:w-40">
								<ListMusic class="h-12 w-12 text-zinc-600" />
							</div>
						{/if}

						<div class="flex min-w-0 flex-1 flex-col justify-between">
							<div>
								<div class="mb-1 flex items-center gap-2">
									<span class="rounded bg-zinc-800 px-2 py-0.5 text-xs font-medium text-zinc-400">PLAYLIST</span>
								</div>
								<h2 class="truncate text-xl font-bold text-white sm:text-2xl">{playlist.name}</h2>
								{#if playlist.owner}
									<p class="mt-1 truncate text-zinc-400">by {playlist.owner}</p>
								{/if}
							</div>

							<div class="mt-4">
								<span class="rounded bg-zinc-800 px-2 py-1 text-xs text-zinc-500">{playlist.total_tracks} tracks</span>
							</div>
						</div>
					</div>

					<!-- Download All Button -->
					<div class="border-t border-zinc-800 p-4">
						{#if batchDownloading}
							<div class="space-y-2">
								<div class="flex items-center justify-between text-sm">
									<span class="text-zinc-400">Downloading {batchProgress.current}/{batchProgress.total}</span>
									<span class="text-zinc-500">
										<span class="text-emerald-500">{batchProgress.success}</span> /
										<span class="text-red-500">{batchProgress.failed}</span>
									</span>
								</div>
								<div class="h-2 overflow-hidden rounded-full bg-zinc-800">
									<div
										class="h-full bg-emerald-500 transition-all"
										style="width: {(batchProgress.current / batchProgress.total) * 100}%"
									></div>
								</div>
							</div>
						{:else}
							<button
								onclick={handleDownloadAll}
								class="flex w-full items-center justify-center gap-2 rounded-lg bg-emerald-500 py-3 font-medium text-black transition-all hover:bg-emerald-400"
							>
								<Download class="h-5 w-5" />
								Download All ({playlist.total_tracks} tracks)
							</button>
						{/if}
					</div>

					<!-- Track List -->
					<div class="max-h-96 overflow-y-auto border-t border-zinc-800">
						{#each playlist.tracks as track, i}
							{@const status = getTrackStatus(track)}
							<div class="flex items-center gap-3 border-b border-zinc-800/50 px-4 py-3 last:border-b-0">
								<span class="w-6 text-center text-sm text-zinc-600">{i + 1}</span>

								{#if track.cover_url}
									<img src={track.cover_url} alt="" class="h-10 w-10 rounded object-cover" />
								{:else}
									<div class="flex h-10 w-10 items-center justify-center rounded bg-zinc-800">
										<Music class="h-4 w-4 text-zinc-600" />
									</div>
								{/if}

								<div class="min-w-0 flex-1">
									<p class="truncate text-sm text-white">{track.title}</p>
									<p class="truncate text-xs text-zinc-500">{track.artist}</p>
								</div>

								{#if track.duration_ms}
									<span class="text-xs text-zinc-600">{formatDuration(track.duration_ms)}</span>
								{/if}

								<div class="w-6">
									{#if status?.status === 'downloading'}
										<Loader2 class="h-4 w-4 animate-spin text-emerald-500" />
									{:else if status?.status === 'success'}
										<Check class="h-4 w-4 text-emerald-500" />
									{:else if status?.status === 'error'}
										<X class="h-4 w-4 text-red-500" />
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
				class="mt-4 w-full rounded-lg border border-zinc-800 py-2 text-sm text-zinc-400 transition-colors hover:border-zinc-700 hover:text-zinc-300"
			>
				Start Over
			</button>

		{:else if !loading}
			<!-- Empty State -->
			<div class="py-16 text-center">
				<div class="mx-auto mb-6 flex h-24 w-24 items-center justify-center rounded-full bg-zinc-900">
					<Music class="h-12 w-12 text-zinc-700" />
				</div>
				<h2 class="text-xl font-semibold text-white">Paste a URL to get started</h2>
				<p class="mt-2 text-zinc-500">
					Supports Spotify and Deezer tracks, albums, and playlists
				</p>
				<div class="mt-6 flex flex-wrap justify-center gap-2">
					<span class="rounded-full bg-zinc-900 px-3 py-1 text-xs text-zinc-500">Tidal</span>
					<span class="rounded-full bg-zinc-900 px-3 py-1 text-xs text-zinc-500">Qobuz</span>
					<span class="rounded-full bg-zinc-900 px-3 py-1 text-xs text-zinc-500">Amazon Music</span>
				</div>
			</div>
		{/if}
	</main>
</div>
