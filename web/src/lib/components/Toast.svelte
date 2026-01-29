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
		success: 'bg-white border-emerald-500 text-emerald-600',
		error: 'bg-white border-red-500 text-red-600',
		info: 'bg-white border-violet-500 text-violet-600',
		loading: 'bg-white border-amber-500 text-amber-600'
	};

	const Icon = $derived(icons[type]);
</script>

<div
	transition:fly={{ y: 20, duration: 200 }}
	class="flex items-center gap-3 rounded-lg border-2 p-3 shadow-[3px_3px_0px_0px_#c4b5fd] {colors[type]}"
>
	<Icon class="h-5 w-5 flex-shrink-0 {type === 'loading' ? 'animate-spin' : ''}" />
	<span class="flex-1 text-sm font-medium">{message}</span>
	{#if onclose}
		<button onclick={onclose} class="flex-shrink-0 opacity-60 hover:opacity-100 transition-opacity">
			<X class="h-4 w-4" />
		</button>
	{/if}
</div>
