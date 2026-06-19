<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type SearchResult, type BookResponse } from '$lib/api';

	let query = $state('');
	let results = $state<SearchResult[]>([]);
	let searching = $state(false);
	let searchError = $state('');

	type CardStatus = 'idle' | 'adding' | 'saved' | 'duplicate' | 'error';
	let cardStatus = $state<Record<string, CardStatus>>({});

	onMount(async () => {
		const res = await api.get('/readlist/');
		if (res.ok) {
			const books: BookResponse[] = await res.json();
			const initial: Record<string, CardStatus> = {};
			for (const b of books) initial[b.work_id] = 'saved';
			cardStatus = initial;
		}
	});

	async function search() {
		const q = query.trim();
		if (!q) return;
		searching = true;
		searchError = '';
		results = [];
		try {
			const res = await api.get(`/search?q=${encodeURIComponent(q)}`);
			if (!res.ok) throw new Error();
			results = await res.json();
		} catch {
			searchError = 'Search failed. Try again.';
		} finally {
			searching = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') search();
	}

	async function addToList(book: SearchResult) {
		cardStatus[book.work_id] = 'adding';
		try {
			const res = await api.post('/readlist/', {
				title: book.title,
				authors: book.authors.join(', '),
				work_id: book.work_id
			});
			cardStatus[book.work_id] = res.status === 409 ? 'duplicate' : res.ok ? 'saved' : 'error';
		} catch {
			cardStatus[book.work_id] = 'error';
		}
	}

	function coverUrl(workId: string) {
		return `https://covers.openlibrary.org/b/olid/${workId}-M.jpg`;
	}
</script>

<div class="mx-auto max-w-4xl">
	<h2 class="h2 mb-6">Search Books</h2>

	<div class="mb-8 flex gap-2">
		<input
			class="input flex-1"
			type="text"
			placeholder="Search by title, author, or ISBN…"
			bind:value={query}
			onkeydown={handleKeydown}
		/>
		<button
			class="btn preset-filled-primary-500"
			onclick={search}
			disabled={searching || !query.trim()}
		>
			{searching ? 'Searching…' : 'Search'}
		</button>
	</div>

	{#if searchError}
		<p class="text-error-400 mb-4">{searchError}</p>
	{/if}

	{#if results.length > 0}
		<p class="text-surface-400 mb-4 text-sm">{results.length} results</p>
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each results as book (book.work_id)}
				{@const status = cardStatus[book.work_id] ?? 'idle'}
				<div class="card bg-surface-800 flex flex-col overflow-hidden rounded-lg">
					<div class="bg-surface-700 flex h-40 items-center justify-center overflow-hidden">
						<img
							src={coverUrl(book.work_id)}
							alt="Cover for {book.title}"
							class="h-full w-full object-cover"
							onerror={(e) => {
								(e.currentTarget as HTMLImageElement).style.display = 'none';
							}}
						/>
					</div>
					<div class="flex flex-1 flex-col gap-2 p-4">
						<p class="font-semibold leading-tight">{book.title}</p>
						<p class="text-surface-400 text-sm">{book.authors.join(', ')}</p>
						{#if book.first_publish_year}
							<p class="text-surface-500 text-xs">{book.first_publish_year}</p>
						{/if}
						<div class="mt-auto pt-3">
							{#if status === 'idle'}
								<button
									class="btn preset-outlined-primary-500 btn-sm w-full"
									onclick={() => addToList(book)}
								>
									Add to list
								</button>
							{:else if status === 'adding'}
								<button class="btn btn-sm w-full" disabled>Adding…</button>
							{:else if status === 'saved'}
								<button class="btn preset-filled-success-500 btn-sm w-full" disabled>
									✓ Saved
								</button>
							{:else if status === 'duplicate'}
								<button class="btn preset-filled-surface-500 btn-sm w-full" disabled>
									Already in list
								</button>
							{:else if status === 'error'}
								<button
									class="btn preset-outlined-error-500 btn-sm w-full"
									onclick={() => addToList(book)}
								>
									Failed — retry
								</button>
							{/if}
						</div>
					</div>
				</div>
			{/each}
		</div>
	{:else if !searching && query && !searchError}
		<p class="text-surface-400">No results found for "{query}".</p>
	{/if}
</div>
