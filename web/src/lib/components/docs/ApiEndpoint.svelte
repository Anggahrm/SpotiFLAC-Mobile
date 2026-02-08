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
</script>

<div class="glass-card border rounded-xl overflow-hidden bg-[var(--card)]">
	<button
		onclick={() => (expanded = !expanded)}
		aria-expanded={expanded}
		aria-controls={contentId}
		class="w-full flex items-center gap-2 sm:gap-3 p-3 sm:p-4 text-left transition-all focus:outline-none focus:ring-2 focus:ring-[var(--ring)] focus:ring-inset {expanded ? 'bg-[var(--secondary)]/75' : 'hover:bg-[var(--secondary)]/55'}"
	>
		<span
			class="px-2 py-1 text-[10px] font-bold rounded {endpoint.method === 'GET'
				? 'bg-emerald-500/20 text-emerald-500'
				: 'bg-amber-500/20 text-amber-500'}"
		>
			{endpoint.method}
		</span>
		<code class="text-xs sm:text-sm font-mono text-[var(--foreground)] truncate max-w-[38%] sm:max-w-none">{endpoint.path}</code>
		<span class="flex-1 text-xs sm:text-sm text-[var(--muted-foreground)] truncate">{endpoint.title}</span>
		<span class="rounded-full border border-[var(--border)] px-2 py-0.5 font-mono text-[9px] uppercase tracking-[0.12em] text-[var(--muted-foreground)]">{expanded ? 'Open' : 'Closed'}</span>
		{#if expanded}
			<ChevronUp class="w-4 h-4 text-[var(--primary)] transition-transform" />
		{:else}
			<ChevronDown class="w-4 h-4 text-[var(--muted-foreground)] transition-transform" />
		{/if}
	</button>

	{#if expanded}
		<div id={contentId} role="region" class="border-t border-[var(--border)] p-3 sm:p-4 space-y-5 fade-up">
			<p class="text-sm text-[var(--muted-foreground)]">{endpoint.description}</p>

			{#if endpoint.parameters?.length}
				<div>
					<h4 class="font-mono text-[10px] uppercase tracking-[0.12em] text-[var(--foreground)] mb-3">Parameters</h4>
					<div class="border border-[var(--border)] rounded-md overflow-hidden">
						<table class="w-full text-xs">
							<thead>
								<tr class="bg-[var(--secondary)]">
									<th class="text-left px-3 py-2 font-medium text-[var(--muted-foreground)]">Name</th>
									<th class="text-left px-3 py-2 font-medium text-[var(--muted-foreground)]">Type</th>
									<th class="text-left px-3 py-2 font-medium text-[var(--muted-foreground)]">Required</th>
									<th class="text-left px-3 py-2 font-medium text-[var(--muted-foreground)]">Description</th>
								</tr>
							</thead>
							<tbody>
								{#each endpoint.parameters as param}
									<tr class="border-t border-[var(--border)]">
										<td class="px-3 py-2 font-mono text-[var(--foreground)]">{param.name}</td>
										<td class="px-3 py-2 text-[var(--muted-foreground)]">{param.type}</td>
										<td class="px-3 py-2">
											{#if param.required}
												<span class="text-red-500">Yes</span>
											{:else}
												<span class="text-[var(--muted-foreground)]">No</span>
											{/if}
										</td>
										<td class="px-3 py-2 text-[var(--muted-foreground)]">{param.description}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>
			{/if}

			{#if endpoint.requestBody}
				<div>
					<h4 class="font-mono text-[10px] uppercase tracking-[0.12em] text-[var(--foreground)] mb-3">Request Body</h4>
					<p class="text-xs text-[var(--muted-foreground)] mb-2">{endpoint.requestBody.description}</p>
					<pre class="p-3 text-xs bg-[var(--secondary)] rounded-md overflow-x-auto border border-[var(--border)]"><code class="text-[var(--foreground)]">{JSON.stringify(endpoint.requestBody.example, null, 2)}</code></pre>
				</div>
			{/if}

			<div>
				<h4 class="font-mono text-[10px] uppercase tracking-[0.12em] text-[var(--foreground)] mb-3">Response Example</h4>
				<pre class="p-3 text-xs bg-[var(--secondary)] rounded-md overflow-x-auto border border-[var(--border)]"><code class="text-[var(--foreground)]">{JSON.stringify(endpoint.responseExample, null, 2)}</code></pre>
			</div>

			<div>
				<h4 class="font-mono text-[10px] uppercase tracking-[0.12em] text-[var(--foreground)] mb-3">Code Examples</h4>
				<CodeBlock {endpoint} />
			</div>

			<div>
				<Playground {endpoint} />
			</div>
		</div>
	{/if}
</div>
