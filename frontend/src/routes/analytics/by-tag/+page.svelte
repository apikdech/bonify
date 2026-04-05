<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Chart, ArcElement, Tooltip, Legend, Title } from 'chart.js';
	import { analyticsDateRangeStore } from '$lib/stores';
	import api from '$lib/api';
	import type { ByTagResponse, TagSpend } from '$lib/api';
	
	Chart.register(ArcElement, Tooltip, Legend, Title);
	
	let canvas: HTMLCanvasElement;
	let chart: Chart | null = null;
	let data: TagSpend[] = [];
	let loading = true;
	let error: string | null = null;
	let warnings: string[] = [];
	let untaggedTotal = 0;
	
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
			
			const response: ByTagResponse = await api.analytics.byTag(params);
			data = response.data || [];
			
			if (response.warnings) {
				warnings = response.warnings.map(w => w.message);
			}
			
			renderChart();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load tag analytics';
		} finally {
			loading = false;
		}
	}
	
	function renderChart() {
		if (!canvas) return;
		
		if (chart) {
			chart.destroy();
		}
		
		if (data.length === 0) return;
		
		const labels = data.map(d => d.name);
		const values = data.map(d => d.total);
		const colors = data.map(d => d.color || getDefaultColor(d.name));
		
		const ctx = canvas.getContext('2d');
		if (!ctx) return;
		
		chart = new Chart(ctx, {
			type: 'doughnut',
			data: {
				labels,
				datasets: [{
					data: values,
					backgroundColor: colors,
					borderWidth: 2,
					borderColor: '#ffffff'
				}]
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				plugins: {
					legend: {
						display: false // We'll show custom legend
					},
					tooltip: {
						callbacks: {
							label: (context) => {
								const value = context.raw as number;
								const label = context.label || '';
								const percentage = data[context.dataIndex]?.percentage || 0;
								return `${label}: $${value.toFixed(2)} (${percentage.toFixed(1)}%)`;
							}
						}
					}
				},
				onClick: (event, elements) => {
					if (elements.length > 0) {
						const index = elements[0].index;
						const tagData = data[index];
						if (tagData) {
							goto(`/receipts?tag_id=${tagData.tag_id}`);
						}
					}
				}
			}
		});
	}
	
	function getDefaultColor(name: string): string {
		// Generate a consistent color from the name
		const colors = [
			'#3b82f6', '#ef4444', '#10b981', '#f59e0b', '#8b5cf6',
			'#ec4899', '#06b6d4', '#f97316', '#84cc16', '#6366f1'
		];
		let hash = 0;
		for (let i = 0; i < name.length; i++) {
			hash = name.charCodeAt(i) + ((hash << 5) - hash);
		}
		return colors[Math.abs(hash) % colors.length];
	}
	
	function formatCurrency(value: number): string {
		return new Intl.NumberFormat('en-US', {
			style: 'currency',
			currency: 'USD'
		}).format(value);
	}
	
	$: totalSpend = data.reduce((sum, d) => sum + d.total, 0);
	
	onMount(() => {
		loadData();
		
		const unsubscribe = analyticsDateRangeStore.subscribe(() => {
			loadData();
		});
		
		return () => {
			unsubscribe();
			if (chart) chart.destroy();
		};
	});
</script>

