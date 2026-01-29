<script lang="ts">
	import { search, downloadTrack, type TrackMetadata, type SearchResult } from '$lib/api';
	import { downloads } from '$lib/stores/downloads';
	import { history } from '$lib/stores/history';
	import { toasts } from '$lib/stores/toasts';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import TrackCard from '$lib/components/TrackCard.svelte';
	import QualitySelector from '$lib/components/QualitySelector.svelte';
	import { Search, Loader2, X, Download, Music } from 'lucide-svelte';

	let query = $state('');
	let source: 'deezer' | 'spotify' = $state('deezer');
	let results: SearchResult | null = $state(null);
	let loading = $state(false);
	let selectedTrack: TrackMetadata | null = $state(null);
	let quality: 'LOSSLESS' | 'HIGH' | 'NORMAL' = $state('LOSSLESS');
	let downloading = $state(false);

	let debounceTimer: ReturnType<typeof setTimeout>;

	function handleInput() {
		clearTimeout(debounceTimer);
		if (query.trim().length >= 2) {
			debounceTimer = setTimeout(handleSearch, 300);
		}
	}

	async function handleSearch() {
		if (!query.trim()) return;

		loading = true;
		results = null;

		try {
			results = await search(query, source, 20);
		} catch (err) {
			toasts.error(err instanceof Error ? err.message : 'Search failed');
		} finally {
			loading = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			clearTimeout(debounceTimer);
			handleSearch();
		}
	}

	async function handleDownload() {
		if (!selectedTrack) return;

		downloading = true;
		const itemId = crypto.randomUUID();

		try {
			downloads.add({
				id: itemId,
				title: selectedTrack.title,
				artist: selectedTrack.artist,
				cover_url: selectedTrack.cover_url,
				quality: quality === 'LOSSLESS' ? 'FLAC' : 'M4A'
			});

			const result = await downloadTrack({
				track_name: selectedTrack.title,
				artist_name: selectedTrack.artist,
				album_name: selectedTrack.album,
				album_artist: selectedTrack.album_artist,
				cover_url: selectedTrack.cover_url,
				spotify_id: selectedTrack.spotify_id,
				isrc: selectedTrack.isrc,
				quality,
				item_id: itemId,
				duration_ms: selectedTrack.duration_ms
			});

			if (result.success && result.file_name) {
				downloads.updateItem(itemId, {
					status: 'completed',
					progress: 100,
					file_name: result.file_name
				});

				history.add({
					title: selectedTrack.title,
					artist: selectedTrack.artist,
					album: selectedTrack.album,
					cover_url: selectedTrack.cover_url,
					quality: quality === 'LOSSLESS' ? 'FLAC' : 'M4A',
					file_name: result.file_name
				});

				toasts.success('Download complete!');
				selectedTrack = null;
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
</script>

<div class="mx-auto max-w-4xl px-4 py-8">
	<!-- Search Input -->
	<div class="mb-6">
		<div class="relative">
			<Search class="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-[var(--muted-foreground)]" />
			<Input
				type="search"
				placeholder="Search for songs, artists, or albums..."
				class="h-14 pl-10 pr-4 text-lg"
				bind:value={query}
				oninput={handleInput}
				onkeydown={handleKeydown}
			/>
		</div>
	</div>

	<!-- Source Toggle -->
	<div class="mb-6 flex gap-2">
		<Button
			variant={source === 'deezer' ? 'default' : 'outline'}
			onclick={() => { source = 'deezer'; if (query) handleSearch(); }}
		>
			Deezer
		</Button>
		<Button
			variant={source === 'spotify' ? 'default' : 'outline'}
			onclick={() => { source = 'spotify'; if (query) handleSearch(); }}
		>
			Spotify
		</Button>
	</div>

	<!-- Loading -->
	{#if loading}
		<div class="flex items-center justify-center py-12">
			<Loader2 class="h-8 w-8 animate-spin text-[var(--primary)]" />
		</div>
	{/if}

	<!-- Results -->
	{#if results && !loading}
		{#if results.tracks && results.tracks.length > 0}
			<div class="grid gap-4 sm:grid-cols-2">
				{#each results.tracks as track}
					<TrackCard {track} onclick={() => (selectedTrack = track)} />
				{/each}
			</div>
		{:else}
			<div class="py-12 text-center">
				<p class="text-[var(--muted-foreground)]">No results found for "{query}"</p>
			</div>
		{/if}
	{/if}

	<!-- Empty State -->
	{#if !results && !loading}
		<div class="mt-12 text-center">
			<div class="mx-auto mb-4 flex h-20 w-20 items-center justify-center rounded-full bg-[var(--secondary)]">
				<Search class="h-10 w-10 text-[var(--muted-foreground)]" />
			</div>
			<h3 class="text-lg font-medium text-[var(--foreground)]">Search for music</h3>
			<p class="mt-2 text-sm text-[var(--muted-foreground)]">
				Find songs by title, artist, or album name
			</p>
		</div>
	{/if}
</div>

<!-- Download Modal -->
{#if selectedTrack}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" role="dialog">
		<Card class="w-full max-w-md">
			<div class="p-6">
				<div class="mb-4 flex items-start justify-between">
					<h2 class="text-lg font-semibold">Download Track</h2>
					<button onclick={() => (selectedTrack = null)} class="text-[var(--muted-foreground)] hover:text-[var(--foreground)]">
						<X class="h-5 w-5" />
					</button>
				</div>

				<div class="mb-6 flex items-center gap-4">
					{#if selectedTrack.cover_url}
						<img src={selectedTrack.cover_url} alt={selectedTrack.title} class="h-16 w-16 rounded-lg object-cover" />
					{:else}
						<div class="flex h-16 w-16 items-center justify-center rounded-lg bg-[var(--secondary)]">
							<Music class="h-8 w-8 text-[var(--muted-foreground)]" />
						</div>
					{/if}
					<div>
						<h3 class="font-medium">{selectedTrack.title}</h3>
						<p class="text-sm text-[var(--muted-foreground)]">{selectedTrack.artist}</p>
					</div>
				</div>

				<div class="mb-6">
					<h4 class="mb-2 text-sm font-medium">Quality</h4>
					<QualitySelector bind:value={quality} />
				</div>

				<Button class="w-full" onclick={handleDownload} disabled={downloading}>
					{#if downloading}
						<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						Downloading...
					{:else}
						<Download class="mr-2 h-4 w-4" />
						Download {quality === 'LOSSLESS' ? 'FLAC' : 'M4A'}
					{/if}
				</Button>
			</div>
		</Card>
	</div>
{/if}
