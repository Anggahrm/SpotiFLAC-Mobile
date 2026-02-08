<script lang="ts">
	import { Copy, Check } from 'lucide-svelte';
	import type { ApiEndpoint } from '$lib/data/api-docs';
	import { generateCurl, generateFetch, generateAxios, generatePython } from '$lib/data/api-docs';

	interface Props {
		endpoint: ApiEndpoint;
	}

	let { endpoint }: Props = $props();

	type Tab = 'curl' | 'fetch' | 'axios' | 'python';
	let activeTab: Tab = $state('curl');
	let copied = $state(false);

	const tabs: { id: Tab; label: string }[] = [
		{ id: 'curl', label: 'cURL' },
		{ id: 'fetch', label: 'Fetch' },
		{ id: 'axios', label: 'Axios' },
		{ id: 'python', label: 'Python' }
	];

	const code = $derived.by(() => {
		switch (activeTab) {
			case 'curl':
				return generateCurl(endpoint);
			case 'fetch':
				return generateFetch(endpoint);
			case 'axios':
				return generateAxios(endpoint);
			case 'python':
				return generatePython(endpoint);
		}
	});

	async function copyCode() {
		try {
			await navigator.clipboard.writeText(code);
			copied = true;
			setTimeout(() => (copied = false), 2000);
		} catch {
			// Clipboard API not available or permission denied
		}
	}
</script>

<div class="border border-[var(--border)] rounded-lg overflow-hidden bg-[var(--card)]">
	<!-- Tabs -->
	<div class="flex border-b border-[var(--border)] bg-[var(--secondary)]">
		{#each tabs as tab}
			<button
				class="px-3 py-2 text-xs font-medium transition-colors {activeTab === tab.id
					? 'text-[var(--foreground)] bg-[var(--card)] border-b-2 border-[var(--foreground)]'
					: 'text-[var(--muted-foreground)] hover:text-[var(--foreground)]'}"
				onclick={() => (activeTab = tab.id)}
			>
				{tab.label}
			</button>
		{/each}
		<div class="flex-1"></div>
		<button
			onclick={copyCode}
			class="px-3 py-2 text-xs text-[var(--muted-foreground)] hover:text-[var(--foreground)] transition-colors flex items-center gap-1"
		>
			{#if copied}
				<Check class="w-3.5 h-3.5 text-emerald-500" />
				<span class="text-emerald-500">Copied</span>
			{:else}
				<Copy class="w-3.5 h-3.5" />
				<span>Copy</span>
			{/if}
		</button>
	</div>

	<!-- Code -->
	<pre class="p-4 text-xs overflow-x-auto"><code class="text-[var(--foreground)]">{code}</code></pre>
</div>
