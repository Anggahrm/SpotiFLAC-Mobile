<script lang="ts">
	import '../app.css';
	import Toast from '$lib/components/Toast.svelte';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import { toasts } from '$lib/stores/toasts';
	import { theme } from '$lib/stores/theme';
	import { browser } from '$app/environment';

	let { children } = $props();

	$effect(() => {
		if (browser) {
			document.documentElement.classList.toggle('light', $theme === 'light');
		}
	});
</script>

<svelte:head>
	<title>zFlac Downloader</title>
	<meta name="description" content="Download high-quality FLAC music from Spotify and Deezer links" />
</svelte:head>

<div class="fixed bottom-4 right-4 z-40 sm:bottom-6 sm:right-6">
	<ThemeToggle />
</div>

{@render children()}

<div class="fixed top-4 left-4 right-4 z-50 flex flex-col gap-2 sm:left-auto sm:right-6 sm:w-[26rem]">
	{#each $toasts as toast (toast.id)}
		<Toast
			message={toast.message}
			type={toast.type}
			onclose={() => toasts.remove(toast.id)}
		/>
	{/each}
</div>
