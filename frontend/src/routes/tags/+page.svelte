<script lang="ts">
	import { onMount } from 'svelte';
	import { Plus, Pencil, Trash2, X, Check, AlertTriangle, Tag as TagIcon, Receipt } from 'lucide-svelte';
	import { api, type Tag, type Receipt as ReceiptType } from '$lib/api';
	import { toastStore } from '$lib/stores';
	import ColorPicker from '$components/ColorPicker.svelte';

	// ============ State ============

	// Data
	let tags: Tag[] = [];
	let receipts: ReceiptType[] = [];

	// Loading states
	let isLoading = true;
	let isCreating = false;
	let isUpdating = false;
	let isDeleting = false;

	// Modal states
	let showCreateModal = false;
	let showDeleteModal = false;
	let tagToDelete: Tag | null = null;

	// Editing states
	let editingTagId: string | null = null;
	let editName = '';
	let editColor = '';

	// New tag form
	let newTagName = '';
	let newTagColor = '#3b82f6';

	// ============ Constants ============

	const defaultColor = '#3b82f6';

	// ============ Computed ============

	$: sortedTags = [...tags].sort((a, b) => a.name.localeCompare(b.name));

	$: tagUsageCounts = (() => {
		const counts: Record<string, number> = {};
		receipts.forEach((receipt) => {
			receipt.tags.forEach((tagId) => {
				counts[tagId] = (counts[tagId] || 0) + 1;
			});
		});
		return counts;
	})();

	$: canCreate = newTagName.trim().length > 0 && !isCreating;

	$: canSaveEdit = editName.trim().length > 0 && !isUpdating;

	// ============ Data Fetching ============

	async function fetchTags() {
		try {
			tags = await api.tags.list();
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to load tags';
			toastStore.error(message);
		}
	}

	async function fetchReceipts() {
		try {
			// Fetch all receipts to calculate tag usage
			const response = await api.receipts.list({ limit: 1000 });
			receipts = response.data;
		} catch (err) {
			console.error('Failed to fetch receipts:', err);
		}
	}

	async function loadData() {
		isLoading = true;
		await Promise.all([fetchTags(), fetchReceipts()]);
		isLoading = false;
	}

	// ============ Create Tag ============

	function openCreateModal() {
		newTagName = '';
		newTagColor = defaultColor;
		showCreateModal = true;
	}

	function closeCreateModal() {
		showCreateModal = false;
		newTagName = '';
		newTagColor = defaultColor;
	}

	async function createTag() {
		if (!newTagName.trim()) return;

		isCreating = true;
		try {
			await api.tags.create({
				name: newTagName.trim(),
				color: newTagColor
			});
			toastStore.success('Tag created successfully');
			closeCreateModal();
			await fetchTags();
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to create tag';
			toastStore.error(message);
		} finally {
			isCreating = false;
		}
	}

	// ============ Edit Tag ============

	function startEditing(tag: Tag) {
		editingTagId = tag.id;
		editName = tag.name;
		editColor = tag.color || defaultColor;
	}

	function cancelEditing() {
		editingTagId = null;
		editName = '';
		editColor = '';
	}

	async function saveEdit(tag: Tag) {
		if (!editName.trim()) return;

		isUpdating = true;
		try {
			await api.tags.update(tag.id, {
				name: editName.trim(),
				color: editColor
			});
			toastStore.success('Tag updated successfully');
			editingTagId = null;
			await fetchTags();
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to update tag';
			toastStore.error(message);
		} finally {
			isUpdating = false;
		}
	}

	// ============ Delete Tag ============

	function openDeleteModal(tag: Tag) {
		tagToDelete = tag;
		showDeleteModal = true;
	}

	function closeDeleteModal() {
		showDeleteModal = false;
		tagToDelete = null;
	}

	async function deleteTag() {
		if (!tagToDelete) return;

		isDeleting = true;
		try {
			await api.tags.delete(tagToDelete.id);
			toastStore.success('Tag deleted successfully');
			closeDeleteModal();
			await fetchTags();
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to delete tag';
			toastStore.error(message);
		} finally {
			isDeleting = false;
		}
	}

	// ============ Helper Functions ============

	function getTagUsageCount(tagId: string): number {
		return tagUsageCounts[tagId] || 0;
	}

	// ============ Lifecycle ============

	onMount(() => {
		loadData();
	});
