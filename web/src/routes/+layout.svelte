<script lang="ts">
	import '../app.css';
	import Toast from '$lib/components/Toast.svelte';
	import { toasts } from '$lib/stores/toasts';

	let { children } = $props();
</script>

<svelte:head>
	<title>zFlac Downloader</title>
	<meta name="description" content="Download high-quality FLAC music from Spotify and Deezer links" />
</svelte:head>

{@render children()}

<!-- Toast Container -->
<div class="fixed bottom-4 left-4 right-4 z-50 flex flex-col gap-2 sm:left-auto sm:right-4 sm:w-96">
	{#each $toasts as toast (toast.id)}
		<Toast
			message={toast.message}
			type={toast.type}
			onclose={() => toasts.remove(toast.id)}
		/>
	{/each}
</div>