<div class="by-tag">
	<header class="page-header">
		<h2>Spending by Tag</h2>
	</header>
	
	{#if loading}
		<div class="loading-state">
			<div class="spinner"></div>
			<p>Loading tag breakdown...</p>
		</div>
	{:else if error}
		<div class="error-state">
			<p>{error}</p>
			<button on:click={loadData}>Try Again</button>
		</div>
	{:else if data.length === 0}
		<div class="empty-state">
			<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"></path>
				<line x1="7" y1="7" x2="7.01" y2="7"></line>
			</svg>
			<h3>No tagged receipts</h3>
			<p>Start tagging your receipts to see spending breakdown by category.</p>
			<a href="/tags">Manage Tags</a>
		</div>
	{:else}
		{#if warnings.length > 0}
			<div class="warnings">
				{#each warnings as warning}
					<div class="warning">{warning}</div>
				{/each}
			</div>
		{/if}
		
		<div class="chart-and-legend">
			<div class="chart-container">
				<canvas bind:this={canvas} height="250"></canvas>
			</div>
			
			<div class="custom-legend">
				{#each data as tag}
					<div
						class="legend-item"
						on:click={() => goto(`/receipts?tag_id=${tag.tag_id}`)}
					>
						<div class="legend-color" style="background-color: {tag.color || getDefaultColor(tag.name)}"></div>
						<div class="legend-info">
							<span class="legend-name">{tag.name}</span>
							<span class="legend-value">{formatCurrency(tag.total)} ({tag.percentage.toFixed(1)}%)</span>
						</div>
					</div>
				{/each}
			</div>
		</div>
		
		<div class="data-table">
			<h3>Tag Breakdown</h3>
			<table>
				<thead>
					<tr>
						<th>Tag</th>
						<th>Receipts</th>
						<th>Total</th>
						<th>% of Total</th>
					</tr>
				</thead>
				<tbody>
					{#each data as tag}
						<tr on:click={() => goto(`/receipts?tag_id=${tag.tag_id}`)} class="clickable">
							<td>
								<span class="tag-badge" style="background-color: {tag.color || getDefaultColor(tag.name)}">
									{tag.name}
								</span>
							</td>
							<td>{tag.count}</td>
							<td>{formatCurrency(tag.total)}</td>
							<td>{tag.percentage.toFixed(1)}%</td>
						</tr>
					{/each}
				</tbody>
				<tfoot>
					<tr>
						<td><strong>Total</strong></td>
						<td><strong>{data.reduce((sum, t) => sum + t.count, 0)}</strong></td>
						<td><strong>{formatCurrency(totalSpend)}</strong></td>
						<td><strong>100%</strong></td>
					</tr>
				</tfoot>
			</table>
			<p class="table-hint">Click on any tag to view receipts with that tag</p>
		</div>
	{/if}
</div>

<style>
	.by-tag {
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
	
	.chart-and-legend {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 24px;
		margin-bottom: 32px;
	}
	
	.chart-container {
		position: relative;
		height: 250px;
	}
	
	.custom-legend {
		display: flex;
		flex-direction: column;
		gap: 8px;
		max-height: 250px;
		overflow-y: auto;
	}
	
	.legend-item {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px;
		border-radius: 6px;
		cursor: pointer;
		transition: background 0.2s;
	}
	
	.legend-item:hover {
		background: #f3f4f6;
	}
	
	.legend-color {
		width: 16px;
		height: 16px;
		border-radius: 4px;
		flex-shrink: 0;
	}
	
	.legend-info {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}
	
	.legend-name {
		font-weight: 500;
		color: #374151;
		font-size: 14px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	
	.legend-value {
		font-size: 12px;
		color: #6b7280;
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
		margin-top: 32px;
	}
	
	.data-table h3 {
		font-size: 16px;
		font-weight: 600;
		color: #374151;
		margin: 0 0 16px 0;
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
	}
	
	tbody tr {
		cursor: pointer;
		transition: background 0.2s;
	}
	
	tbody tr:hover {
		background: #f3f4f6;
	}
	
	tfoot td {
		font-weight: 600;
		border-top: 2px solid #e5e7eb;
		border-bottom: none;
	}
	
	.tag-badge {
		padding: 4px 8px;
		border-radius: 4px;
		font-size: 12px;
		font-weight: 500;
		color: white;
		white-space: nowrap;
	}
	
	.table-hint {
		font-size: 12px;
		color: #6b7280;
		margin-top: 8px;
	}
	
	@media (max-width: 768px) {
		.chart-and-legend {
			grid-template-columns: 1fr;
		}
		
		.custom-legend {
			max-height: 200px;
		}
	}
	
	@media (max-width: 640px) {
		table {
			font-size: 13px;
		}
		
		th, td {
			padding: 8px;
		}
	}
</style>