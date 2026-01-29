<script lang="ts">
	import { getMetadata, downloadTrack, getFileUrl, type TrackMetadata } from '$lib/api';
	import { downloads } from '$lib/stores/downloads';
	import { history } from '$lib/stores/history';
	import { toasts } from '$lib/stores/toasts';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Progress from '$lib/components/ui/Progress.svelte';
	import TrackPreview from '$lib/components/TrackPreview.svelte';
	import QualitySelector from '$lib/components/QualitySelector.svelte';
	import { Download, Link, Loader2, CheckCircle, Music } from 'lucide-svelte';

	let url = $state('');
	let track: TrackMetadata | null = $state(null);
	let quality: 'LOSSLESS' | 'HIGH' | 'NORMAL' = $state('LOSSLESS');
	let loading = $state(false);
	let downloading = $state(false);
	let downloadComplete = $state(false);
	let downloadProgress = $state(0);
	let downloadedFileName = $state('');

	async function handlePaste(e: ClipboardEvent) {
		const text = e.clipboardData?.getData('text');
		if (text && (text.includes('spotify.com') || text.includes('deezer.com'))) {
			url = text;
			await fetchMetadata();
		}
	}

	async function fetchMetadata() {
		if (!url.trim()) return;

		if (!url.includes('spotify.com') && !url.includes('deezer.com')) {
			toasts.error('Please enter a valid Spotify or Deezer URL');
			return;
		}

		loading = true;
		track = null;
		downloadComplete = false;

		try {
			track = await getMetadata(url);
			toasts.success('Track found!');
		} catch (err) {
			toasts.error(err instanceof Error ? err.message : 'Failed to fetch metadata');
		} finally {
			loading = false;
		}
	}

	async function handleDownload() {
		if (!track) return;

		downloading = true;
		downloadProgress = 0;
		downloadComplete = false;

		const itemId = crypto.randomUUID();

		try {
			downloads.add({
				id: itemId,
				title: track.title,
				artist: track.artist,
				cover_url: track.cover_url,
				quality: quality === 'LOSSLESS' ? 'FLAC' : 'M4A'
			});

			const result = await downloadTrack({
				track_name: track.title,
				artist_name: track.artist,
				album_name: track.album,
				album_artist: track.album_artist,
				cover_url: track.cover_url,
				spotify_id: track.spotify_id,
				isrc: track.isrc,
				quality,
				item_id: itemId,
				duration_ms: track.duration_ms
			});

			if (result.success && result.file_name) {
				downloadedFileName = result.file_name;
				downloadComplete = true;
				downloadProgress = 100;

				downloads.updateItem(itemId, {
					status: 'completed',
					progress: 100,
					file_name: result.file_name
				});

				history.add({
					title: track.title,
					artist: track.artist,
					album: track.album,
					cover_url: track.cover_url,
					quality: quality === 'LOSSLESS' ? 'FLAC' : 'M4A',
					file_name: result.file_name
				});

				toasts.success('Download complete!');
			} else {
				throw new Error(result.error || 'Download failed');
			}
		} catch (err) {
			toasts.error(err instanceof Error ? err.message : 'Download failed');
			downloads.updateItem(itemId, { status: 'error' });
		} finally {
			downloading = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			fetchMetadata();
		}
	}
</script>

<div class="mx-auto max-w-2xl px-4 py-8">
	<!-- Header -->
	<div class="mb-8 text-center">
		<div class="mb-4 flex items-center justify-center gap-3">
			<div class="flex h-12 w-12 items-center justify-center rounded-xl bg-[var(--primary)]">
				<Music class="h-6 w-6 text-black" />
			</div>
			<h1 class="text-3xl font-bold text-[var(--foreground)]">zFlac Downloader</h1>
		</div>
		<p class="text-[var(--muted-foreground)]">
			Download high-quality FLAC music from Spotify & Deezer
		</p>
	</div>

	<!-- URL Input -->
	<div class="mb-6">
		<div class="relative">
			<Link class="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-[var(--muted-foreground)]" />
			<Input
				type="url"
				placeholder="Paste Spotify or Deezer URL here..."
				class="h-14 pl-10 pr-24 text-lg"
				bind:value={url}
				onpaste={handlePaste}
				onkeydown={handleKeydown}
			/>
			<Button
				class="absolute right-2 top-1/2 -translate-y-1/2"
				onclick={fetchMetadata}
				disabled={loading || !url.trim()}
			>
				{#if loading}
					<Loader2 class="h-4 w-4 animate-spin" />
				{:else}
					Fetch
				{/if}
			</Button>
		</div>
	</div>

	<!-- Track Preview -->
	{#if track}
		<div class="mb-6">
			<TrackPreview {track} />
		</div>

		<!-- Quality Selector -->
		<div class="mb-6">
			<h3 class="mb-3 text-sm font-medium text-[var(--foreground)]">Quality</h3>
			<QualitySelector bind:value={quality} />
		</div>

		<!-- Download Button -->
		<div class="mb-6">
			{#if downloading}
				<div class="space-y-3">
					<Progress value={downloadProgress} showLabel />
					<p class="text-center text-sm text-[var(--muted-foreground)]">
						Downloading...
					</p>
				</div>
			{:else if downloadComplete}
				<div class="space-y-3">
					<div class="flex items-center justify-center gap-2 text-green-400">
						<CheckCircle class="h-5 w-5" />
						<span>Download Complete!</span>
					</div>
					<a
						href={getFileUrl(downloadedFileName)}
						download
						class="flex w-full items-center justify-center gap-2 rounded-lg bg-[var(--primary)] px-4 py-3 font-medium text-black transition-opacity hover:opacity-90"
					>
						<Download class="h-5 w-5" />
						Save File
					</a>
				</div>
			{:else}
				<Button class="w-full py-6 text-lg" onclick={handleDownload}>
					<Download class="mr-2 h-5 w-5" />
					Download {quality === 'LOSSLESS' ? 'FLAC' : 'M4A'}
				</Button>
			{/if}
		</div>
	{/if}

	<!-- Empty State -->
	{#if !track && !loading}
		<div class="mt-12 text-center">
			<div class="mx-auto mb-4 flex h-20 w-20 items-center justify-center rounded-full bg-[var(--secondary)]">
				<Link class="h-10 w-10 text-[var(--muted-foreground)]" />
			</div>
			<h3 class="text-lg font-medium text-[var(--foreground)]">Paste a link to get started</h3>
			<p class="mt-2 text-sm text-[var(--muted-foreground)]">
				Supports Spotify and Deezer track, album, and playlist links
			</p>
		</div>
	{/if}
</div>
