<script lang="ts">
	import { Terminal, ArrowLeft, Sparkles } from 'lucide-svelte';
	import { apiEndpoints } from '$lib/data/api-docs';
	import ApiEndpoint from '$lib/components/docs/ApiEndpoint.svelte';
</script>

<div class="relative min-h-screen bg-[var(--background)] text-[var(--foreground)]">
	<header class="sticky top-0 z-30 border-b border-[var(--border)] bg-[color:color-mix(in_srgb,var(--background)_78%,transparent)] backdrop-blur-xl">
		<div class="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 lg:px-8">
			<div class="flex items-center gap-4">
				<a href="/" class="inline-flex items-center gap-2 rounded-full border border-[var(--border)] px-3 py-1.5 text-xs text-[var(--muted-foreground)] transition-all hover:border-[var(--border-glow)] hover:text-[var(--foreground)]">
					<ArrowLeft class="h-4 w-4" />
					Back
				</a>
				<div class="h-4 w-px bg-[var(--border)]"></div>
				<div class="flex items-center gap-2">
					<Terminal class="h-5 w-5 text-[var(--primary)]" />
					<span class="font-display text-lg">zFlac API</span>
				</div>
			</div>
		</div>
	</header>

	<main class="relative z-10 mx-auto max-w-6xl px-4 py-8 lg:px-8">
		<section class="mb-8 rounded-2xl glass-card p-4 sm:p-6 fade-up">
			<p class="mb-2 inline-flex items-center gap-1 rounded-full border border-[var(--border)] px-2.5 py-1 font-mono text-[10px] uppercase tracking-[0.18em] text-[var(--muted-foreground)]">
				<Sparkles class="h-3 w-3" /> Developer Reference
			</p>
			<h1 class="font-display text-2xl sm:text-3xl leading-tight">API docs built for <span class="text-gradient">speed</span>.</h1>
			<p class="mt-2 max-w-3xl text-sm text-[var(--muted-foreground)]">Complete reference for the zFlac API to fetch metadata, check availability, and download FLAC-ready assets.</p>
		</section>

		<div class="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8">
			<div class="glass-card rounded-lg p-4">
				<h3 class="font-mono text-[10px] uppercase tracking-[0.12em] text-[var(--muted-foreground)] mb-1">Base URL</h3>
				<code class="text-sm text-[var(--foreground)]">/</code>
			</div>
			<div class="glass-card rounded-lg p-4">
				<h3 class="font-mono text-[10px] uppercase tracking-[0.12em] text-[var(--muted-foreground)] mb-1">Format</h3>
				<code class="text-sm text-[var(--foreground)]">JSON</code>
			</div>
			<div class="glass-card rounded-lg p-4">
				<h3 class="font-mono text-[10px] uppercase tracking-[0.12em] text-[var(--muted-foreground)] mb-1">Endpoints</h3>
				<code class="text-sm text-[var(--foreground)]">{apiEndpoints.length} available</code>
			</div>
		</div>

		<div class="space-y-4">
			<h2 class="font-display text-xl">Endpoints</h2>
			{#each apiEndpoints as endpoint}
				<ApiEndpoint {endpoint} />
			{/each}
		</div>
	</main>

	<footer class="relative z-10 border-t border-[var(--border)] mt-12 bg-[var(--card)]/80 backdrop-blur">
		<div class="mx-auto max-w-6xl px-4 lg:px-8 py-6">
			<p class="text-xs text-[var(--muted-foreground)] text-center">
				zFlac API Documentation
			</p>
		</div>
	</footer>
</div>
