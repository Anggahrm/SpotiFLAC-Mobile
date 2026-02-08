<script lang="ts">
	interface Props {
		current: number;
		total: number;
		success: number;
		failed: number;
	}

	let { current, total, success, failed }: Props = $props();
</script>

<div class="space-y-2">
	<div class="flex items-center justify-between text-[10px] sm:text-xs font-mono">
		<span class="text-[var(--muted-foreground)]">{current}/{total}</span>
		<span>
			<span class="text-emerald-500">{success}</span> /
			<span class="text-red-500">{failed}</span>
		</span>
	</div>
	<div class="h-1.5 sm:h-2 bg-[var(--secondary)] rounded-full overflow-hidden border border-[var(--border)]">
		<div
			class="h-full bg-[var(--foreground)] transition-all"
			style="width: {(current / total) * 100}%"
		></div>
	</div>
</div>
