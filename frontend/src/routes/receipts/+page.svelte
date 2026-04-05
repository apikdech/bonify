<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { api, type Receipt, type Tag } from '$lib/api';
	import { toastStore } from '$lib/stores';
	import {
		Search,
		Filter,
		Plus,
		ChevronLeft,
		ChevronRight,
		Receipt as ReceiptIcon,
		AlertCircle,
		RefreshCw,
		ArrowUpDown,
		ArrowUp,
		ArrowDown,
		X,
		Calendar,
		Smartphone,
		MessageCircle,
		FileText,
		Bot
	} from 'lucide-svelte';

	// ============ State ============
	
	// Data
	let receipts: Receipt[] = [];
	let tags: Tag[] = [];
	let totalCount = 0;
	
	// Loading and error states
	let isLoading = true;
	let isFetchingTags = true;
	let error: string | null = null;
	
	// Search and filters
	let searchQuery = '';
	let selectedStatus: string = 'all';
	let selectedTag: string = 'all';
	let selectedSource: string = 'all';
	let fromDate = '';
	let toDate = '';
	
	// Pagination
	let currentPage = 1;
	let itemsPerPage = 20;
	const itemsPerPageOptions = [10, 20, 50];
	
	// Sorting
	let sortBy: 'receipt_date' | 'total' | 'created_at' = 'created_at';
	let sortOrder: 'asc' | 'desc' = 'desc';
	
	// UI state
	let showFilters = false;
	let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null;
	
	// ============ Constants ============
	
	const statusOptions = [
		{ value: 'all', label: 'All Statuses' },
		{ value: 'confirmed', label: 'Confirmed' },
		{ value: 'pending_review', label: 'Pending Review' },
		{ value: 'rejected', label: 'Rejected' }
	];
	
	const sourceOptions = [
		{ value: 'all', label: 'All Sources' },
		{ value: 'manual', label: 'Manual', icon: FileText },
		{ value: 'telegram', label: 'Telegram', icon: MessageCircle },
		{ value: 'discord', label: 'Discord', icon: Smartphone },
		{ value: 'ocr', label: 'OCR', icon: Bot }
	];
	
	// ============ Computed ============
	
	$: totalPages = Math.ceil(totalCount / itemsPerPage);
	$: startItem = totalCount === 0 ? 0 : (currentPage - 1) * itemsPerPage + 1;
	$: endItem = Math.min(currentPage * itemsPerPage, totalCount);
	$: hasActiveFilters = selectedStatus !== 'all' || selectedTag !== 'all' || 
	                      selectedSource !== 'all' || fromDate || toDate || searchQuery;
	
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
	
	function getStatusColor(status: string): string {
		switch (status) {
			case 'confirmed':
				return 'bg-green-100 text-green-800 border-green-200';
			case 'pending_review':
				return 'bg-yellow-100 text-yellow-800 border-yellow-200';
			case 'rejected':
				return 'bg-red-100 text-red-800 border-red-200';
			default:
				return 'bg-gray-100 text-gray-800 border-gray-200';
		}
	}
	
	function getStatusLabel(status: string): string {
		switch (status) {
			case 'confirmed':
				return 'Confirmed';
			case 'pending_review':
				return 'Pending Review';
			case 'rejected':
				return 'Rejected';
			default:
				return status;
		}
	}
	
	function getSourceIcon(source: string) {
		switch (source) {
			case 'telegram':
				return MessageCircle;
			case 'discord':
				return Smartphone;
			case 'manual':
				return FileText;
			case 'ocr':
				return Bot;
			default:
				return FileText;
		}
	}
	
	function getTagById(tagId: string): Tag | undefined {
		return tags.find(t => t.id === tagId);
	}
	
	function getTagColorClass(color: string | undefined): string {
		if (!color) return 'bg-gray-100 text-gray-700';
		// Map hex colors to Tailwind classes
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
	
	// ============ Data Fetching ============
	
	async function fetchTags() {
		isFetchingTags = true;
		try {
			tags = await api.tags.list();
		} catch (err) {
			console.error('Failed to fetch tags:', err);
		} finally {
			isFetchingTags = false;
		}
	}
	
	async function fetchReceipts() {
		isLoading = true;
		error = null;
		
		try {
			const params: Parameters<typeof api.receipts.list>[0] = {
				page: currentPage,
				limit: itemsPerPage,
				sort_by: sortBy,
				sort_order: sortOrder
			};
			
			// Add filters
			if (searchQuery) {
				params.q = searchQuery;
			}
			
			if (selectedStatus !== 'all') {
				params.status = selectedStatus as 'confirmed' | 'pending_review' | 'rejected';
			}
			
			if (selectedTag !== 'all') {
				params.tag_id = selectedTag;
			}
			
			if (selectedSource !== 'all') {
				params.source = selectedSource as 'manual' | 'telegram' | 'discord' | 'ocr' | 'api';
			}
			
			if (fromDate) {
				params.from = fromDate;
			}
			
			if (toDate) {
				params.to = toDate;
			}
			
			const response = await api.receipts.list(params);
			receipts = response.data;
			totalCount = response.total;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load receipts';
			toastStore.error('Failed to load receipts');
		} finally {
			isLoading = false;
		}
	}
	
	// ============ Event Handlers ============
	
	function handleSearchInput(event: Event) {
		const target = event.target as HTMLInputElement;
		searchQuery = target.value;
		
		// Debounce search
		if (searchDebounceTimer) {
			clearTimeout(searchDebounceTimer);
		}
		
		searchDebounceTimer = setTimeout(() => {
			currentPage = 1;
			updateURL();
			fetchReceipts();
		}, 300);
	}
	
	function handleSort(column: 'receipt_date' | 'total' | 'created_at') {
		if (sortBy === column) {
			sortOrder = sortOrder === 'asc' ? 'desc' : 'asc';
		} else {
			sortBy = column;
			sortOrder = 'desc';
		}
		updateURL();
		fetchReceipts();
	}
	
	function handlePageChange(newPage: number) {
		if (newPage < 1 || newPage > totalPages) return;
		currentPage = newPage;
		updateURL();
		fetchReceipts();
		// Scroll to top of table
		document.getElementById('receipts-table')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
	}
	
	function handleItemsPerPageChange(event: Event) {
		const target = event.target as HTMLSelectElement;
		itemsPerPage = parseInt(target.value);
		currentPage = 1;
		updateURL();
		fetchReceipts();
	}
	
	function applyFilters() {
		currentPage = 1;
		updateURL();
		fetchReceipts();
	}
	
	function clearFilters() {
		searchQuery = '';
		selectedStatus = 'all';
		selectedTag = 'all';
		selectedSource = 'all';
		fromDate = '';
		toDate = '';
		currentPage = 1;
		updateURL();
		fetchReceipts();
	}
	
	function handleRowClick(receiptId: string) {
		goto(`/receipts/${receiptId}`);
	}
	
	// ============ URL Sync ============
	
	function updateURL() {
		const url = new URL($page.url);
		
		// Update query params
		if (searchQuery) {
			url.searchParams.set('q', searchQuery);
		} else {
			url.searchParams.delete('q');
		}
		
		if (selectedStatus !== 'all') {
			url.searchParams.set('status', selectedStatus);
		} else {
			url.searchParams.delete('status');
		}
		
		if (selectedTag !== 'all') {
			url.searchParams.set('tag', selectedTag);
		} else {
			url.searchParams.delete('tag');
		}
		
		if (selectedSource !== 'all') {
			url.searchParams.set('source', selectedSource);
		} else {
			url.searchParams.delete('source');
		}
		
		if (fromDate) {
			url.searchParams.set('from', fromDate);
		} else {
			url.searchParams.delete('from');
		}
		
		if (toDate) {
			url.searchParams.set('to', toDate);
		} else {
			url.searchParams.delete('to');
		}
		
		if (currentPage > 1) {
			url.searchParams.set('page', String(currentPage));
		} else {
			url.searchParams.delete('page');
		}
		
		if (itemsPerPage !== 20) {
			url.searchParams.set('limit', String(itemsPerPage));
		} else {
			url.searchParams.delete('limit');
		}
		
		if (sortBy !== 'created_at') {
			url.searchParams.set('sort_by', sortBy);
		} else {
			url.searchParams.delete('sort_by');
		}
		
		if (sortOrder !== 'desc') {
			url.searchParams.set('sort_order', sortOrder);
		} else {
			url.searchParams.delete('sort_order');
		}
		
		goto(url.toString(), { replaceState: true, keepFocus: true });
	}
	
	function loadFiltersFromURL() {
		const url = $page.url;
		
		searchQuery = url.searchParams.get('q') || '';
		selectedStatus = url.searchParams.get('status') || 'all';
		selectedTag = url.searchParams.get('tag') || 'all';
		selectedSource = url.searchParams.get('source') || 'all';
		fromDate = url.searchParams.get('from') || '';
		toDate = url.searchParams.get('to') || '';
		currentPage = parseInt(url.searchParams.get('page') || '1');
		itemsPerPage = parseInt(url.searchParams.get('limit') || '20');
		sortBy = (url.searchParams.get('sort_by') as typeof sortBy) || 'created_at';
		sortOrder = (url.searchParams.get('sort_order') as typeof sortOrder) || 'desc';
	}
	
	// ============ Lifecycle ============
	
	onMount(() => {
		loadFiltersFromURL();
		fetchTags();
		fetchReceipts();
	});
</script>

<div class="p-4 md:p-8 max-w-7xl mx-auto">
	<!-- Header -->
	<div class="mb-8 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
		<div>
			<h1 class="text-3xl font-bold text-gray-900 mb-2">Receipts</h1>
			<p class="text-gray-600">Manage and review your receipts</p>
		</div>
		<button
			on:click={() => goto('/receipts/new')}
			class="flex items-center gap-2 px-6 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors shadow-sm self-start"
		>
			<Plus class="w-5 h-5" />
			<span class="font-medium">Add Receipt</span>
		</button>
	</div>
	
	<!-- Error State -->
	{#if error}
		<div class="bg-red-50 border border-red-200 rounded-lg p-4 mb-6 flex items-center justify-between">
			<div class="flex items-center gap-3">
				<AlertCircle class="w-5 h-5 text-red-600" />
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
	
	<!-- Search and Filter Bar -->
	<div class="mb-6 space-y-4">
		<div class="flex flex-col sm:flex-row gap-4">
			<!-- Search Input -->
			<div class="relative flex-1">
				<Search class="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
				<input
					type="text"
					placeholder="Search receipts..."
					value={searchQuery}
					on:input={handleSearchInput}
					class="w-full pl-10 pr-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-colors"
				/>
				{#if searchQuery}
					<button
						on:click={() => { searchQuery = ''; currentPage = 1; updateURL(); fetchReceipts(); }}
						class="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600"
					>
						<X class="w-4 h-4" />
					</button>
				{/if}
			</div>
			
			<!-- Filter Toggle Button -->
			<button
				on:click={() => showFilters = !showFilters}
				class="flex items-center gap-2 px-4 py-3 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors {hasActiveFilters ? 'border-primary text-primary' : 'text-gray-700'}"
			>
				<Filter class="w-5 h-5" />
				<span class="font-medium">Filters</span>
				{#if hasActiveFilters}
					<span class="bg-primary text-white text-xs font-bold px-2 py-0.5 rounded-full">
						{[selectedStatus !== 'all', selectedTag !== 'all', selectedSource !== 'all', fromDate, toDate].filter(Boolean).length}
					</span>
				{/if}
			</button>
		</div>
		
		<!-- Filters Panel -->
		{#if showFilters}
			<div class="bg-gray-50 border border-gray-200 rounded-lg p-4 space-y-4">
				<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
					<!-- Status Filter -->
					<div>
						<label for="status-filter" class="block text-sm font-medium text-gray-700 mb-2">Status</label>
						<select
							id="status-filter"
							bind:value={selectedStatus}
							on:change={applyFilters}
							class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
						>
							{#each statusOptions as option}
								<option value={option.value}>{option.label}</option>
							{/each}
						</select>
					</div>
					
					<!-- Tag Filter -->
					<div>
						<label for="tag-filter" class="block text-sm font-medium text-gray-700 mb-2">Tag</label>
						<select
							id="tag-filter"
							bind:value={selectedTag}
							on:change={applyFilters}
							disabled={isFetchingTags}
							class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary disabled:bg-gray-100"
						>
							<option value="all">All Tags</option>
							{#each tags as tag}
								<option value={tag.id}>{tag.name}</option>
							{/each}
						</select>
					</div>
					
					<!-- Source Filter -->
					<div>
						<label for="source-filter" class="block text-sm font-medium text-gray-700 mb-2">Source</label>
						<select
							id="source-filter"
							bind:value={selectedSource}
							on:change={applyFilters}
							class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
						>
							{#each sourceOptions as option}
								<option value={option.value}>{option.label}</option>
							{/each}
						</select>
					</div>
					
					<!-- Items Per Page -->
					<div>
						<label for="per-page" class="block text-sm font-medium text-gray-700 mb-2">Per Page</label>
						<select
							id="per-page"
							value={itemsPerPage}
							on:change={handleItemsPerPageChange}
							class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
						>
							{#each itemsPerPageOptions as option}
								<option value={option}>{option} items</option>
							{/each}
						</select>
					</div>
				</div>
				
				<!-- Date Range -->
				<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
					<div>
						<label class="block text-sm font-medium text-gray-700 mb-2 flex items-center gap-2">
							<Calendar class="w-4 h-4" />
							From Date
						</label>
						<input
							type="date"
							bind:value={fromDate}
							on:change={applyFilters}
							class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
						/>
					</div>
					<div>
						<label class="block text-sm font-medium text-gray-700 mb-2 flex items-center gap-2">
							<Calendar class="w-4 h-4" />
							To Date
						</label>
						<input
							type="date"
							bind:value={toDate}
							on:change={applyFilters}
							class="w-full px-3 py-2 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
						/>
					</div>
				</div>
				
				<!-- Clear Filters -->
				{#if hasActiveFilters}
					<div class="flex justify-end">
						<button
							on:click={clearFilters}
							class="flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 transition-colors"
						>
							<X class="w-4 h-4" />
							Clear all filters
						</button>
					</div>
				{/if}
			</div>
		{/if}
	</div>
	
	<!-- Results Count -->
	<div class="mb-4 text-sm text-gray-600">
		{#if isLoading}
			<span>Loading receipts...</span>
		{:else}
			<span>Showing {startItem}-{endItem} of {totalCount} receipts</span>
		{/if}
	</div>
	
	<!-- Receipts Table -->
	<div id="receipts-table" class="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
		{#if isLoading && receipts.length === 0}
			<!-- Loading Skeleton -->
			<div class="divide-y divide-gray-100">
				{#each Array(5) as _}
					<div class="p-4 flex items-center gap-4">
						<div class="w-10 h-10 bg-gray-200 rounded-lg animate-pulse"></div>
						<div class="flex-1 space-y-2">
							<div class="h-4 bg-gray-200 rounded w-1/4 animate-pulse"></div>
							<div class="h-3 bg-gray-200 rounded w-1/6 animate-pulse"></div>
						</div>
						<div class="h-4 bg-gray-200 rounded w-20 animate-pulse"></div>
						<div class="h-4 bg-gray-200 rounded w-16 animate-pulse"></div>
					</div>
				{/each}
			</div>
		{:else if receipts.length === 0}
			<!-- Empty State -->
			<div class="p-12 text-center">
				<ReceiptIcon class="w-16 h-16 text-gray-300 mx-auto mb-4" />
				<h3 class="text-lg font-medium text-gray-900 mb-2">
					{hasActiveFilters ? 'No receipts match your filters' : 'No receipts yet'}
				</h3>
				<p class="text-gray-500 mb-6">
					{hasActiveFilters 
						? 'Try adjusting your search or filters to find what you\'re looking for.' 
						: 'Start by adding your first receipt to track your expenses.'}
				</p>
				{#if hasActiveFilters}
					<button
						on:click={clearFilters}
						class="text-primary hover:text-primary/80 font-medium transition-colors"
					>
						Clear filters
					</button>
				{:else}
					<button
						on:click={() => goto('/receipts/new')}
						class="inline-flex items-center gap-2 px-6 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
					>
						<Plus class="w-5 h-5" />
						Add Receipt
					</button>
				{/if}
			</div>
		{:else}
			<!-- Table Header -->
			<div class="hidden md:grid grid-cols-12 gap-4 px-6 py-3 bg-gray-50 border-b border-gray-200 text-sm font-medium text-gray-700">
				<button
					type="button"
					class="col-span-4 text-left flex items-center gap-1 hover:text-gray-900 cursor-pointer"
					on:click={() => handleSort('created_at')}
				>
					Shop Name
					{#if sortBy === 'created_at'}
						{#if sortOrder === 'asc'}
							<ArrowUp class="w-4 h-4" />
						{:else}
							<ArrowDown class="w-4 h-4" />
						{/if}
					{:else}
						<ArrowUpDown class="w-4 h-4 text-gray-400" />
					{/if}
				</button>
				<button
					type="button"
					class="col-span-2 text-left flex items-center gap-1 hover:text-gray-900 cursor-pointer"
					on:click={() => handleSort('receipt_date')}
				>
					Date
					{#if sortBy === 'receipt_date'}
						{#if sortOrder === 'asc'}
							<ArrowUp class="w-4 h-4" />
						{:else}
							<ArrowDown class="w-4 h-4" />
						{/if}
					{:else}
						<ArrowUpDown class="w-4 h-4 text-gray-400" />
					{/if}
				</button>
				<button
					type="button"
					class="col-span-2 text-left flex items-center gap-1 hover:text-gray-900 cursor-pointer"
					on:click={() => handleSort('total')}
				>
					Total
					{#if sortBy === 'total'}
						{#if sortOrder === 'asc'}
							<ArrowUp class="w-4 h-4" />
						{:else}
							<ArrowDown class="w-4 h-4" />
						{/if}
					{:else}
						<ArrowUpDown class="w-4 h-4 text-gray-400" />
					{/if}
				</button>
				<div class="col-span-2">Tags</div>
				<div class="col-span-2 text-right">Status</div>
			</div>
			
			<!-- Table Body -->
			<div class="divide-y divide-gray-100">
				{#each receipts as receipt}
					<button
						on:click={() => handleRowClick(receipt.id)}
						class="w-full grid grid-cols-1 md:grid-cols-12 gap-2 md:gap-4 px-4 md:px-6 py-4 hover:bg-gray-50 transition-colors text-left items-start md:items-center"
					>
						<!-- Shop Name & Source -->
						<div class="col-span-1 md:col-span-4 flex items-center gap-3">
							<div class="w-10 h-10 rounded-lg bg-gray-100 flex items-center justify-center flex-shrink-0">
								<svelte:component this={getSourceIcon(receipt.source)} class="w-5 h-5 text-gray-500" />
							</div>
							<div class="min-w-0">
								<div class="font-medium text-gray-900 truncate">{receipt.title || 'Untitled Receipt'}</div>
								<div class="text-sm text-gray-500 md:hidden">
									{formatDate(receipt.receipt_date || receipt.created_at)} • {formatCurrency(receipt.total, receipt.currency)}
								</div>
							</div>
						</div>
						
						<!-- Date -->
						<div class="hidden md:block col-span-2 text-sm text-gray-600">
							{formatDate(receipt.receipt_date || receipt.created_at)}
						</div>
						
						<!-- Total -->
						<div class="hidden md:block col-span-2">
							<span class="font-medium text-gray-900">
								{formatCurrency(receipt.total, receipt.currency)}
							</span>
						</div>
						
						<!-- Tags -->
						<div class="col-span-1 md:col-span-2 flex flex-wrap gap-1">
							{#if receipt.tags && receipt.tags.length > 0}
								{#each receipt.tags.slice(0, 2) as tagId}
									{@const tag = getTagById(tagId)}
									{#if tag}
										<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium {getTagColorClass(tag.color)}">
											{tag.name}
										</span>
									{/if}
								{/each}
								{#if receipt.tags.length > 2}
									<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-600">
										+{receipt.tags.length - 2}
									</span>
								{/if}
							{:else}
								<span class="text-gray-400 text-sm">-</span>
							{/if}
						</div>
						
						<!-- Status -->
						<div class="col-span-1 md:col-span-2 flex justify-end">
							<span class="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium border {getStatusColor(receipt.status)}">
								{getStatusLabel(receipt.status)}
							</span>
						</div>
					</button>
				{/each}
			</div>
			
			<!-- Pagination -->
			{#if totalPages > 1}
				<div class="px-6 py-4 border-t border-gray-200 flex items-center justify-between">
					<div class="flex items-center gap-2">
						<button
							on:click={() => handlePageChange(currentPage - 1)}
							disabled={currentPage === 1}
							class="flex items-center gap-1 px-3 py-2 border border-gray-200 rounded-lg text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
						>
							<ChevronLeft class="w-4 h-4" />
							Previous
						</button>
						
						<div class="hidden sm:flex items-center gap-1">
							{#each Array(totalPages) as _, i}
								{@const pageNum = i + 1}
								{#if pageNum === 1 || pageNum === totalPages || (pageNum >= currentPage - 1 && pageNum <= currentPage + 1)}
									<button
										on:click={() => handlePageChange(pageNum)}
										class="w-10 h-10 rounded-lg text-sm font-medium transition-colors {pageNum === currentPage ? 'bg-primary text-white' : 'text-gray-700 hover:bg-gray-100'}"
									>
										{pageNum}
									</button>
								{:else if pageNum === currentPage - 2 || pageNum === currentPage + 2}
									<span class="w-10 h-10 flex items-center justify-center text-gray-400">...</span>
								{/if}
							{/each}
						</div>
						
						<button
							on:click={() => handlePageChange(currentPage + 1)}
							disabled={currentPage === totalPages}
							class="flex items-center gap-1 px-3 py-2 border border-gray-200 rounded-lg text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
						>
							Next
							<ChevronRight class="w-4 h-4" />
						</button>
					</div>
					
					<div class="hidden md:block text-sm text-gray-600">
						Page {currentPage} of {totalPages}
					</div>
				</div>
			{/if}
		{/if}
	</div>
</div>
