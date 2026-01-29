<script lang="ts">
	import { cn } from '$lib/utils';

	interface Props {
		value: number;
		max?: number;
		class?: string;
		showLabel?: boolean;
	}

	let { value, max = 100, class: className = '', showLabel = false }: Props = $props();

	const percentage = $derived(Math.min(100, Math.max(0, (value / max) * 100)));
</script>

<div class={cn('relative', className)}>
	<div class="h-2 w-full overflow-hidden rounded-full bg-[var(--secondary)]">
		<div
			class="h-full bg-[var(--primary)] transition-all duration-300 ease-out"
			style="width: {percentage}%"
		></div>
	</div>
	{#if showLabel}
		<span class="mt-1 block text-xs text-[var(--muted-foreground)]">{Math.round(percentage)}%</span>
	{/if}
</div>
