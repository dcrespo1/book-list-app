<script lang="ts">
	import { isAuthenticated, login, logout } from '$lib/auth';
	import { page } from '$app/stores';

	let { children } = $props();

	$effect(() => {
		if (!$isAuthenticated) {
			login();
		}
	});
</script>

{#if $isAuthenticated}
	<div class="flex min-h-screen flex-col">
		<header class="bg-surface-900 border-surface-700 flex items-center justify-between border-b px-6 py-3">
			<a href="/readlist" class="h3 font-bold">BookList</a>
			<nav class="flex items-center gap-6">
				<a
					href="/search"
					class="hover:text-primary-400 transition-colors"
					class:text-primary-400={$page.url.pathname === '/search'}
				>
					Search
				</a>
				<a
					href="/readlist"
					class="hover:text-primary-400 transition-colors"
					class:text-primary-400={$page.url.pathname === '/readlist'}
				>
					My List
				</a>
				<button class="btn preset-outlined-surface-500 btn-sm" onclick={logout}>
					Sign out
				</button>
			</nav>
		</header>

		<main class="flex-1 p-6">
			{@render children()}
		</main>
	</div>
{/if}
