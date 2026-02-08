<script lang="ts">
	import { Loader2, Play, Copy, Check } from 'lucide-svelte';
	import type { ApiEndpoint } from '$lib/data/api-docs';

	interface Props {
		endpoint: ApiEndpoint;
	}

	let { endpoint }: Props = $props();

	let paramValues: Record<string, string> = $state({});
	let bodyValue = $state('');
	let response: string | null = $state(null);
	let loading = $state(false);
	let error: string | null = $state(null);
	let copied = $state(false);

	// Initialize body value for POST requests
	$effect(() => {
		if (endpoint.requestBody) {
			bodyValue = JSON.stringify(endpoint.requestBody.example, null, 2);
		}
	});

	async function executeRequest() {
		loading = true;
		error = null;
		response = null;

		try {
			let url = endpoint.path;

			// Build URL with parameters for GET requests
			if (endpoint.method === 'GET' && endpoint.parameters?.length) {
				const params = new URLSearchParams();
				for (const param of endpoint.parameters) {
					const value = paramValues[param.name];
					if (value) {
						params.set(param.name, value);
					}
				}
				const queryString = params.toString();
				if (queryString) {
					url = `${url}?${queryString}`;
				}
			}

			const options: RequestInit = {
				method: endpoint.method,
				headers: {
					'Content-Type': 'application/json'
				}
			};

			if (endpoint.method === 'POST' && bodyValue) {
				options.body = bodyValue;
			}

			const res = await fetch(url, options);
			if (!res.ok) {
				throw new Error(`HTTP ${res.status}: ${res.statusText}`);
			}
			const data = await res.json();
			response = JSON.stringify(data, null, 2);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Request failed';
		} finally {
			loading = false;
		}
	}

	async function copyResponse() {
		if (response) {
			try {
				await navigator.clipboard.writeText(response);
				copied = true;
				setTimeout(() => (copied = false), 2000);
			} catch {
				// Clipboard API not available or permission denied
			}
		}
	}
</script>

<div class="border border-[var(--border)] rounded-lg overflow-hidden bg-[var(--card)]">
	<div class="px-4 py-3 border-b border-[var(--border)] bg-[var(--secondary)]">
		<span class="text-xs font-medium text-[var(--muted-foreground)]">PLAYGROUND</span>
	</div>

	<div class="p-4 space-y-4">
		<!-- Parameters -->
		{#if endpoint.parameters?.length}
			<div class="space-y-3">
				{#each endpoint.parameters as param}
					<div>
						<label class="block text-xs font-medium text-[var(--muted-foreground)] mb-1">
							{param.name}
							{#if param.required}
								<span class="text-red-500">*</span>
							{/if}
						</label>
						<input
							type="text"
							placeholder={param.example || param.description}
							bind:value={paramValues[param.name]}
							class="w-full h-9 px-3 text-xs border border-[var(--border)] bg-[var(--background)] rounded-md outline-none focus:border-[var(--foreground)] placeholder:text-[var(--muted-foreground)]"
						/>
					</div>
				{/each}
			</div>
		{/if}

		<!-- Request Body -->
		{#if endpoint.requestBody}
			<div>
				<label class="block text-xs font-medium text-[var(--muted-foreground)] mb-1">
					Request Body
				</label>
				<textarea
					bind:value={bodyValue}
					rows={8}
					class="w-full px-3 py-2 text-xs font-mono border border-[var(--border)] bg-[var(--background)] rounded-md outline-none focus:border-[var(--foreground)] resize-none"
				></textarea>
			</div>
		{/if}

		<!-- Execute Button -->
		<button
			onclick={executeRequest}
			disabled={loading}
			class="flex items-center justify-center gap-2 w-full py-2 text-xs font-medium bg-[var(--foreground)] text-[var(--background)] rounded-md hover:opacity-90 transition-all disabled:opacity-50"
		>
			{#if loading}
				<Loader2 class="w-3.5 h-3.5 animate-spin" />
				Executing...
			{:else}
				<Play class="w-3.5 h-3.5" />
				Try it
			{/if}
		</button>

		<!-- Response -->
		{#if response || error}
			<div class="border border-[var(--border)] rounded-md overflow-hidden">
				<div class="flex items-center justify-between px-3 py-2 border-b border-[var(--border)] bg-[var(--secondary)]">
					<span class="text-xs font-medium text-[var(--muted-foreground)]">Response</span>
					{#if response}
						<button
							onclick={copyResponse}
							class="text-xs text-[var(--muted-foreground)] hover:text-[var(--foreground)] flex items-center gap-1"
						>
							{#if copied}
								<Check class="w-3 h-3 text-emerald-500" />
							{:else}
								<Copy class="w-3 h-3" />
							{/if}
						</button>
					{/if}
				</div>
				<pre class="p-3 text-xs overflow-x-auto max-h-64 overflow-y-auto {error ? 'text-red-500' : 'text-[var(--foreground)]'}"><code>{error || response}</code></pre>
			</div>
		{/if}
	</div>
</div>
