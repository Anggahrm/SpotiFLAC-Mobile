<script lang="ts">
	import { Terminal, ArrowLeft } from 'lucide-svelte';
	import { apiEndpoints } from '$lib/data/api-docs';
	import ApiEndpoint from '$lib/components/docs/ApiEndpoint.svelte';
</script>

<div class="min-h-screen font-mono bg-[var(--background)] text-[var(--foreground)]">
	<!-- Header -->
	<header class="sticky top-0 z-30 border-b border-[var(--border)] bg-[var(--background)]/80 backdrop-blur-sm">
		<div class="mx-auto flex h-14 max-w-6xl items-center justify-between px-4 lg:px-8">
			<div class="flex items-center gap-4">
				<a href="/" class="flex items-center gap-2 text-[var(--muted-foreground)] hover:text-[var(--foreground)] transition-colors">
					<ArrowLeft class="h-4 w-4" />
					<span class="text-sm">Back</span>
				</a>
				<div class="h-4 w-px bg-[var(--border)]"></div>
				<div class="flex items-center gap-2">
					<Terminal class="h-5 w-5" />
					<span class="text-lg font-bold">zFlac API</span>
				</div>
			</div>
		</div>
	</header>

	<!-- Main Content -->
	<main class="mx-auto max-w-6xl px-4 lg:px-8 py-8">
		<!-- Hero -->
		<div class="mb-8">
			<h1 class="text-2xl font-bold mb-2">API Documentation</h1>
			<p class="text-[var(--muted-foreground)]">
				Complete reference for the zFlac API. Download high-quality FLAC music from Tidal, Qobuz, and Amazon.
			</p>
		</div>

		<!-- Quick Info -->
		<div class="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8">
			<div class="border border-[var(--border)] rounded-lg p-4 bg-[var(--card)]">
				<h3 class="text-xs font-medium text-[var(--muted-foreground)] mb-1">Base URL</h3>
				<code class="text-sm text-[var(--foreground)]">/</code>
			</div>
			<div class="border border-[var(--border)] rounded-lg p-4 bg-[var(--card)]">
				<h3 class="text-xs font-medium text-[var(--muted-foreground)] mb-1">Format</h3>
				<code class="text-sm text-[var(--foreground)]">JSON</code>
			</div>
			<div class="border border-[var(--border)] rounded-lg p-4 bg-[var(--card)]">
				<h3 class="text-xs font-medium text-[var(--muted-foreground)] mb-1">Endpoints</h3>
				<code class="text-sm text-[var(--foreground)]">{apiEndpoints.length} available</code>
			</div>
		</div>

		<!-- Endpoints -->
		<div class="space-y-4">
			<h2 class="text-lg font-bold">Endpoints</h2>
			{#each apiEndpoints as endpoint}
				<ApiEndpoint {endpoint} />
			{/each}
		</div>
	</main>

	<!-- Footer -->
	<footer class="border-t border-[var(--border)] mt-12">
		<div class="mx-auto max-w-6xl px-4 lg:px-8 py-6">
			<p class="text-xs text-[var(--muted-foreground)] text-center">
				zFlac API Documentation
			</p>
		</div>
	</footer>
</div>
