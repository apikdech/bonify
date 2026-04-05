<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, type Receipt, type Tag, type UpdateReceiptRequest } from '$lib/api';
	import { toastStore, pendingCountStore } from '$lib/stores';
	import {
		CheckCircle,
		X,
		RefreshCw,
		Trash2,
		Inbox,
		Receipt as ReceiptIcon,
		ArrowLeft,
		AlertTriangle,
		Loader2,
		Save,
		RotateCcw,
		Eye,
		Bot,
		FileText,
		MessageCircle,
		Smartphone
	} from 'lucide-svelte';

	// ============ State ============
	let receipts: Receipt[] = [];
	let tags: Tag[] = [];
	let isLoading = true;
	let error: string | null = null;
	let isProcessing = false;
	let processingId: string | null = null;
	let deleteConfirmId: string | null = null;

	// Inline editing state
	let editingReceiptId: string | null = null;
	let editingField: string | null = null;
	let editValues: {
		title?: string;
		receipt_date?: string;
		total?: number;
		tags?: string[];
	} = {};

	// ============ Constants ============
	const SOURCE_ICONS: Record<string, typeof Bot> = {
		manual: FileText,
		telegram: MessageCircle,
		discord: Smartphone,
		ocr: Bot
	};

	// ============ Helper Functions ============
	function formatCurrency(amount: number, currency: string = 'IDR'): string {
		return new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: currency,
			minimumFractionDigits: 0
		}).format(amount);
	}

	function formatDate(dateString: string | undefined): string {
		if (!dateString) return '-';
		const date = new Date(dateString);
		return date.toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		});
	}

	function getConfidenceColor(confidence: number | undefined): { bg: string; text: string; border: string } {
		if (confidence === undefined) return { bg: 'bg-gray-100', text: 'text-gray-700', border: 'border-gray-300' };
		if (confidence >= 85) return { bg: 'bg-green-100', text: 'text-green-700', border: 'border-green-300' };
		if (confidence >= 60) return { bg: 'bg-yellow-100', text: 'text-yellow-700', border: 'border-yellow-300' };
		return { bg: 'bg-red-100', text: 'text-red-700', border: 'border-red-300' };
	}

	function getConfidenceLabel(confidence: number | undefined): string {
		if (confidence === undefined) return 'N/A';
		return `${Math.round(confidence)}%`;
	}

	function getTagById(tagId: string): Tag | undefined {
		return tags.find((t) => t.id === tagId);
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

	function getSourceIcon(source: string): typeof Bot {
		return SOURCE_ICONS[source] || FileText;
	}

	// ============ Data Fetching ============
	async function fetchTags() {
		try {
			tags = await api.tags.list();
		} catch (err) {
			console.error('Failed to fetch tags:', err);
		}
	}

	async function fetchReceipts() {
		isLoading = true;
		error = null;

		try {
			const response = await api.receipts.list({
				status: 'pending_review' as const,
				sort_by: 'created_at',
				sort_order: 'desc',
				limit: 100
			});
			receipts = response.data;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load receipts';
			toastStore.error('Failed to load review queue');
		} finally {
			isLoading = false;
		}
	}

	// ============ Actions ============
	async function handleConfirm(receiptId: string) {
		isProcessing = true;
		processingId = receiptId;
		try {
			await api.receipts.confirm(receiptId);
			toastStore.success('Receipt confirmed');
			pendingCountStore.decrement();
			receipts = receipts.filter((r) => r.id !== receiptId);
		} catch (err) {
			toastStore.error(err instanceof Error ? err.message : 'Failed to confirm receipt');
		} finally {
			isProcessing = false;
			processingId = null;
		}
	}

	async function handleReject(receiptId: string) {
		isProcessing = true;
		processingId = receiptId;
		try {
			await api.receipts.reject(receiptId);
			toastStore.success('Receipt rejected');
			pendingCountStore.decrement();
			receipts = receipts.filter((r) => r.id !== receiptId);
		} catch (err) {
			toastStore.error(err instanceof Error ? err.message : 'Failed to reject receipt');
		} finally {
			isProcessing = false;
			processingId = null;
		}
	}

	async function handleDelete(receiptId: string) {
		isProcessing = true;
		processingId = receiptId;
		try {
			await api.receipts.delete(receiptId);
			toastStore.success('Receipt deleted');
			pendingCountStore.decrement();
			receipts = receipts.filter((r) => r.id !== receiptId);
			deleteConfirmId = null;
		} catch (err) {
			toastStore.error(err instanceof Error ? err.message : 'Failed to delete receipt');
		} finally {
			isProcessing = false;
			processingId = null;
		}
	}

	function handleRerunOCR(_receiptId: string) {
		toastStore.info('OCR reprocessing scheduled');
		// Placeholder - would trigger backend reprocessing
	}

	async function handleViewDetails(receiptId: string) {
		goto(`/receipts/${receiptId}`);
	}

	// ============ Inline Editing ============
	function startEditing(receipt: Receipt, field: string) {
		editingReceiptId = receipt.id;
		editingField = field;
		editValues = {
			title: receipt.title,
			receipt_date: receipt.receipt_date || receipt.created_at.split('T')[0],
			total: receipt.total,
			tags: [...receipt.tags]
		};
	}

	function cancelEditing() {
		editingReceiptId = null;
		editingField = null;
		editValues = {};
	}

	async function saveEditing(receipt: Receipt) {
		if (!editingReceiptId) return;

		isProcessing = true;
		processingId = receipt.id;

		try {
			const updateData: UpdateReceiptRequest = {};
			if (editValues.title !== undefined && editValues.title !== receipt.title) {
				updateData.title = editValues.title;
			}
			if (editValues.receipt_date !== undefined && editValues.receipt_date !== receipt.receipt_date) {
				updateData.receipt_date = editValues.receipt_date;
			}
			if (editValues.total !== undefined && editValues.total !== receipt.total) {
				updateData.total = editValues.total;
			}
			if (editValues.tags !== undefined) {
				// Check if tags changed
				const currentTags = receipt.tags.sort();
				const newTags = editValues.tags.sort();
				if (JSON.stringify(currentTags) !== JSON.stringify(newTags)) {
					updateData.tags = editValues.tags;
				}
			}

			if (Object.keys(updateData).length > 0) {
				await api.receipts.update(receipt.id, updateData);
				toastStore.success('Receipt updated');

				// Update local receipt data
				const index = receipts.findIndex((r) => r.id === receipt.id);
				if (index !== -1) {
					receipts[index] = { ...receipts[index], ...updateData };
					receipts = [...receipts]; // Trigger reactivity
				}
			}

			editingReceiptId = null;
			editingField = null;
		} catch (err) {
			toastStore.error(err instanceof Error ? err.message : 'Failed to update receipt');
		} finally {
			isProcessing = false;
			processingId = null;
		}
	}

	function toggleTag(tagId: string) {
		if (!editValues.tags) return;
		const index = editValues.tags.indexOf(tagId);
		if (index === -1) {
			editValues.tags = [...editValues.tags, tagId];
		} else {
			editValues.tags = editValues.tags.filter((t) => t !== tagId);
		}
		editValues = { ...editValues };
	}

	// ============ Lifecycle ============
	onMount(() => {
		fetchTags();
		fetchReceipts();
	});
</script>

<div class="p-4 md:p-8 max-w-7xl mx-auto">
	<!-- Header -->
	<div class="mb-8">
		<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
			<div>
				<h1 class="text-3xl font-bold text-gray-900 mb-2">Review Queue</h1>
				<p class="text-gray-600">Receipts that need your review</p>
			</div>
			<div class="flex items-center gap-3">
				{#if receipts.length > 0}
					<span class="inline-flex items-center px-3 py-1.5 rounded-full text-sm font-medium bg-primary/10 text-primary border border-primary/20">
						<Inbox class="w-4 h-4 mr-2" />
						{receipts.length} pending
					</span>
				{/if}
				<button
					on:click={() => goto('/receipts')}
					class="flex items-center gap-2 px-4 py-2 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors text-gray-700"
				>
					<ReceiptIcon class="w-4 h-4" />
					<span>All Receipts</span>
				</button>
			</div>
		</div>
	</div>

	<!-- Error State -->
	{#if error}
		<div class="bg-red-50 border border-red-200 rounded-lg p-4 mb-6 flex items-center justify-between">
			<div class="flex items-center gap-3">
				<AlertTriangle class="w-5 h-5 text-red-600" />
				<span class="text-red-800">{error}</span>
			</div>
			<button
				on:click={fetchReceipts}
				class="flex items-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors"
			>
				<RefreshCw class="w-4 h-4" />
				Retry
			</button>
		</div>
	{/if}

	<!-- Loading State -->
	{#if isLoading && receipts.length === 0}
		<div class="space-y-4">
			{#each Array(3) as _}
				<div class="bg-white rounded-xl border border-gray-200 p-6 shadow-sm">
					<div class="flex flex-col md:flex-row gap-6">
						<!-- Image Skeleton -->
						<div class="w-full md:w-48 h-32 bg-gray-200 rounded-lg animate-pulse flex-shrink-0"></div>

						<!-- Content Skeleton -->
						<div class="flex-1 space-y-4">
							<div class="flex items-center gap-3">
								<div class="h-6 bg-gray-200 rounded w-1/3 animate-pulse"></div>
								<div class="h-5 bg-gray-200 rounded w-16 animate-pulse"></div>
							</div>
							<div class="h-4 bg-gray-200 rounded w-24 animate-pulse"></div>
							<div class="h-4 bg-gray-200 rounded w-32 animate-pulse"></div>
							<div class="flex gap-2">
								<div class="h-6 bg-gray-200 rounded w-20 animate-pulse"></div>
								<div class="h-6 bg-gray-200 rounded w-20 animate-pulse"></div>
							</div>
						</div>

						<!-- Actions Skeleton -->
						<div class="flex md:flex-col gap-2">
							{#each Array(3) as _}
								<div class="h-9 bg-gray-200 rounded w-24 animate-pulse"></div>
							{/each}
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}

	<!-- Empty State -->
	{#if !isLoading && receipts.length === 0}
		<div class="bg-white rounded-xl border border-gray-200 p-12 text-center shadow-sm">
			<div class="w-20 h-20 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-6">
				<CheckCircle class="w-10 h-10 text-green-600" />
			</div>
			<h3 class="text-xl font-semibold text-gray-900 mb-2">You're all caught up!</h3>
			<p class="text-gray-500 mb-8">No receipts need review.</p>
			<button
				on:click={() => goto('/receipts')}
				class="inline-flex items-center gap-2 px-6 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors shadow-sm"
			>
				<ArrowLeft class="w-5 h-5" />
				<span>View All Receipts</span>
			</button>
		</div>
	{/if}

	<!-- Receipt Cards -->
	{#if receipts.length > 0}
		<div class="space-y-4">
			{#each receipts as receipt}
				<div class="bg-white rounded-xl border border-gray-200 p-4 md:p-6 shadow-sm transition-all hover:shadow-md">
					<div class="flex flex-col lg:flex-row gap-4 lg:gap-6">
						<!-- Receipt Image Thumbnail -->
						<div class="w-full lg:w-48 h-32 bg-gray-100 rounded-lg flex-shrink-0 overflow-hidden border border-gray-200">
							{#if receipt.image_url}
								<img
									src={receipt.image_url}
									alt="Receipt"
									class="w-full h-full object-cover"
								/>
							{:else}
								<div class="w-full h-full flex items-center justify-center">
									<svelte:component this={getSourceIcon(receipt.source)} class="w-10 h-10 text-gray-400" />
								</div>
							{/if}
						</div>

						<!-- Receipt Details -->
						<div class="flex-1 space-y-4 min-w-0">
							<div class="flex flex-wrap items-center gap-3">
								<!-- OCR Confidence Badge -->
								{#if true}
									{@const confidenceColors = getConfidenceColor(receipt.ocr_confidence)}
									<span class="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium border {confidenceColors.bg} {confidenceColors.text} {confidenceColors.border}">
										<Bot class="w-3 h-3 mr-1" />
										{getConfidenceLabel(receipt.ocr_confidence)}
									</span>
								{/if}

								<!-- Status Badge -->
								<span class="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800 border border-yellow-200">
									Pending Review
								</span>
							</div>

							<!-- Shop Name -->
							<div>
								<label class="block text-xs font-medium text-gray-500 mb-1">Shop Name</label>
								{#if editingReceiptId === receipt.id && editingField === 'title'}
									<input
										type="text"
										bind:value={editValues.title}
										class="w-full md:w-80 px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary text-lg font-medium"
										placeholder="Shop name"
										on:blur={() => saveEditing(receipt)}
										on:keydown={(e) => e.key === 'Enter' && saveEditing(receipt)}
										autofocus
									/>
								{:else}
									<button
										on:click={() => startEditing(receipt, 'title')}
										class="text-lg font-semibold text-gray-900 hover:text-primary transition-colors text-left"
									>
										{receipt.title || 'Untitled Receipt'}
									</button>
								{/if}
							</div>

							<!-- Date and Total Row -->
							<div class="flex flex-wrap gap-6">
								<!-- Date -->
								<div>
									<label class="block text-xs font-medium text-gray-500 mb-1">Date</label>
									{#if editingReceiptId === receipt.id && editingField === 'date'}
										<input
											type="date"
											bind:value={editValues.receipt_date}
											class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
											on:blur={() => saveEditing(receipt)}
											on:keydown={(e) => e.key === 'Enter' && saveEditing(receipt)}
											autofocus
										/>
									{:else}
										<button
											on:click={() => startEditing(receipt, 'date')}
											class="text-sm text-gray-700 hover:text-primary transition-colors"
										>
											{formatDate(receipt.receipt_date)}
										</button>
									{/if}
								</div>

								<!-- Total -->
								<div>
									<label class="block text-xs font-medium text-gray-500 mb-1">Total</label>
									{#if editingReceiptId === receipt.id && editingField === 'total'}
										<div class="flex items-center gap-2">
											<span class="text-gray-500 text-sm">{receipt.currency}</span>
											<input
												type="number"
												bind:value={editValues.total}
												step="0.01"
												min="0"
												class="w-32 px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
												on:blur={() => saveEditing(receipt)}
												on:keydown={(e) => e.key === 'Enter' && saveEditing(receipt)}
												autofocus
											/>
										</div>
									{:else}
										<button
											on:click={() => startEditing(receipt, 'total')}
											class="text-sm font-semibold text-gray-900 hover:text-primary transition-colors"
										>
											{formatCurrency(receipt.total, receipt.currency)}
										</button>
									{/if}
								</div>
							</div>

							<!-- Tags -->
							<div>
								<label class="block text-xs font-medium text-gray-500 mb-1">Tags</label>
								{#if editingReceiptId === receipt.id && editingField === 'tags'}
									<div class="bg-white border border-gray-200 rounded-lg p-3 max-w-md">
										<div class="space-y-2 max-h-40 overflow-y-auto">
											{#each tags as tag}
												<label class="flex items-center gap-2 cursor-pointer hover:bg-gray-50 p-1 rounded">
													<input
														type="checkbox"
														checked={editValues.tags?.includes(tag.id)}
														on:change={() => toggleTag(tag.id)}
														class="rounded border-gray-300 text-primary focus:ring-primary"
													/>
													<span class="text-sm text-gray-700">{tag.name}</span>
												</label>
											{/each}
										</div>
										<div class="flex gap-2 mt-3 pt-3 border-t border-gray-200">
											<button
												on:click={() => saveEditing(receipt)}
												class="flex items-center gap-1 px-3 py-1.5 bg-primary text-white text-sm rounded-lg hover:bg-primary/90 transition-colors"
												disabled={isProcessing}
											>
												{#if isProcessing && processingId === receipt.id}
													<Loader2 class="w-4 h-4 animate-spin" />
												{:else}
													<Save class="w-4 h-4" />
												{/if}
												Save
											</button>
											<button
												on:click={cancelEditing}
												class="flex items-center gap-1 px-3 py-1.5 border border-gray-200 text-gray-700 text-sm rounded-lg hover:bg-gray-50 transition-colors"
											>
												<RotateCcw class="w-4 h-4" />
												Cancel
											</button>
										</div>
									</div>
								{:else}
									<button
										on:click={() => startEditing(receipt, 'tags')}
										class="flex flex-wrap gap-1 hover:bg-gray-50 p-1 rounded -ml-1 transition-colors"
									>
										{#if receipt.tags && receipt.tags.length > 0}
											{#each receipt.tags.slice(0, 5) as tagId}
												{@const tag = getTagById(tagId)}
												{#if tag}
													<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium {getTagColorClass(tag.color)}">
														{tag.name}
													</span>
												{/if}
											{/each}
											{#if receipt.tags.length > 5}
												<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-600">
													+{receipt.tags.length - 5}
												</span>
											{/if}
										{:else}
											<span class="text-sm text-gray-400 italic">Click to add tags...</span>
										{/if}
									</button>
								{/if}
							</div>
						</div>

						<!-- Actions -->
						<div class="flex lg:flex-col gap-2 lg:min-w-[140px]">
							<!-- Confirm Button -->
							<button
								on:click={() => handleConfirm(receipt.id)}
								disabled={isProcessing && processingId === receipt.id}
								class="flex items-center justify-center gap-2 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex-1 lg:flex-none"
							>
								{#if isProcessing && processingId === receipt.id}
									<Loader2 class="w-4 h-4 animate-spin" />
								{:else}
									<CheckCircle class="w-4 h-4" />
								{/if}
								<span class="text-sm font-medium">Confirm</span>
							</button>

							<!-- Reject Button -->
							<button
								on:click={() => handleReject(receipt.id)}
								disabled={isProcessing && processingId === receipt.id}
								class="flex items-center justify-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex-1 lg:flex-none"
							>
								{#if isProcessing && processingId === receipt.id}
									<Loader2 class="w-4 h-4 animate-spin" />
								{:else}
									<X class="w-4 h-4" />
								{/if}
								<span class="text-sm font-medium">Reject</span>
							</button>

							<!-- View Details Button -->
							<button
								on:click={() => handleViewDetails(receipt.id)}
								class="flex items-center justify-center gap-2 px-4 py-2 border border-gray-200 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors flex-1 lg:flex-none"
							>
								<Eye class="w-4 h-4" />
								<span class="text-sm font-medium hidden lg:inline">Details</span>
							</button>

							<!-- Re-run OCR Button -->
							<button
								on:click={() => handleRerunOCR(receipt.id)}
								disabled={isProcessing && processingId === receipt.id}
								class="flex items-center justify-center gap-2 px-4 py-2 border border-gray-200 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex-1 lg:flex-none"
								title="Re-run OCR"
							>
								<RefreshCw class="w-4 h-4" />
								<span class="text-sm font-medium hidden lg:inline">OCR</span>
							</button>

							<!-- Delete Button -->
							{#if deleteConfirmId === receipt.id}
								<div class="flex gap-1 flex-1 lg:flex-none">
									<button
										on:click={() => handleDelete(receipt.id)}
										disabled={isProcessing && processingId === receipt.id}
										class="flex items-center justify-center gap-1 px-3 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50 flex-1"
									>
										{#if isProcessing && processingId === receipt.id}
											<Loader2 class="w-4 h-4 animate-spin" />
										{:else}
											<CheckCircle class="w-4 h-4" />
										{/if}
									</button>
									<button
										on:click={() => (deleteConfirmId = null)}
										class="flex items-center justify-center gap-1 px-3 py-2 border border-gray-200 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors flex-1"
									>
										<X class="w-4 h-4" />
									</button>
								</div>
							{:else}
								<button
									on:click={() => (deleteConfirmId = receipt.id)}
									disabled={isProcessing}
									class="flex items-center justify-center gap-2 px-4 py-2 border border-gray-200 text-gray-700 rounded-lg hover:bg-red-50 hover:text-red-600 hover:border-red-200 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex-1 lg:flex-none"
									title="Delete"
								>
									<Trash2 class="w-4 h-4" />
									<span class="text-sm font-medium hidden lg:inline">Delete</span>
								</button>
							{/if}
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