</script>

<div class="p-4 md:p-8 max-w-7xl mx-auto">
	<!-- Header -->
	<div class="mb-8 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
		<div>
			<h1 class="text-3xl font-bold text-gray-900 mb-2">Tags</h1>
			<p class="text-gray-600">Organize your receipts with color-coded tags</p>
		</div>
		<button
			on:click={openCreateModal}
			class="flex items-center gap-2 px-6 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors shadow-sm self-start"
		>
			<Plus class="w-5 h-5" />
			<span class="font-medium">Create Tag</span>
		</button>
	</div>

	<!-- Loading State -->
	{#if isLoading}
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
			{#each Array(8) as _}
				<div class="bg-white rounded-xl border border-gray-200 p-6 animate-pulse">
					<div class="flex items-center gap-4">
						<div class="w-12 h-12 rounded-full bg-gray-200"></div>
						<div class="flex-1 space-y-2">
							<div class="h-4 bg-gray-200 rounded w-24"></div>
							<div class="h-3 bg-gray-200 rounded w-16"></div>
						</div>
					</div>
				</div>
			{/each}
		</div>

		<!-- Tags Grid -->
	{:else if tags.length > 0}
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
			{#each sortedTags as tag (tag.id)}
				<div class="bg-white rounded-xl border border-gray-200 p-6 hover:shadow-md transition-shadow">
					{#if editingTagId === tag.id}
						<!-- Edit Mode -->
						<div class="space-y-4">
							<!-- Name Input -->
							<div>
								<label class="block text-sm font-medium text-gray-700 mb-1">Tag Name</label>
								<input
									type="text"
									bind:value={editName}
									class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary text-sm"
									placeholder="Enter tag name"
								/>
							</div>

							<!-- Color Picker -->
							<div>
								<label class="block text-sm font-medium text-gray-700 mb-2">Color</label>
								<ColorPicker selectedColor={editColor} on:select={(e) => (editColor = e.detail)} />
							</div>

							<!-- Action Buttons -->
							<div class="flex gap-2 pt-2">
								<button
									on:click={() => saveEdit(tag)}
									disabled={!canSaveEdit || isUpdating}
									class="flex-1 flex items-center justify-center gap-1 px-3 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed text-sm font-medium"
								>
									<Check class="w-4 h-4" />
									{isUpdating ? 'Saving...' : 'Save'}
								</button>
								<button
									on:click={cancelEditing}
									disabled={isUpdating}
									class="flex-1 flex items-center justify-center gap-1 px-3 py-2 border border-gray-200 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50 text-sm font-medium"
								>
									<X class="w-4 h-4" />
									Cancel
								</button>
							</div>
						</div>
					{:else}
						<!-- Display Mode -->
						<div class="flex items-start gap-4">
							<!-- Color Circle -->
							<div
								class="w-12 h-12 rounded-full flex-shrink-0 flex items-center justify-center"
								style="background-color: {tag.color || defaultColor}"
							>
								<TagIcon class="w-6 h-6 text-white/90" />
							</div>

							<!-- Tag Info -->
							<div class="flex-1 min-w-0">
								<h3 class="font-semibold text-gray-900 truncate">{tag.name}</h3>
								<p class="text-sm text-gray-500 flex items-center gap-1 mt-1">
									<Receipt class="w-3 h-3" />
									{getTagUsageCount(tag.id)} receipt{getTagUsageCount(tag.id) === 1 ? '' : 's'}
								</p>
							</div>

							<!-- Action Buttons -->
							<div class="flex items-center gap-1 flex-shrink-0">
								<button
									on:click={() => startEditing(tag)}
									class="p-2 text-gray-400 hover:text-primary hover:bg-primary/10 rounded-lg transition-colors"
									title="Edit tag"
								>
									<Pencil class="w-4 h-4" />
								</button>
								<button
									on:click={() => openDeleteModal(tag)}
									class="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
									title="Delete tag"
								>
									<Trash2 class="w-4 h-4" />
								</button>
							</div>
						</div>
					{/if}
				</div>
			{/each}
		</div>

		<!-- Empty State -->
	{:else}
		<div class="bg-white rounded-xl border border-gray-200 p-12 text-center">
		<div
			class="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center mx-auto mb-4"
		>
			<TagIcon class="w-8 h-8 text-primary" />
			</div>
			<h3 class="text-lg font-medium text-gray-900 mb-2">No tags yet</h3>
			<p class="text-gray-500 mb-6 max-w-md mx-auto">
				Create tags to organize your receipts. Tags help you categorize and quickly find receipts
				by type, store, or any custom grouping.
			</p>
			<button
				on:click={openCreateModal}
				class="inline-flex items-center gap-2 px-6 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
			>
				<Plus class="w-5 h-5" />
				Create Your First Tag
			</button>
		</div>
	{/if}
</div>

<!-- Create Tag Modal -->
{#if showCreateModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm"
		on:click={closeCreateModal}
	>
		<div
			class="bg-white rounded-xl shadow-xl max-w-md w-full p-6"
			on:click|stopPropagation
		>
			<h2 class="text-xl font-semibold text-gray-900 mb-1">Create New Tag</h2>
			<p class="text-gray-500 text-sm mb-6">Add a new tag to organize your receipts</p>

			<div class="space-y-4">
				<!-- Tag Name Input -->
				<div>
					<label for="tag-name" class="block text-sm font-medium text-gray-700 mb-1">
						Tag Name
					</label>
					<input
						type="text"
						id="tag-name"
						bind:value={newTagName}
						placeholder="e.g., Groceries, Restaurant, Gas"
						class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
					/>
				</div>

				<!-- Color Picker -->
				<div>
					<label class="block text-sm font-medium text-gray-700 mb-2">Color</label>
					<ColorPicker
						selectedColor={newTagColor}
						on:select={(e) => (newTagColor = e.detail)}
					/>
				</div>
			</div>

			<!-- Action Buttons -->
			<div class="flex gap-3 mt-6">
				<button
					on:click={closeCreateModal}
					class="flex-1 px-4 py-2 border border-gray-200 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors font-medium"
				>
					Cancel
				</button>
				<button
					on:click={createTag}
					disabled={!canCreate}
					class="flex-1 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{isCreating ? 'Creating...' : 'Create Tag'}
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- Delete Tag Modal -->
{#if showDeleteModal && tagToDelete}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm"
		on:click={closeDeleteModal}
	>
		<div
			class="bg-white rounded-xl shadow-xl max-w-md w-full p-6"
			on:click|stopPropagation
		>
			<div class="flex items-center gap-3 mb-4">
				<div class="w-12 h-12 rounded-full bg-red-100 flex items-center justify-center flex-shrink-0">
					<AlertTriangle class="w-6 h-6 text-red-600" />
				</div>
				<div>
					<h2 class="text-xl font-semibold text-gray-900">Delete Tag</h2>
					<p class="text-gray-500 text-sm">This action cannot be undone</p>
				</div>
			</div>

			<div class="bg-gray-50 rounded-lg p-4 mb-6">
					<div class="flex items-center gap-3 mb-3">
						<div
							class="w-8 h-8 rounded-full flex-shrink-0"
							style="background-color: {tagToDelete.color || defaultColor}"
						></div>
						<span class="font-medium text-gray-900">{tagToDelete.name}</span>
					</div>

				{#if getTagUsageCount(tagToDelete.id) > 0}
					<p class="text-sm text-amber-700 flex items-start gap-2">
						<AlertTriangle class="w-4 h-4 flex-shrink-0 mt-0.5" />
						<span>
							This tag is currently used on {getTagUsageCount(tagToDelete.id)} receipt{getTagUsageCount(tagToDelete.id) === 1
								? ''
								: 's'}. Deleting it will remove the tag from
							<span class="font-semibold">all</span> of them.
						</span>
					</p>
				{:else}
					<p class="text-sm text-gray-600">This tag is not currently used on any receipts.</p>
				{/if}
			</div>

			<!-- Action Buttons -->
			<div class="flex gap-3">
				<button
					on:click={closeDeleteModal}
					class="flex-1 px-4 py-2 border border-gray-200 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors font-medium"
				>
					Cancel
				</button>
				<button
					on:click={deleteTag}
					disabled={isDeleting}
					class="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{isDeleting ? 'Deleting...' : 'Delete Tag'}
				</button>
			</div>
		</div>
	</div>
{/if}
