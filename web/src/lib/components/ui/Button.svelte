<script lang="ts">
	import { cn } from '$lib/utils';

	interface Props {
		variant?: 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link';
		size?: 'default' | 'sm' | 'lg' | 'icon';
		class?: string;
		disabled?: boolean;
		type?: 'button' | 'submit' | 'reset';
		onclick?: () => void;
	}

	let {
		variant = 'default',
		size = 'default',
		class: className = '',
		disabled = false,
		type = 'button',
		onclick,
		children
	}: Props & { children?: any } = $props();

	const variants = {
		default: 'bg-[var(--primary)] text-[var(--primary-foreground)] hover:bg-[var(--primary)]/90',
		destructive: 'bg-[var(--destructive)] text-[var(--destructive-foreground)] hover:bg-[var(--destructive)]/90',
		outline: 'border border-[var(--border)] bg-transparent hover:bg-[var(--secondary)] hover:text-[var(--foreground)]',
		secondary: 'bg-[var(--secondary)] text-[var(--secondary-foreground)] hover:bg-[var(--secondary)]/80',
		ghost: 'hover:bg-[var(--secondary)] hover:text-[var(--foreground)]',
		link: 'text-[var(--primary)] underline-offset-4 hover:underline'
	};

	const sizes = {
		default: 'h-10 px-4 py-2',
		sm: 'h-9 rounded-md px-3',
		lg: 'h-11 rounded-md px-8',
		icon: 'h-10 w-10'
	};
</script>

<button
	{type}
	{disabled}
	{onclick}
	class={cn(
		'inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--ring)] focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50',
		variants[variant],
		sizes[size],
		className
	)}
>
	{@render children?.()}
</button>
