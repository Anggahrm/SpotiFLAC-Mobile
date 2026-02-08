<script lang="ts">
	import { Terminal, ArrowLeft, Sparkles, Code2, BookOpen, Zap } from 'lucide-svelte';
	import { apiEndpoints } from '$lib/data/api-docs';
	import ApiEndpoint from '$lib/components/docs/ApiEndpoint.svelte';
</script>

<div class="relative min-h-screen bg-[var(--background)] text-[var(--foreground)]">
	<!-- Elegant Header -->
	<header class="sticky top-0 z-30 border-b border-[var(--border)] bg-[color:color-mix(in_srgb,var(--background)_85%,transparent)] backdrop-blur-2xl">
		<div class="mx-auto flex h-20 max-w-[min(1400px,94vw)] items-center justify-between px-6 lg:px-12">
			<div class="flex items-center gap-6">
				<a
					href="/"
					class="inline-flex items-center gap-2 text-[11px] font-medium tracking-wide text-[var(--muted-foreground)] hover:text-[var(--foreground)] transition-colors"
				>
					<ArrowLeft class="h-4 w-4" />
					Back
				</a>
				<div class="hidden sm:block h-4 w-px bg-[var(--border)]"></div>
				<div class="flex items-center gap-3">
					<Terminal class="h-5 w-5 text-[var(--primary)]" />
					<span class="font-display text-xl tracking-tight">API Reference</span>
				</div>
			</div>
			<a href="/" class="font-display text-lg text-gradient">zFlac</a>
		</div>
	</header>

	<main class="relative z-10 mx-auto max-w-[min(1400px,94vw)] px-6 lg:px-12 py-12 lg:py-16">
		<!-- Hero Section -->
		<section class="mb-12 lg:mb-16">
			<div class="max-w-2xl">
				<p class="mb-4 font-mono text-[11px] uppercase tracking-[0.25em] text-[var(--primary)]">
					Developer Documentation
				</p>
				<h1 class="font-display text-4xl sm:text-5xl leading-[1.1] tracking-tight">
					API built for <span class="text-gradient italic">precision</span>
				</h1>
				<p class="mt-6 text-base sm:text-lg text-[var(--muted-foreground)] leading-relaxed">
					Complete reference for integrating with the zFlac API. Fetch metadata, check availability, and download high-fidelity audio programmatically.
				</p>
			</div>
		</section>

		<!-- Quick Stats -->
		<section class="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-12">
			<div class="glass-card rounded-2xl p-6 text-center luxury-hover">
				<div class="w-10 h-10 mx-auto mb-4 rounded-xl bg-[var(--secondary)] flex items-center justify-center">
					<Zap class="w-5 h-5 text-[var(--primary)]" />
				</div>
				<p class="font-mono text-[10px] uppercase tracking-[0.2em] text-[var(--muted-foreground)] mb-1">Base URL</p>
				<code class="text-sm font-medium">/</code>
			</div>
			<div class="glass-card rounded-2xl p-6 text-center luxury-hover">
				<div class="w-10 h-10 mx-auto mb-4 rounded-xl bg-[var(--secondary)] flex items-center justify-center">
					<Code2 class="w-5 h-5 text-[var(--primary)]" />
				</div>
				<p class="font-mono text-[10px] uppercase tracking-[0.2em] text-[var(--muted-foreground)] mb-1">Format</p>
				<code class="text-sm font-medium">JSON</code>
			</div>
			<div class="glass-card rounded-2xl p-6 text-center luxury-hover">
				<div class="w-10 h-10 mx-auto mb-4 rounded-xl bg-[var(--secondary)] flex items-center justify-center">
					<BookOpen class="w-5 h-5 text-[var(--primary)]" />
				</div>
				<p class="font-mono text-[10px] uppercase tracking-[0.2em] text-[var(--muted-foreground)] mb-1">Endpoints</p>
				<code class="text-sm font-medium">{apiEndpoints.length} available</code>
			</div>
		</section>

		<!-- Authentication Note -->
		<section class="glass-card rounded-2xl p-6 lg:p-8 mb-12">
			<div class="flex items-start gap-4">
				<div class="w-10 h-10 rounded-xl bg-[var(--primary)]/10 flex items-center justify-center flex-shrink-0">
					<Sparkles class="w-5 h-5 text-[var(--primary)]" />
				</div>
				<div>
					<h3 class="font-display text-lg mb-2">Authentication</h3>
					<p class="text-sm text-[var(--muted-foreground)] leading-relaxed">
						The zFlac API is currently open and does not require authentication. All endpoints are accessible without API keys.
						Please be mindful of rate limits and fair usage.
					</p>
				</div>
			</div>
		</section>

		<!-- Endpoints -->
		<section>
			<div class="flex items-center gap-4 mb-8">
				<div class="h-px flex-1 bg-gradient-to-r from-[var(--border)] to-transparent"></div>
				<h2 class="font-display text-2xl">Endpoints</h2>
				<div class="h-px flex-1 bg-gradient-to-l from-[var(--border)] to-transparent"></div>
			</div>

			<div class="space-y-4">
				{#each apiEndpoints as endpoint}
					<ApiEndpoint {endpoint} />
				{/each}
			</div>
		</section>
	</main>

	<!-- Elegant Footer -->
	<footer class="relative z-10 border-t border-[var(--border)]">
		<div class="mx-auto max-w-[min(1400px,94vw)] px-6 lg:px-12 py-8">
			<div class="flex flex-col sm:flex-row justify-between items-center gap-4">
				<div class="flex items-center gap-3">
					<p class="font-display text-lg text-gradient">zFlac</p>
					<span class="text-[var(--border)]">|</span>
					<p class="text-xs text-[var(--muted-foreground)]">API Documentation</p>
				</div>
				<div class="flex items-center gap-2 text-[11px] font-mono uppercase tracking-[0.15em] text-[var(--muted-foreground)]">
					<span class="relative h-1.5 w-1.5">
						<span class="status-dot absolute"></span>
					</span>
					<span>v1.0</span>
				</div>
			</div>
		</div>
	</footer>
</div>
