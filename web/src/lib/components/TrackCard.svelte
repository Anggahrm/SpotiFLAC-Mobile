<script lang="ts">
	import type { TrackMetadata } from '$lib/api';
	import Card from '$lib/components/ui/Card.svelte';
	import { Music, Clock, Disc } from 'lucide-svelte';

	interface Props {
		track: TrackMetadata;
		onclick?: () => void;
	}

	let { track, onclick }: Props = $props();

	function formatDuration(ms: number): string {
		const minutes = Math.floor(ms / 60000);
		const seconds = Math.floor((ms % 60000) / 1000);
		return `${minutes}:${seconds.toString().padStart(2, '0')}`;
	}
</script>

<Card class="cursor-pointer overflow-hidden transition-all hover:border-[var(--primary)] hover:shadow-lg">
	<button class="flex w-full items-start gap-4 p-4 text-left" onclick={onclick}>
		{#if track.cover_url}
			<img
				src={track.cover_url}
				alt={track.title}
				class="h-16 w-16 flex-shrink-0 rounded-md object-cover"
			/>
		{:else}
			<div class="flex h-16 w-16 flex-shrink-0 items-center justify-center rounded-md bg-[var(--secondary)]">
				<Music class="h-8 w-8 text-[var(--muted-foreground)]" />
			</div>
		{/if}
		<div class="min-w-0 flex-1">
			<h3 class="truncate text-sm font-medium text-[var(--foreground)]">{track.title}</h3>
			<p class="truncate text-sm text-[var(--muted-foreground)]">{track.artist}</p>
			<div class="mt-2 flex items-center gap-3 text-xs text-[var(--muted-foreground)]">
				{#if track.album}
					<span class="flex items-center gap-1 truncate">
						<Disc class="h-3 w-3" />
						{track.album}
					</span>
				{/if}
				{#if track.duration_ms}
					<span class="flex items-center gap-1">
						<Clock class="h-3 w-3" />
						{formatDuration(track.duration_ms)}
					</span>
				{/if}
			</div>
		</div>
	</button>
</Card>
