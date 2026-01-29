<script lang="ts">
	import { cn } from '$lib/utils';

	interface Props {
		class?: string;
		type?: 'text' | 'password' | 'email' | 'url' | 'search';
		placeholder?: string;
		value?: string;
		disabled?: boolean;
		oninput?: (e: Event) => void;
		onpaste?: (e: ClipboardEvent) => void;
		onkeydown?: (e: KeyboardEvent) => void;
	}

	let {
		class: className = '',
		type = 'text',
		placeholder = '',
		value = $bindable(''),
		disabled = false,
		oninput,
		onpaste,
		onkeydown
	}: Props = $props();
</script>

<input
	{type}
	{placeholder}
	{disabled}
	bind:value
	{oninput}
	{onpaste}
	{onkeydown}
	class={cn(
		'flex h-10 w-full rounded-md border border-[var(--border)] bg-[var(--background)] px-3 py-2 text-sm ring-offset-[var(--background)] file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-[var(--muted-foreground)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--ring)] focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50',
		className
	)}
/>
