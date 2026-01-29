<script lang="ts">
	import '../app.css';
	import Navbar from '$lib/components/Navbar.svelte';
	import Toast from '$lib/components/Toast.svelte';
	import { toasts } from '$lib/stores/toasts';

	let { children } = $props();
</script>

<svelte:head>
	<title>zFlac Downloader</title>
	<meta name="description" content="Download high-quality FLAC music from Spotify and Deezer links" />
</svelte:head>

<div class="flex min-h-screen flex-col">
	<div class="hidden md:block">
		<Navbar />
	</div>

	<main class="flex-1 pb-20 md:pb-0">
		{@render children()}
	</main>

	<div class="md:hidden">
		<Navbar />
	</div>
</div>

<!-- Toast Container -->
<div class="fixed bottom-24 left-4 right-4 z-50 flex flex-col gap-2 md:bottom-4 md:left-auto md:right-4 md:w-96">
	{#each $toasts as toast (toast.id)}
		<Toast
			message={toast.message}
			type={toast.type}
			onclose={() => toasts.remove(toast.id)}
		/>
	{/each}
</div>
