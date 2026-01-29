<script lang="ts">
	import { cn } from '$lib/utils';

	interface Props {
		class?: string;
	}

	let { class: className = '', children }: Props & { children?: any } = $props();
</script>

<div class={cn('rounded-lg border border-[var(--border)] bg-[var(--card)] text-[var(--card-foreground)] shadow-sm', className)}>
	{@render children?.()}
</div>
