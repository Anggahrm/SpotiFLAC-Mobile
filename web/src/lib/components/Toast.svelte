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
		success: 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400',
		error: 'bg-red-500/10 border-red-500/30 text-red-400',
		info: 'bg-blue-500/10 border-blue-500/30 text-blue-400',
		loading: 'bg-amber-500/10 border-amber-500/30 text-amber-400'
	};

	const Icon = $derived(icons[type]);
</script>

<div
	transition:fly={{ y: 20, duration: 200 }}
	class="flex items-center gap-3 rounded-xl border p-4 shadow-2xl backdrop-blur-sm {colors[type]}"
>
	<Icon class="h-5 w-5 flex-shrink-0 {type === 'loading' ? 'animate-spin' : ''}" />
	<span class="flex-1 text-sm">{message}</span>
	{#if onclose}
		<button onclick={onclose} class="flex-shrink-0 opacity-60 hover:opacity-100 transition-opacity">
			<X class="h-4 w-4" />
		</button>
	{/if}
</div>
