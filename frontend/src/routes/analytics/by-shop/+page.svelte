<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { analyticsDateRangeStore } from '$lib/stores';
	import api from '$lib/api';
	import type { ByShopResponse, ShopSpend } from '$lib/api';
	
	let data: ShopSpend[] = [];
	let loading = true;
	let error: string | null = null;
	let warnings: string[] = [];
	let searchQuery = '';
	let sortColumn: 'name' | 'total' | 'visit_count' | 'avg_ticket' | 'last_visit' = 'total';
	let sortOrder: 'asc' | 'desc' = 'desc';
	let currentPage = 1;
	const itemsPerPage = 10;
	
	async function loadData() {
		loading = true;
		error = null;
		warnings = [];
		
		try {
			const { from, to } = $analyticsDateRangeStore;
			const params = {
				from: from.toISOString().split('T')[0],
				to: to.toISOString().split('T')[0]
			};
			
			const response: ByShopResponse = await api.analytics.byShop(params);
			data = response.data || [];
			
			if (response.warnings) {
				warnings = response.warnings.map(w => w.message);
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load shop analytics';
		} finally {
			loading = false;
		}
	}
	
	function handleSort(column: typeof sortColumn) {
		if (sortColumn === column) {
			sortOrder = sortOrder === 'asc' ? 'desc' : 'asc';
		} else {
			sortColumn = column;
			sortOrder = 'desc';
		}
	}
	
	function formatCurrency(value: number): string {
		return new Intl.NumberFormat('en-US', {
			style: 'currency',
			currency: 'USD'
		}).format(value);
	}
	
	function formatDate(dateStr: string): string {
		const date = new Date(dateStr);
		return date.toLocaleDateString('en-US', {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	}
	
	$: filteredData = data.filter(shop => 
		shop.name.toLowerCase().includes(searchQuery.toLowerCase())
	);
	
	$: sortedData = [...filteredData].sort((a, b) => {
		let comparison = 0;
		switch (sortColumn) {
			case 'name':
				comparison = a.name.localeCompare(b.name);
				break;
			case 'total':
				comparison = a.total - b.total;
				break;
			case 'visit_count':
				comparison = a.visit_count - b.visit_count;
				break;
			case 'avg_ticket':
				comparison = a.avg_ticket - b.avg_ticket;
				break;
			case 'last_visit':
				comparison = new Date(a.last_visit).getTime() - new Date(b.last_visit).getTime();
				break;
		}
		return sortOrder === 'asc' ? comparison : -comparison;
	});
	
	$: totalPages = Math.ceil(sortedData.length / itemsPerPage);
	$: paginatedData = sortedData.slice(
		(currentPage - 1) * itemsPerPage,
		currentPage * itemsPerPage
	);
	
	$: totalSpend = data.reduce((sum, shop) => sum + shop.total, 0);
	$: totalVisits = data.reduce((sum, shop) => sum + shop.visit_count, 0);
	
	onMount(() => {
		loadData();
		
		const unsubscribe = analyticsDateRangeStore.subscribe(() => {
			loadData();
		});
		
		return () => {
			unsubscribe();
		};
	});
</script>

<div class="by-shop">
	<header class="page-header">
		<h2>Spending by Shop</h2>
	</header>
	
	{#if loading}
		<div class="loading-state">
			<div class="spinner"></div>
			<p>Loading shop breakdown...</p>
		</div>
	{:else if error}
		<div class="error-state">
			<p>{error}</p>
			<button on:click={loadData}>Try Again</button>
		</div>
	{:else if data.length === 0}
		<div class="empty-state">
			<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path>
				<polyline points="9 22 9 12 15 12 15 22"></polyline>
			</svg>
			<h3>No shop data</h3>
			<p>Start adding receipts to see spending breakdown by merchant.</p>
			<a href="/receipts/new">Add Receipt</a>
		</div>
	{:else}
		{#if warnings.length > 0}
			<div class="warnings">
				{#each warnings as warning}
					<div class="warning">{warning}</div>
				{/each}
			</div>
		{/if}
		
		<div class="summary-cards">
			<div class="summary-card">
				<span class="summary-label">Shops</span>
				<span class="summary-value">{data.length}</span>
			</div>
			<div class="summary-card">
				<span class="summary-label">Total Spent</span>
				<span class="summary-value">{formatCurrency(totalSpend)}</span>
			</div>
			<div class="summary-card">
				<span class="summary-label">Total Visits</span>
				<span class="summary-value">{totalVisits}</span>
			</div>
			<div class="summary-card">
				<span class="summary-label">Avg Ticket</span>
				<span class="summary-value">{formatCurrency(totalSpend / totalVisits)}</span>
			</div>
		</div>
		
		<div class="search-bar">
			<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<circle cx="11" cy="11" r="8"></circle>
				<line x1="21" y1="21" x2="16.65" y2="16.65"></line>
			</svg>
			<input
				type="text"
				placeholder="Search shops..."
				bind:value={searchQuery}
			/>
		</div>
		
		<div class="data-table">
			<table>
				<thead>
					<tr>
						<th class="sortable" on:click={() => handleSort('name')}>
							Shop
							{#if sortColumn === 'name'}
								<span class="sort-indicator">{sortOrder === 'asc' ? '↑' : '↓'}</span>
							{/if}
						</th>
						<th class="sortable" on:click={() => handleSort('total')}>
							Total Spent
							{#if sortColumn === 'total'}
								<span class="sort-indicator">{sortOrder === 'asc' ? '↑' : '↓'}</span>
							{/if}
						</th>
						<th class="sortable" on:click={() => handleSort('visit_count')}>
							Visits
							{#if sortColumn === 'visit_count'}
								<span class="sort-indicator">{sortOrder === 'asc' ? '↑' : '↓'}</span>
							{/if}
						</th>
						<th class="sortable" on:click={() => handleSort('avg_ticket')}>
							Avg Ticket
							{#if sortColumn === 'avg_ticket'}
								<span class="sort-indicator">{sortOrder === 'asc' ? '↑' : '↓'}</span>
							{/if}
						</th>
						<th class="sortable" on:click={() => handleSort('last_visit')}>
							Last Visit
							{#if sortColumn === 'last_visit'}
								<span class="sort-indicator">{sortOrder === 'asc' ? '↑' : '↓'}</span>
							{/if}
						</th>
					</tr>
				</thead>
				<tbody>
					{#each paginatedData as shop}
						<tr on:click={() => goto(`/receipts?q=${encodeURIComponent(shop.name)}`)} class="clickable">
							<td class="shop-name">{shop.name}</td>
							<td class="amount">{formatCurrency(shop.total)}</td>
							<td>{shop.visit_count}</td>
							<td class="amount">{formatCurrency(shop.avg_ticket)}</td>
							<td>{formatDate(shop.last_visit)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
			
			{#if totalPages > 1}
				<div class="pagination">
					<button
						disabled={currentPage === 1}
						on:click={() => currentPage--}
					>
						Previous
					</button>
					<span>Page {currentPage} of {totalPages}</span>
					<button
						disabled={currentPage === totalPages}
						on:click={() => currentPage++}
					>
						Next
					</button>
				</div>
			{/if}
			
			<p class="table-hint">Click on any shop to view receipts from that merchant</p>
		</div>
	{/if}
</div>

<style>
	.by-shop {
		padding: 0;
	}
	
	.page-header {
		margin-bottom: 24px;
	}
	
	.page-header h2 {
		font-size: 20px;
		font-weight: 600;
		color: #111827;
		margin: 0;
	}
	
	.summary-cards {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
		gap: 16px;
		margin-bottom: 24px;
	}
	
	.summary-card {
		background: #f9fafb;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		padding: 16px;
		text-align: center;
	}
	
	.summary-label {
		display: block;
		font-size: 12px;
		color: #6b7280;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		margin-bottom: 4px;
	}
	
	.summary-value {
		display: block;
		font-size: 20px;
		font-weight: 600;
		color: #111827;
	}
	
	.search-bar {
		display: flex;
		align-items: center;
		gap: 8px;
		background: #f9fafb;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		padding: 10px 12px;
		margin-bottom: 16px;
	}
	
	.search-bar svg {
		color: #9ca3af;
		flex-shrink: 0;
	}
	
	.search-bar input {
		flex: 1;
		border: none;
		background: transparent;
		font-size: 14px;
		color: #374151;
		outline: none;
	}
	
	.search-bar input::placeholder {
		color: #9ca3af;
	}
	
	.loading-state,
	.error-state,
	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 48px 24px;
		text-align: center;
	}
	
	.spinner {
		width: 32px;
		height: 32px;
		border: 3px solid #e5e7eb;
		border-top-color: #3b82f6;
		border-radius: 50%;
		animation: spin 1s linear infinite;
		margin-bottom: 12px;
	}
	
	@keyframes spin {
		to { transform: rotate(360deg); }
	}
	
	.error-state p {
		color: #dc2626;
		margin-bottom: 12px;
	}
	
	.error-state button {
		padding: 8px 16px;
		background: #3b82f6;
		color: white;
		border: none;
		border-radius: 6px;
		cursor: pointer;
		font-size: 14px;
	}
	
	.error-state button:hover {
		background: #2563eb;
	}
	
	.empty-state {
		color: #6b7280;
	}
	
	.empty-state svg {
		color: #d1d5db;
		margin-bottom: 16px;
	}
	
	.empty-state h3 {
		font-size: 18px;
		font-weight: 600;
		color: #374151;
		margin: 0 0 8px 0;
	}
	
	.empty-state p {
		margin: 0 0 16px 0;
	}
	
	.empty-state a {
		color: #3b82f6;
		text-decoration: none;
		font-weight: 500;
	}
	
	.empty-state a:hover {
		text-decoration: underline;
	}
	
	.warnings {
		margin-bottom: 16px;
	}
	
	.warning {
		background: #fef3c7;
		border: 1px solid #f59e0b;
		color: #92400e;
		padding: 8px 12px;
		border-radius: 6px;
		font-size: 14px;
		margin-bottom: 8px;
	}
	
	.data-table {
		overflow-x: auto;
	}
	
	table {
		width: 100%;
		border-collapse: collapse;
		font-size: 14px;
	}
	
	th, td {
		padding: 12px;
		text-align: left;
		border-bottom: 1px solid #e5e7eb;
	}
	
	th {
		font-weight: 600;
		color: #6b7280;
		background: #f9fafb;
		white-space: nowrap;
	}
	
	th.sortable {
		cursor: pointer;
		user-select: none;
	}
	
	th.sortable:hover {
		background: #f3f4f6;
	}
	
	.sort-indicator {
		margin-left: 4px;
		color: #3b82f6;
	}
	
	tbody tr {
		cursor: pointer;
		transition: background 0.2s;
	}
	
	tbody tr:hover {
		background: #f3f4f6;
	}
	
	.shop-name {
		font-weight: 500;
		color: #374151;
	}
	
	.amount {
		text-align: right;
		font-variant-numeric: tabular-nums;
	}
	
	.pagination {
		display: flex;
		justify-content: center;
		align-items: center;
		gap: 16px;
		margin-top: 16px;
		padding-top: 16px;
		border-top: 1px solid #e5e7eb;
	}
	
	.pagination button {
		padding: 8px 16px;
		background: white;
		border: 1px solid #d1d5db;
		border-radius: 6px;
		cursor: pointer;
		font-size: 14px;
		color: #374151;
	}
	
	.pagination button:hover:not(:disabled) {
		background: #f3f4f6;
	}
	
	.pagination button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
	
	.pagination span {
		font-size: 14px;
		color: #6b7280;
	}
	
	.table-hint {
		font-size: 12px;
		color: #6b7280;
		margin-top: 8px;
		text-align: center;
	}
	
	@media (max-width: 640px) {
		table {
			font-size: 13px;
		}
		
		th, td {
			padding: 8px;
		}
		
		.summary-cards {
			grid-template-columns: repeat(2, 1fr);
		}
	}
</style>