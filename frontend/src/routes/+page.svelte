<script lang="ts">
	import { isAuthenticated, login } from '$lib/auth';
	import { goto } from '$app/navigation';

	$effect(() => {
		if ($isAuthenticated) {
			goto('/readlist');
		}
	});
</script>

<main class="flex min-h-screen flex-col items-center justify-center gap-6 p-8">
	<div class="text-center">
		<h1 class="h1 mb-2">BookList</h1>
		<p class="text-surface-400 text-lg">Your personal reading list, powered by Open Library.</p>
	</div>

	<div class="flex flex-col items-center gap-3">
		<button class="btn preset-filled-primary-500 px-8 py-3 text-lg" onclick={login}>
			Get Started
		</button>
		<p class="text-surface-500 text-sm">Sign in with your account to manage your reading list.</p>
	</div>
</main>
