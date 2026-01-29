<script lang="ts">
	import { history, type HistoryItem } from '$lib/stores/history';
	import { getFileUrl } from '$lib/api';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import { Download, Trash2, Music, Clock } from 'lucide-svelte';

	function formatDate(timestamp: number): string {
		const date = new Date(timestamp);
		const now = new Date();
		const diff = now.getTime() - date.getTime();
		const days = Math.floor(diff / (1000 * 60 * 60 * 24));

		if (days === 0) {
			return 'Today';
		} else if (days === 1) {
			return 'Yesterday';
		} else if (days < 7) {
			return `${days} days ago`;
		} else {
			return date.toLocaleDateString();
		}
	}

	function handleClear() {
		if (confirm('Are you sure you want to clear all history?')) {
			history.clear();
		}
	}
</script>

<div class="mx-auto max-w-4xl px-4 py-8">
	<!-- Header -->
	<div class="mb-6 flex items-center justify-between">
		<h1 class="text-2xl font-bold text-[var(--foreground)]">Download History</h1>
		{#if $history.length > 0}
			<Button variant="outline" size="sm" onclick={handleClear}>
				<Trash2 class="mr-2 h-4 w-4" />
				Clear All
			</Button>
		{/if}
	</div>

	<!-- History List -->
	{#if $history.length > 0}
		<div class="space-y-4">
			{#each $history as item (item.id)}
				<Card class="overflow-hidden">
					<div class="flex items-center gap-4 p-4">
						{#if item.cover_url}
							<img
								src={item.cover_url}
								alt={item.title}
								class="h-14 w-14 flex-shrink-0 rounded-lg object-cover"
							/>
						{:else}
							<div class="flex h-14 w-14 flex-shrink-0 items-center justify-center rounded-lg bg-[var(--secondary)]">
								<Music class="h-7 w-7 text-[var(--muted-foreground)]" />
							</div>
						{/if}

						<div class="min-w-0 flex-1">
							<h3 class="truncate font-medium text-[var(--foreground)]">{item.title}</h3>
							<p class="truncate text-sm text-[var(--muted-foreground)]">{item.artist}</p>
							<div class="mt-1 flex items-center gap-3 text-xs text-[var(--muted-foreground)]">
								<span class="rounded bg-[var(--secondary)] px-2 py-0.5">{item.quality}</span>
								<span class="flex items-center gap-1">
									<Clock class="h-3 w-3" />
									{formatDate(item.downloaded_at)}
								</span>
							</div>
						</div>

						<div class="flex gap-2">
							<a
								href={getFileUrl(item.file_name)}
								download
								class="flex h-10 w-10 items-center justify-center rounded-lg bg-[var(--primary)] text-black transition-opacity hover:opacity-90"
							>
								<Download class="h-5 w-5" />
							</a>
							<button
								onclick={() => history.remove(item.id)}
								class="flex h-10 w-10 items-center justify-center rounded-lg bg-[var(--secondary)] text-[var(--muted-foreground)] transition-colors hover:bg-[var(--destructive)] hover:text-white"
							>
								<Trash2 class="h-5 w-5" />
							</button>
						</div>
					</div>
				</Card>
			{/each}
		</div>
	{:else}
		<!-- Empty State -->
		<div class="mt-12 text-center">
			<div class="mx-auto mb-4 flex h-20 w-20 items-center justify-center rounded-full bg-[var(--secondary)]">
				<Clock class="h-10 w-10 text-[var(--muted-foreground)]" />
			</div>
			<h3 class="text-lg font-medium text-[var(--foreground)]">No download history</h3>
			<p class="mt-2 text-sm text-[var(--muted-foreground)]">
				Your downloaded tracks will appear here
			</p>
		</div>
	{/if}
</div>
