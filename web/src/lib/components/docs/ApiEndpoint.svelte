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
</script>

<div class="border border-[var(--border)] rounded-lg overflow-hidden bg-[var(--card)]">
	<!-- Header -->
	<button
		onclick={() => (expanded = !expanded)}
		aria-expanded={expanded}
		aria-controls="endpoint-content-{endpoint.id}"
		class="w-full flex items-center gap-3 p-4 text-left hover:bg-[var(--secondary)] transition-colors focus:outline-none focus:ring-2 focus:ring-[var(--ring)] focus:ring-inset"
	>
		<span
			class="px-2 py-1 text-[10px] font-bold rounded {endpoint.method === 'GET'
				? 'bg-emerald-500/20 text-emerald-500'
				: 'bg-amber-500/20 text-amber-500'}"
		>
			{endpoint.method}
		</span>
		<code class="text-sm font-mono text-[var(--foreground)]">{endpoint.path}</code>
		<span class="flex-1 text-sm text-[var(--muted-foreground)] truncate">{endpoint.title}</span>
		{#if expanded}
			<ChevronUp class="w-4 h-4 text-[var(--muted-foreground)]" />
		{:else}
			<ChevronDown class="w-4 h-4 text-[var(--muted-foreground)]" />
		{/if}
	</button>

	<!-- Content -->
	{#if expanded}
		<div id="endpoint-content-{endpoint.id}" role="region" class="border-t border-[var(--border)] p-4 space-y-6">
			<!-- Description -->
			<p class="text-sm text-[var(--muted-foreground)]">{endpoint.description}</p>

			<!-- Parameters -->
			{#if endpoint.parameters?.length}
				<div>
					<h4 class="text-xs font-medium text-[var(--foreground)] mb-3">Parameters</h4>
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

			<!-- Request Body -->
			{#if endpoint.requestBody}
				<div>
					<h4 class="text-xs font-medium text-[var(--foreground)] mb-3">Request Body</h4>
					<p class="text-xs text-[var(--muted-foreground)] mb-2">{endpoint.requestBody.description}</p>
					<pre class="p-3 text-xs bg-[var(--secondary)] rounded-md overflow-x-auto border border-[var(--border)]"><code class="text-[var(--foreground)]">{JSON.stringify(endpoint.requestBody.example, null, 2)}</code></pre>
				</div>
			{/if}

			<!-- Response Example -->
			<div>
				<h4 class="text-xs font-medium text-[var(--foreground)] mb-3">Response Example</h4>
				<pre class="p-3 text-xs bg-[var(--secondary)] rounded-md overflow-x-auto border border-[var(--border)]"><code class="text-[var(--foreground)]">{JSON.stringify(endpoint.responseExample, null, 2)}</code></pre>
			</div>

			<!-- Code Examples -->
			<div>
				<h4 class="text-xs font-medium text-[var(--foreground)] mb-3">Code Examples</h4>
				<CodeBlock {endpoint} />
			</div>

			<!-- Playground -->
			<div>
				<Playground {endpoint} />
			</div>
		</div>
	{/if}
</div>
