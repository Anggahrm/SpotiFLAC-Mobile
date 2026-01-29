<script lang="ts">
	import type { TrackMetadata } from '$lib/api';
	import Card from '$lib/components/ui/Card.svelte';
	import { Music, Clock, Disc, User, Calendar } from 'lucide-svelte';

	interface Props {
		track: TrackMetadata;
	}

	let { track }: Props = $props();

	function formatDuration(ms: number): string {
		const minutes = Math.floor(ms / 60000);
		const seconds = Math.floor((ms % 60000) / 1000);
		return `${minutes}:${seconds.toString().padStart(2, '0')}`;
	}
</script>

<Card class="overflow-hidden">
	<div class="flex flex-col gap-4 p-4 sm:flex-row">
		{#if track.cover_url}
			<img
				src={track.cover_url}
				alt={track.title}
				class="h-40 w-40 flex-shrink-0 self-center rounded-lg object-cover shadow-lg sm:self-start"
			/>
		{:else}
			<div class="flex h-40 w-40 flex-shrink-0 items-center justify-center self-center rounded-lg bg-[var(--secondary)] sm:self-start">
				<Music class="h-16 w-16 text-[var(--muted-foreground)]" />
			</div>
		{/if}
		<div class="flex flex-1 flex-col justify-center">
			<h2 class="text-xl font-bold text-[var(--foreground)]">{track.title}</h2>
			<div class="mt-2 flex items-center gap-2 text-[var(--muted-foreground)]">
				<User class="h-4 w-4" />
				<span>{track.artist}</span>
			</div>
			{#if track.album}
				<div class="mt-1 flex items-center gap-2 text-sm text-[var(--muted-foreground)]">
					<Disc class="h-4 w-4" />
					<span>{track.album}</span>
				</div>
			{/if}
			<div class="mt-3 flex flex-wrap gap-4 text-sm text-[var(--muted-foreground)]">
				{#if track.duration_ms}
					<span class="flex items-center gap-1">
						<Clock class="h-4 w-4" />
						{formatDuration(track.duration_ms)}
					</span>
				{/if}
				{#if track.release_date}
					<span class="flex items-center gap-1">
						<Calendar class="h-4 w-4" />
						{track.release_date}
					</span>
				{/if}
			</div>
		</div>
	</div>
</Card>
