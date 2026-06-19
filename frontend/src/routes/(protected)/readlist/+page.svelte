<script lang="ts">
	import { onMount } from 'svelte';
	import { RatingGroup } from '@skeletonlabs/skeleton-svelte';
	import { api, type BookResponse } from '$lib/api';

	const STATUS_LABELS: Record<BookResponse['status'], string> = {
		want_to_read: 'Want to Read',
		reading: 'Reading',
		finished: 'Finished',
		abandoned: 'Abandoned'
	};

	let books = $state<BookResponse[]>([]);
	let loading = $state(true);
	let editingNotesId = $state<number | null>(null);
	let notesDraft = $state('');
	let confirmDeleteId = $state<number | null>(null);

	onMount(async () => {
		await loadList();
	});

	async function loadList() {
		loading = true;
		const res = await api.get('/readlist/');
		if (res.ok) books = await res.json();
		loading = false;
	}

	async function patchBook(id: number, body: Partial<Pick<BookResponse, 'status' | 'rating' | 'notes'>>) {
		const res = await api.patch(`/readlist/${id}`, body);
		if (res.ok) {
			const updated: BookResponse = await res.json();
			books = books.map((b) => (b.id === id ? updated : b));
		}
	}

	function onStatusChange(id: number, e: Event) {
		const val = (e.currentTarget as HTMLSelectElement).value as BookResponse['status'];
		patchBook(id, { status: val });
	}

	function onRatingChange(id: number, value: number) {
		if (value > 0) patchBook(id, { rating: value });
	}

	function openNotes(book: BookResponse) {
		editingNotesId = book.id;
		notesDraft = book.notes ?? '';
	}

	function saveNotes(book: BookResponse) {
		if (notesDraft !== (book.notes ?? '')) {
			patchBook(book.id, { notes: notesDraft });
		}
		editingNotesId = null;
	}

	async function deleteBook(id: number) {
		const res = await api.delete(`/readlist/${id}`);
		if (res.ok || res.status === 204) {
			books = books.filter((b) => b.id !== id);
		}
		confirmDeleteId = null;
	}

	function coverUrl(workId: string) {
		return `https://covers.openlibrary.org/b/olid/${workId}-M.jpg`;
	}
</script>

<div class="mx-auto max-w-3xl">
	<div class="mb-6 flex items-center justify-between">
		<h2 class="h2">My Reading List</h2>
		<a href="/search" class="btn preset-outlined-primary-500 btn-sm">+ Search books</a>
	</div>

	{#if loading}
		<div class="flex justify-center py-16">
			<span class="text-surface-400">Loading…</span>
		</div>
	{:else if books.length === 0}
		<div class="flex flex-col items-center gap-4 py-16 text-center">
			<p class="text-surface-400 text-lg">Your reading list is empty.</p>
			<a href="/search" class="btn preset-filled-primary-500">Search for books</a>
		</div>
	{:else}
		<ul class="flex flex-col gap-4">
			{#each books as book (book.id)}
				<li class="card bg-surface-800 flex gap-4 overflow-hidden rounded-lg p-4">
					<!-- Cover -->
					<div class="bg-surface-700 flex h-28 w-20 shrink-0 items-center justify-center overflow-hidden rounded">
						<img
							src={coverUrl(book.work_id)}
							alt="Cover for {book.title}"
							class="h-full w-full object-cover"
							onerror={(e) => {
								(e.currentTarget as HTMLImageElement).style.display = 'none';
							}}
						/>
					</div>

					<!-- Content -->
					<div class="flex flex-1 flex-col gap-3 overflow-hidden">
						<!-- Title + delete -->
						<div class="flex items-start justify-between gap-2">
							<div>
								<p class="font-semibold leading-tight">{book.title}</p>
								<p class="text-surface-400 text-sm">{book.authors}</p>
							</div>
							<div class="shrink-0">
								{#if confirmDeleteId === book.id}
									<span class="text-surface-400 text-sm">Delete?</span>
									<button
										class="btn preset-filled-error-500 btn-sm ml-1"
										onclick={() => deleteBook(book.id)}
									>
										Yes
									</button>
									<button
										class="btn preset-outlined-surface-500 btn-sm ml-1"
										onclick={() => (confirmDeleteId = null)}
									>
										No
									</button>
								{:else}
									<button
										class="btn btn-sm text-surface-400 hover:text-error-400"
										onclick={() => (confirmDeleteId = book.id)}
										aria-label="Delete {book.title}"
									>
										✕
									</button>
								{/if}
							</div>
						</div>

						<!-- Status + Rating row -->
						<div class="flex flex-wrap items-center gap-4">
							<select
								class="select text-sm"
								value={book.status}
								onchange={(e) => onStatusChange(book.id, e)}
							>
								{#each Object.entries(STATUS_LABELS) as [val, label]}
									<option value={val}>{label}</option>
								{/each}
							</select>

							<RatingGroup.Root
								value={book.rating ?? 0}
								onValueChange={(d) => onRatingChange(book.id, d.value)}
							>
								<RatingGroup.Control class="flex gap-0.5">
									{#each [1, 2, 3, 4, 5] as i}
										<RatingGroup.Item index={i} class="text-warning-400 cursor-pointer text-xl" />
									{/each}
								</RatingGroup.Control>
							</RatingGroup.Root>

							{#if book.rating}
								<span class="text-surface-400 text-xs">{book.rating}/5</span>
							{/if}
						</div>

						<!-- Notes -->
						<div class="w-full">
							{#if editingNotesId === book.id}
								<textarea
									class="textarea w-full text-sm"
									rows="3"
									placeholder="Add notes…"
									bind:value={notesDraft}
									onblur={() => saveNotes(book)}
									autofocus
								></textarea>
							{:else}
								<button
									class="text-surface-400 hover:text-surface-200 w-full cursor-text text-left text-sm transition-colors"
									onclick={() => openNotes(book)}
								>
									{book.notes || 'Add notes…'}
								</button>
							{/if}
						</div>
					</div>
				</li>
			{/each}
		</ul>
	{/if}
</div>
