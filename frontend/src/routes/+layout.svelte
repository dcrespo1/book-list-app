<script lang="ts">
	import { onMount } from 'svelte';
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { initAuth, isInitialized } from '$lib/auth';

	let { children } = $props();

	onMount(async () => {
		await initAuth();
	});
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

{#if $isInitialized}
	{@render children()}
{:else}
	<div class="flex h-screen items-center justify-center">
		<span class="text-surface-400">Loading...</span>
	</div>
{/if}
