<script lang="ts">
	import { ChevronDown, ChevronUp } from 'lucide-svelte';
	import type { ApiEndpoint } from '$lib/data/api-docs';
	import CodeBlock from './CodeBlock.svelte';
	import Playground from './Playground.svelte';

	interface Props {
		endpoint: ApiEndpoint;
	}

	let { endpoint }: Props = $props();
	let expanded = $state(false);

	const contentId = $derived(`endpoint-content-${endpoint.id}`);

	function getMethodColor(method: string): string {
		switch (method) {
			case 'GET':
				return 'bg-emerald-500/15 text-emerald-500';
			case 'POST':
				return 'bg-amber-500/15 text-amber-500';
			case 'PUT':
				return 'bg-blue-500/15 text-blue-500';
			case 'DELETE':
				return 'bg-red-500/15 text-red-500';
			default:
				return 'bg-[var(--secondary)] text-[var(--muted-foreground)]';
		}
	}
</script>

<div class="glass-card rounded-2xl overflow-hidden bg-[var(--card)] luxury-hover">
	<button
		onclick={() => (expanded = !expanded)}
		aria-expanded={expanded}
		aria-controls={contentId}
		class="w-full flex items-center gap-4 p-5 lg:p-6 text-left transition-all focus:outline-none focus:ring-2 focus:ring-[var(--ring)] focus:ring-inset {expanded ? 'bg-[var(--secondary)]/30' : 'hover:bg-[var(--secondary)]/20'}"
	>
		<span
			class="px-3 py-1.5 text-[10px] font-bold tracking-wider rounded-lg {getMethodColor(endpoint.method)}"
		>
			{endpoint.method}
		</span>
		<div class="flex-1 min-w-0">
			<code class="text-sm font-mono text-[var(--foreground)] block truncate">{endpoint.path}</code>
			<span class="text-xs text-[var(--muted-foreground)] block truncate mt-0.5">{endpoint.title}</span>
		</div>
		<span class="hidden sm:inline-block px-3 py-1 text-[10px] font-mono uppercase tracking-[0.15em] text-[var(--muted-foreground)] border border-[var(--border)] rounded-full">
			{expanded ? 'Open' : 'Closed'}
		</span>
		{#if expanded}
			<ChevronUp class="w-5 h-5 text-[var(--primary)] transition-transform flex-shrink-0" />
		{:else}
			<ChevronDown class="w-5 h-5 text-[var(--muted-foreground)] transition-transform flex-shrink-0" />
		{/if}
	</button>

	{#if expanded}
		<div id={contentId} role="region" class="border-t border-[var(--border)] p-5 lg:p-6 space-y-8 fade-up">
			<p class="text-sm text-[var(--muted-foreground)] leading-relaxed">{endpoint.description}</p>

			{#if endpoint.parameters?.length}
				<div>
					<h4 class="font-mono text-[10px] uppercase tracking-[0.2em] text-[var(--foreground)] mb-4">Parameters</h4>
					<div class="border border-[var(--border)] rounded-xl overflow-hidden">
						<table class="w-full text-sm">
							<thead>
								<tr class="bg-[var(--secondary)]/50">
									<th class="text-left px-4 py-3 font-medium text-[var(--muted-foreground)] text-xs">Name</th>
									<th class="text-left px-4 py-3 font-medium text-[var(--muted-foreground)] text-xs">Type</th>
									<th class="text-left px-4 py-3 font-medium text-[var(--muted-foreground)] text-xs">Required</th>
									<th class="text-left px-4 py-3 font-medium text-[var(--muted-foreground)] text-xs">Description</th>
								</tr>
							</thead>
							<tbody>
								{#each endpoint.parameters as param}
									<tr class="border-t border-[var(--border)]">
										<td class="px-4 py-3 font-mono text-[var(--foreground)] text-xs">{param.name}</td>
										<td class="px-4 py-3 text-[var(--muted-foreground)] text-xs">{param.type}</td>
										<td class="px-4 py-3">
											{#if param.required}
												<span class="text-[10px] font-medium text-emerald-500">Required</span>
											{:else}
												<span class="text-[10px] text-[var(--muted-foreground)]">Optional</span>
											{/if}
										</td>
										<td class="px-4 py-3 text-[var(--muted-foreground)] text-xs">{param.description}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>
			{/if}

			{#if endpoint.requestBody}
				<div>
					<h4 class="font-mono text-[10px] uppercase tracking-[0.2em] text-[var(--foreground)] mb-4">Request Body</h4>
					<p class="text-xs text-[var(--muted-foreground)] mb-3">{endpoint.requestBody.description}</p>
					<pre class="p-4 text-xs bg-[var(--secondary)]/50 rounded-xl overflow-x-auto border border-[var(--border)]"><code class="text-[var(--foreground)]">{JSON.stringify(endpoint.requestBody.example, null, 2)}</code></pre>
				</div>
			{/if}

			<div>
				<h4 class="font-mono text-[10px] uppercase tracking-[0.2em] text-[var(--foreground)] mb-4">Response Example</h4>
				<pre class="p-4 text-xs bg-[var(--secondary)]/50 rounded-xl overflow-x-auto border border-[var(--border)]"><code class="text-[var(--foreground)]">{JSON.stringify(endpoint.responseExample, null, 2)}</code></pre>
			</div>

			<div>
				<h4 class="font-mono text-[10px] uppercase tracking-[0.2em] text-[var(--foreground)] mb-4">Code Examples</h4>
				<CodeBlock {endpoint} />
			</div>

			<div>
				<Playground {endpoint} />
			</div>
		</div>
	{/if}
</div>
