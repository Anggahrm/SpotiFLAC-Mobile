<script lang="ts">
	import { page } from '$app/stores';
	import { cn } from '$lib/utils';
	import { Home, Search, History } from 'lucide-svelte';

	const navItems = [
		{ href: '/', label: 'Home', icon: Home },
		{ href: '/search', label: 'Search', icon: Search },
		{ href: '/history', label: 'History', icon: History }
	];
</script>

<nav class="fixed bottom-0 left-0 right-0 z-50 border-t border-[var(--border)] bg-[var(--card)] md:relative md:border-b md:border-t-0">
	<div class="mx-auto flex max-w-screen-lg items-center justify-around px-4 py-2 md:justify-start md:gap-8">
		<a href="/" class="hidden text-xl font-bold text-[var(--primary)] md:block">
			zFlac Downloader
		</a>
		{#each navItems as item}
			<a
				href={item.href}
				class={cn(
					'flex flex-col items-center gap-1 px-4 py-2 text-xs transition-colors md:flex-row md:gap-2 md:text-sm',
					$page.url.pathname === item.href
						? 'text-[var(--primary)]'
						: 'text-[var(--muted-foreground)] hover:text-[var(--foreground)]'
				)}
			>
				<svelte:component this={item.icon} class="h-5 w-5" />
				<span>{item.label}</span>
			</a>
		{/each}
	</div>
</nav>
