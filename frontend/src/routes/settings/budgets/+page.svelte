<script lang="ts">
	import { onMount } from 'svelte';
	import { Plus, Pencil, Trash2, X, Check, AlertTriangle, Wallet, TrendingUp, TrendingDown, Calendar, Tag as TagIcon } from 'lucide-svelte';
	import { api, type Budget, type BudgetStatus, type Tag } from '$lib/api';
	import { toastStore } from '$lib/stores';

	// ============ State ============

	// Data
	let budgets: Budget[] = [];
	let budgetStatuses: BudgetStatus[] = [];
	let tags: Tag[] = [];

	// Loading states
	let isLoading = true;
	let isCreating = false;
	let isUpdating = false;
	let isDeleting = false;
	let isLoadingStatus = false;

	// Modal states
	let showCreateModal = false;
	let showDeleteModal = false;
	let budgetToDelete: Budget | null = null;

	// Editing states
	let editingBudgetId: string | null = null;
	let editAmount = 0;
	let editTagId: string | null = null;

	// New budget form
	let newBudgetAmount = 0;
	let newBudgetTagId: string | null = null;

	// Month selector
	let selectedMonth = getCurrentMonth();

	// ============ Constants ============

	function getCurrentMonth(): string {
		const now = new Date();
		return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
	}

	function formatMonthDisplay(monthStr: string): string {
		const [year, month] = monthStr.split('-');
		const date = new Date(parseInt(year), parseInt(month) - 1);
		return date.toLocaleDateString('en-US', { month: 'long', year: 'numeric' });
	}

	// ============ Computed ============

	$: sortedBudgets = [...budgets].sort((a, b) => {
		// Sort by tag name, with untagged budgets at the end
		const tagA = tags.find(t => t.id === a.tag_id)?.name || 'zzz_untagged';
		const tagB = tags.find(t => t.id === b.tag_id)?.name || 'zzz_untagged';
		return tagA.localeCompare(tagB);
	});

	$: canCreate = newBudgetAmount > 0 && !isCreating;

	$: canSaveEdit = editAmount > 0 && !isUpdating;

	$: totalBudgetLimit = budgets.reduce((sum, b) => sum + b.amount_limit, 0);

	$: totalSpent = budgetStatuses.reduce((sum, s) => sum + s.spent, 0);

	$: overallPercentage = totalBudgetLimit > 0 ? (totalSpent / totalBudgetLimit) * 100 : 0;

	// ============ Data Fetching ============

	async function fetchTags() {
		try {
			tags = await api.tags.list();
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to load tags';
			toastStore.error(message);
		}
	}

	async function fetchBudgets() {
		try {
			budgets = await api.budgets.list({ month: selectedMonth });
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to load budgets';
			toastStore.error(message);
		}
	}

	async function fetchBudgetStatus() {
		isLoadingStatus = true;
		try {
			budgetStatuses = await api.budgets.status(selectedMonth);
		} catch (err) {
			console.error('Failed to fetch budget status:', err);
			budgetStatuses = [];
		} finally {
			isLoadingStatus = false;
		}
	}

	async function loadData() {
		isLoading = true;
		await Promise.all([fetchTags(), fetchBudgets()]);
		await fetchBudgetStatus();
		isLoading = false;
	}

	function getBudgetStatus(budgetId: string): BudgetStatus | undefined {
		return budgetStatuses.find(s => s.budget_id === budgetId);
	}

	function getTagById(tagId: string | null): Tag | undefined {
		if (!tagId) return undefined;
		return tags.find(t => t.id === tagId);
	}

	// ============ Month Navigation ============

	function handleMonthChange(event: Event) {
		const target = event.target as HTMLInputElement;
		selectedMonth = target.value;
		loadData();
	}

	function goToPreviousMonth() {
		const [year, month] = selectedMonth.split('-').map(Number);
		const date = new Date(year, month - 2); // month - 2 because month is 1-indexed
		selectedMonth = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`;
		loadData();
	}

	function goToNextMonth() {
		const [year, month] = selectedMonth.split('-').map(Number);
		const date = new Date(year, month); // month is already 1-indexed, so this goes to next
		selectedMonth = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`;
		loadData();
	}

	// ============ Create Budget ============

	function openCreateModal() {
		newBudgetAmount = 0;
		newBudgetTagId = null;
		showCreateModal = true;
	}

	function closeCreateModal() {
		showCreateModal = false;
		newBudgetAmount = 0;
		newBudgetTagId = null;
	}

	async function createBudget() {
		if (newBudgetAmount <= 0) return;

		isCreating = true;
		try {
			await api.budgets.create({
				tag_id: newBudgetTagId,
				month: selectedMonth,
				amount_limit: newBudgetAmount
			});
			toastStore.success('Budget created successfully');
			closeCreateModal();
			await loadData();
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to create budget';
			toastStore.error(message);
		} finally {
			isCreating = false;
		}
	}

	// ============ Edit Budget ============

	function startEditing(budget: Budget) {
		editingBudgetId = budget.id;
		editAmount = budget.amount_limit;
		editTagId = budget.tag_id;
	}

	function cancelEditing() {
		editingBudgetId = null;
		editAmount = 0;
		editTagId = null;
	}

	async function saveEdit(budget: Budget) {
		if (editAmount <= 0) return;

		isUpdating = true;
		try {
			await api.budgets.update(budget.id, {
				amount_limit: editAmount,
				tag_id: editTagId
			});
			toastStore.success('Budget updated successfully');
			editingBudgetId = null;
			await loadData();
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to update budget';
			toastStore.error(message);
		} finally {
			isUpdating = false;
		}
	}

	// ============ Delete Budget ============

	function openDeleteModal(budget: Budget) {
		budgetToDelete = budget;
		showDeleteModal = true;
	}

	function closeDeleteModal() {
		showDeleteModal = false;
		budgetToDelete = null;
	}

	async function deleteBudget() {
		if (!budgetToDelete) return;

		isDeleting = true;
		try {
			await api.budgets.delete(budgetToDelete.id);
			toastStore.success('Budget deleted successfully');
			closeDeleteModal();
			await loadData();
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to delete budget';
			toastStore.error(message);
		} finally {
			isDeleting = false;
		}
	}

	// ============ Helper Functions ============

	function formatCurrency(amount: number): string {
		return new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: 'IDR',
			minimumFractionDigits: 0
		}).format(amount);
	}

	function getProgressColorClass(percentage: number): string {
		if (percentage >= 100) return 'bg-red-500';
		if (percentage >= 80) return 'bg-yellow-500';
		return 'bg-green-500';
	}

	function getStatusTextClass(percentage: number): string {
		if (percentage >= 100) return 'text-red-600';
		if (percentage >= 80) return 'text-yellow-600';
		return 'text-green-600';
	}

	function getBudgetDisplayName(budget: Budget): string {
		if (budget.tag_id) {
			const tag = getTagById(budget.tag_id);
			return tag?.name || 'Unknown Tag';
		}
		return 'Overall Budget';
	}

	function getTagColorClass(color: string | undefined): string {
		if (!color) return 'bg-gray-100 text-gray-700';
		const colorMap: Record<string, string> = {
			'#ef4444': 'bg-red-100 text-red-700',
			'#f97316': 'bg-orange-100 text-orange-700',
			'#f59e0b': 'bg-amber-100 text-amber-700',
			'#84cc16': 'bg-lime-100 text-lime-700',
			'#22c55e': 'bg-green-100 text-green-700',
			'#10b981': 'bg-emerald-100 text-emerald-700',
			'#14b8a6': 'bg-teal-100 text-teal-700',
			'#06b6d4': 'bg-cyan-100 text-cyan-700',
			'#0ea5e9': 'bg-sky-100 text-sky-700',
			'#3b82f6': 'bg-blue-100 text-blue-700',
			'#6366f1': 'bg-indigo-100 text-indigo-700',
			'#8b5cf6': 'bg-violet-100 text-violet-700',
			'#a855f7': 'bg-purple-100 text-purple-700',
			'#d946ef': 'bg-fuchsia-100 text-fuchsia-700',
			'#ec4899': 'bg-pink-100 text-pink-700',
			'#f43f5e': 'bg-rose-100 text-rose-700',
			'#6b7280': 'bg-gray-100 text-gray-700'
		};
		return colorMap[color] || 'bg-gray-100 text-gray-700';
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
			<h1 class="text-3xl font-bold text-gray-900 mb-2">Budgets</h1>
			<p class="text-gray-600">Set monthly spending limits and track your progress</p>
		</div>
		<button
			on:click={openCreateModal}
			class="flex items-center gap-2 px-6 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors shadow-sm self-start"
		>
			<Plus class="w-5 h-5" />
			<span class="font-medium">Add Budget</span>
		</button>
	</div>

	<!-- Month Selector -->
	<div class="mb-6 bg-white rounded-xl border border-gray-200 p-4">
		<div class="flex items-center justify-between">
			<button
				on:click={goToPreviousMonth}
				class="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
				</svg>
			</button>
			
			<div class="flex items-center gap-3">
				<Calendar class="w-5 h-5 text-gray-500" />
				<input
					type="month"
					value={selectedMonth}
					on:change={handleMonthChange}
					class="px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary font-medium text-gray-900"
				/>
			</div>
			
			<button
				on:click={goToNextMonth}
				class="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
			</button>
		</div>
	</div>

	<!-- Overall Summary Card -->
	{#if !isLoading && budgets.length > 0}
		<div class="mb-6 bg-white rounded-xl border border-gray-200 p-6">
			<div class="flex items-start justify-between mb-4">
				<div>
					<h2 class="text-lg font-semibold text-gray-900">Monthly Overview</h2>
					<p class="text-sm text-gray-500">{formatMonthDisplay(selectedMonth)}</p>
				</div>
				<div class="text-right">
					<div class="text-2xl font-bold {getStatusTextClass(overallPercentage)}">
						{formatCurrency(totalSpent)}
					</div>
					<div class="text-sm text-gray-500">
						of {formatCurrency(totalBudgetLimit)} budget
					</div>
				</div>
			</div>
			
			<!-- Progress Bar -->
			<div class="relative h-4 bg-gray-100 rounded-full overflow-hidden">
				<div
					class="absolute top-0 left-0 h-full {getProgressColorClass(overallPercentage)} transition-all duration-300"
					style="width: {Math.min(overallPercentage, 100)}%"
				></div>
			</div>
			
			<div class="flex items-center justify-between mt-2 text-sm">
				<span class="text-gray-500">{overallPercentage.toFixed(1)}% used</span>
				<span class="{getStatusTextClass(overallPercentage)}">
					{#if overallPercentage >= 100}
						<TrendingUp class="w-4 h-4 inline mr-1" />
						Over budget
					{:else if overallPercentage >= 80}
						<TrendingUp class="w-4 h-4 inline mr-1" />
						Approaching limit
					{:else}
						<TrendingDown class="w-4 h-4 inline mr-1" />
						On track
					{/if}
				</span>
			</div>
		</div>
	{/if}

	<!-- Loading State -->
	{#if isLoading}
		<div class="space-y-4">
			{#each Array(4) as _}
				<div class="bg-white rounded-xl border border-gray-200 p-6 animate-pulse">
					<div class="flex items-center justify-between mb-4">
						<div class="flex items-center gap-4">
							<div class="w-12 h-12 rounded-full bg-gray-200"></div>
							<div class="space-y-2">
								<div class="h-4 bg-gray-200 rounded w-32"></div>
								<div class="h-3 bg-gray-200 rounded w-24"></div>
							</div>
						</div>
						<div class="h-4 bg-gray-200 rounded w-24"></div>
					</div>
					<div class="h-3 bg-gray-200 rounded w-full"></div>
				</div>
			{/each}
		</div>

		<!-- Budgets List -->
	{:else if budgets.length > 0}
		<div class="space-y-4">
			{#each sortedBudgets as budget (budget.id)}
				{@const status = getBudgetStatus(budget.id)}
				{@const tag = getTagById(budget.tag_id)}
				<div class="bg-white rounded-xl border border-gray-200 p-6 hover:shadow-md transition-shadow">
					{#if editingBudgetId === budget.id}
						<!-- Edit Mode -->
						<div class="space-y-4">
							<!-- Tag Selector -->
							<div>
							<label class="block text-sm font-medium text-gray-700 mb-2">
								<TagIcon class="w-4 h-4 inline mr-1" />
								Tag (optional)
							</label>
								<select
									bind:value={editTagId}
									class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
								>
									<option value={null}>Overall Budget (no tag)</option>
									{#each tags as tag}
										<option value={tag.id}>{tag.name}</option>
									{/each}
								</select>
							</div>

							<!-- Amount Input -->
							<div>
								<label class="block text-sm font-medium text-gray-700 mb-1">
									<Wallet class="w-4 h-4 inline mr-1" />
									Budget Amount
								</label>
								<input
									type="number"
									bind:value={editAmount}
									min="0"
									step="1000"
									class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
									placeholder="Enter budget amount"
								/>
							</div>

							<!-- Action Buttons -->
							<div class="flex gap-2 pt-2">
								<button
									on:click={() => saveEdit(budget)}
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
						<div class="flex items-start justify-between mb-4">
							<div class="flex items-center gap-4">
								<!-- Tag Color or Budget Icon -->
								<div
									class="w-12 h-12 rounded-full flex-shrink-0 flex items-center justify-center {tag ? getTagColorClass(tag.color) : 'bg-primary/10'}"
								>
									{#if tag}
										<TagIcon class="w-6 h-6" />
									{:else}
										<Wallet class="w-6 h-6 text-primary" />
									{/if}
								</div>

								<!-- Budget Info -->
								<div>
									<h3 class="font-semibold text-gray-900">{getBudgetDisplayName(budget)}</h3>
									<p class="text-sm text-gray-500">
										Budget: {formatCurrency(budget.amount_limit)}
									</p>
								</div>
							</div>

							<!-- Action Buttons -->
							<div class="flex items-center gap-1 flex-shrink-0">
								<button
									on:click={() => startEditing(budget)}
									class="p-2 text-gray-400 hover:text-primary hover:bg-primary/10 rounded-lg transition-colors"
									title="Edit budget"
								>
									<Pencil class="w-4 h-4" />
								</button>
								<button
									on:click={() => openDeleteModal(budget)}
									class="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
									title="Delete budget"
								>
									<Trash2 class="w-4 h-4" />
								</button>
							</div>
						</div>

						<!-- Progress Section -->
						{#if status}
							<div class="mt-4">
								<div class="flex items-center justify-between mb-2 text-sm">
									<span class="text-gray-600">
										Spent: {formatCurrency(status.spent)}
									</span>
									<span class="{getStatusTextClass(status.percentage)}">
										{status.percentage.toFixed(1)}%
									</span>
								</div>
								
								<!-- Progress Bar -->
								<div class="relative h-3 bg-gray-100 rounded-full overflow-hidden">
									<div
										class="absolute top-0 left-0 h-full {getProgressColorClass(status.percentage)} transition-all duration-300"
										style="width: {Math.min(status.percentage, 100)}%"
									></div>
								</div>
								
								<div class="flex items-center justify-between mt-2 text-sm">
									<span class="text-gray-500">
										Remaining: {formatCurrency(status.remaining)}
									</span>
									{#if status.percentage >= 100}
										<span class="text-red-600 font-medium">
											Over budget by {formatCurrency(Math.abs(status.remaining))}
										</span>
									{:else if status.percentage >= 80}
										<span class="text-yellow-600 font-medium">
											Approaching limit
										</span>
									{:else}
										<span class="text-green-600 font-medium">
											On track
										</span>
									{/if}
								</div>
							</div>
						{:else if isLoadingStatus}
							<div class="mt-4 animate-pulse">
								<div class="h-3 bg-gray-200 rounded-full w-full"></div>
							</div>
						{:else}
							<div class="mt-4 text-sm text-gray-500">
								No spending data available for this period
							</div>
						{/if}
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
				<Wallet class="w-8 h-8 text-primary" />
			</div>
			<h3 class="text-lg font-medium text-gray-900 mb-2">No budgets set</h3>
			<p class="text-gray-500 mb-6 max-w-md mx-auto">
				Create budgets to track your spending and stay within your limits. You can set budgets for specific tags or an overall monthly budget.
			</p>
			<button
				on:click={openCreateModal}
				class="inline-flex items-center gap-2 px-6 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
			>
				<Plus class="w-5 h-5" />
				Create Your First Budget
			</button>
		</div>
	{/if}
</div>

<!-- Create Budget Modal -->
{#if showCreateModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm"
		on:click={closeCreateModal}
	>
		<div
			class="bg-white rounded-xl shadow-xl max-w-md w-full p-6"
			on:click|stopPropagation
		>
			<h2 class="text-xl font-semibold text-gray-900 mb-1">Create New Budget</h2>
			<p class="text-gray-500 text-sm mb-6">Set a spending limit for {formatMonthDisplay(selectedMonth)}</p>

			<div class="space-y-4">
				<!-- Tag Selector -->
				<div>
				<label for="budget-tag" class="block text-sm font-medium text-gray-700 mb-2">
					<TagIcon class="w-4 h-4 inline mr-1" />
					Tag (optional)
				</label>
					<select
						id="budget-tag"
						bind:value={newBudgetTagId}
						class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
					>
						<option value={null}>Overall Budget (no tag)</option>
						{#each tags as tag}
							<option value={tag.id}>{tag.name}</option>
						{/each}
					</select>
					<p class="text-xs text-gray-500 mt-1">
						Select a tag to track spending for a specific category, or leave empty for overall budget
					</p>
				</div>

				<!-- Amount Input -->
				<div>
					<label for="budget-amount" class="block text-sm font-medium text-gray-700 mb-1">
						<Wallet class="w-4 h-4 inline mr-1" />
						Budget Amount
					</label>
					<input
						type="number"
						id="budget-amount"
						bind:value={newBudgetAmount}
						min="0"
						step="1000"
						placeholder="e.g., 1000000"
						class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
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
					on:click={createBudget}
					disabled={!canCreate}
					class="flex-1 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{isCreating ? 'Creating...' : 'Create Budget'}
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- Delete Budget Modal -->
{#if showDeleteModal && budgetToDelete}
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
					<h2 class="text-xl font-semibold text-gray-900">Delete Budget</h2>
					<p class="text-gray-500 text-sm">This action cannot be undone</p>
				</div>
			</div>

			<div class="bg-gray-50 rounded-lg p-4 mb-6">
				<div class="flex items-center gap-3">
					{@const tag = getTagById(budgetToDelete.tag_id)}
					<div
						class="w-10 h-10 rounded-full flex-shrink-0 flex items-center justify-center {tag ? getTagColorClass(tag.color) : 'bg-primary/10'}"
					>
						{#if tag}
							<TagIcon class="w-5 h-5" />
						{:else}
							<Wallet class="w-5 h-5 text-primary" />
						{/if}
					</div>
					<div>
						<span class="font-medium text-gray-900 block">{getBudgetDisplayName(budgetToDelete)}</span>
						<span class="text-sm text-gray-500">{formatMonthDisplay(budgetToDelete.month)}</span>
					</div>
				</div>
				<div class="mt-3 pt-3 border-t border-gray-200">
					<p class="text-sm text-gray-600">
						Budget limit: <span class="font-medium">{formatCurrency(budgetToDelete.amount_limit)}</span>
					</p>
				</div>
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
					on:click={deleteBudget}
					disabled={isDeleting}
					class="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{isDeleting ? 'Deleting...' : 'Delete Budget'}
				</button>
			</div>
		</div>
	</div>
{/if}
