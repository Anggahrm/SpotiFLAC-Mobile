<script lang="ts">
	import { X, CheckCircle, AlertCircle, Info, Loader2 } from 'lucide-svelte';
	import { fly } from 'svelte/transition';

	type ToastType = 'success' | 'error' | 'info' | 'loading';

	interface Props {
		message: string;
		type?: ToastType;
		onclose?: () => void;
	}

	let { message, type = 'info', onclose }: Props = $props();

	const icons = {
		success: CheckCircle,
		error: AlertCircle,
		info: Info,
		loading: Loader2
	};

	const colors = {
		success: 'bg-[var(--card)] border-emerald-500 text-emerald-500',
		error: 'bg-[var(--card)] border-red-500 text-red-500',
		info: 'bg-[var(--card)] border-[var(--foreground)] text-[var(--foreground)]',
		loading: 'bg-[var(--card)] border-amber-500 text-amber-500'
	};

	const Icon = $derived(icons[type]);
</script>

<div
	transition:fly={{ y: -20, duration: 200 }}
	class="flex items-center gap-3 rounded-lg border p-3 shadow-lg {colors[type]}"
>
	<Icon class="h-5 w-5 flex-shrink-0 {type === 'loading' ? 'animate-spin' : ''}" />
	<span class="flex-1 text-sm font-medium text-[var(--foreground)]">{message}</span>
	{#if onclose}
		<button onclick={onclose} class="flex-shrink-0 opacity-60 hover:opacity-100 transition-opacity text-[var(--foreground)]">
			<X class="h-4 w-4" />
		</button>
	{/if}
</div>
