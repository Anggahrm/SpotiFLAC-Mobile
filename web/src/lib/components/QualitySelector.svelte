<script lang="ts">
	import { cn } from '$lib/utils';

	type Quality = 'LOSSLESS' | 'HIGH' | 'NORMAL';

	interface Props {
		value: Quality;
		onchange?: (quality: Quality) => void;
	}

	let { value = $bindable('LOSSLESS'), onchange }: Props = $props();

	const qualities: { value: Quality; label: string; description: string }[] = [
		{ value: 'LOSSLESS', label: 'FLAC', description: 'Lossless ~1411 kbps' },
		{ value: 'HIGH', label: 'M4A', description: 'AAC 320 kbps' },
		{ value: 'NORMAL', label: 'M4A', description: 'AAC 128 kbps' }
	];

	function select(quality: Quality) {
		value = quality;
		onchange?.(quality);
	}
</script>

<div class="flex gap-2">
	{#each qualities as q}
		<button
			type="button"
			onclick={() => select(q.value)}
			class={cn(
				'flex-1 rounded-lg border px-4 py-3 text-center transition-all',
				value === q.value
					? 'border-[var(--primary)] bg-[var(--primary)]/10 text-[var(--primary)]'
					: 'border-[var(--border)] text-[var(--muted-foreground)] hover:border-[var(--primary)]/50'
			)}
		>
			<div class="text-sm font-medium">{q.label}</div>
			<div class="mt-1 text-xs opacity-75">{q.description}</div>
		</button>
	{/each}
</div>
