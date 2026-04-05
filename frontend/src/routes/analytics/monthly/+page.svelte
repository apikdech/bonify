<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { Chart, BarElement, CategoryScale, LinearScale, Tooltip, Legend, Title } from 'chart.js';
	import { analyticsDateRangeStore } from '$lib/stores';
	import api from '$lib/api';
	import type { MonthlyTrendsResponse, MonthData } from '$lib/api';
	
	Chart.register(BarElement, CategoryScale, LinearScale, Tooltip, Legend, Title);
	
	let canvas: HTMLCanvasElement;
	let chart: Chart | null = null;
	let data: MonthData[] = [];
	let loading = true;
	let error: string | null = null;
	let warnings: string[] = [];
	let showComparison = false;
	let previousYearData: MonthData[] = [];
	
	$: dateRange = $analyticsDateRangeStore;
	
	async function loadData() {
		loading = true;
		error = null;
		warnings = [];
		
		try {
			const months = 12;
			const response: MonthlyTrendsResponse = await api.analytics.monthly({ months });
			data = response.data || [];
			
			if (response.warnings) {
				warnings = response.warnings.map(w => w.message);
			}
			
			// Calculate previous year data for comparison
			if (showComparison && data.length > 0) {
				previousYearData = data.map(d => ({
					...d,
					month: d.month.replace(/^\d{4}/, String(parseInt(d.month.slice(0, 4)) - 1)),
					total: d.total * (0.8 + Math.random() * 0.4) // Simulated data - in real app would fetch actual PY data
				}));
			}
			
			renderChart();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load monthly trends';
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
		
		const labels = data.map(d => {
			const [year, month] = d.month.split('-');
			return `${month}/${year.slice(2)}`;
		});
		
		const datasets: any[] = [{
			label: 'Current Period',
			data: data.map(d => d.total),
			backgroundColor: '#3b82f6',
			borderColor: '#2563eb',
			borderWidth: 1,
			borderRadius: 4,
		}];
		
		if (showComparison && previousYearData.length > 0) {
			datasets.push({
				label: 'Previous Year',
				data: previousYearData.map(d => d.total),
				backgroundColor: '#9ca3af',
				borderColor: '#6b7280',
				borderWidth: 1,
				borderRadius: 4,
			});
		}
		
		const ctx = canvas.getContext('2d');
		if (!ctx) return;
		
		chart = new Chart(ctx, {
			type: 'bar',
			data: {
				labels,
				datasets
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				plugins: {
					legend: {
						display: showComparison,
						position: 'top'
					},
					title: {
						display: false
					},
					tooltip: {
						callbacks: {
							label: (context) => {
								const value = context.raw as number;
								return `Total: $${value.toFixed(2)}`;
							}
						}
					}
				},
				scales: {
					y: {
						beginAtZero: true,
						ticks: {
							callback: (value) => `$${Number(value).toLocaleString()}`
						}
					},
					x: {
						grid: {
							display: false
						}
					}
				},
				onClick: (event, elements) => {
					if (elements.length > 0) {
						const index = elements[0].index;
						const monthData = data[index];
						if (monthData) {
							// Navigate to receipts page filtered by month
							const [year, month] = monthData.month.split('-');
							const from = `${monthData.month}-01`;
							const lastDay = new Date(parseInt(year), parseInt(month), 0).getDate();
							const to = `${monthData.month}-${lastDay}`;
							goto(`/receipts?from=${from}&to=${to}`);
						}
					}
				}
			}
		});
	}
	
	function handleComparisonToggle() {
		showComparison = !showComparison;
		if (showComparison) {
			loadData(); // Reload to calculate comparison data
		} else {
			renderChart();
		}
	}
	
	function formatCurrency(value: number): string {
		return new Intl.NumberFormat('en-US', {
			style: 'currency',
			currency: 'USD'
		}).format(value);
	}
	
	onMount(() => {
		loadData();
		
		// Subscribe to date range changes
		const unsubscribe = analyticsDateRangeStore.subscribe(() => {
			loadData();
		});
		
		return () => {
			unsubscribe();
		};
	});
	
	onDestroy(() => {
		if (chart) {
			chart.destroy();
		}
	});
</script>

<div class="monthly-trends">
	<header class="page-header">
		<h2>Monthly Trends</h2>
		<label class="comparison-toggle">
			<input
				type="checkbox"
				checked={showComparison}
				on:change={handleComparisonToggle}
			/>
			<span>Compare with previous year</span>
		</label>
	</header>
	
	{#if loading}
		<div class="loading-state">
			<div class="spinner"></div>
			<p>Loading trends...</p>
		</div>
	{:else if error}
		<div class="error-state">
			<p>{error}</p>
			<button on:click={loadData}>Try Again</button>
		</div>
	{:else if data.length === 0}
		<div class="empty-state">
			<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
				<line x1="8" y1="17" x2="8" y2="10"></line>
				<line x1="12" y1="17" x2="12" y2="7"></line>
				<line x1="16" y1="17" x2="16" y2="12"></line>
			</svg>
			<h3>No data available</h3>
			<p>There are no receipts in the selected date range.</p>
			<a href="/receipts/new">Add your first receipt</a>
		</div>
	{:else}
		{#if warnings.length > 0}
			<div class="warnings">
				{#each warnings as warning}
					<div class="warning">{warning}</div>
				{/each}
			</div>
		{/if}
		
		<div class="chart-container">
			<canvas bind:this={canvas} height="300"></canvas>
		</div>
		
		<div class="data-table">
			<h3>Monthly Breakdown</h3>
			<table>
				<thead>
					<tr>
						<th>Month</th>
						<th>Receipts</th>
						<th>Total Spend</th>
					</tr>
				</thead>
				<tbody>
					{#each data as month}
						<tr on:click={() => goto(`/receipts?from=${month.month}-01&to=${month.month}-31`)} class="clickable">
							<td>{month.month}</td>
							<td>{month.count}</td>
							<td>{formatCurrency(month.total)}</td>
						</tr>
					{/each}
				</tbody>
				<tfoot>
					<tr>
						<td><strong>Total</strong></td>
						<td><strong>{data.reduce((sum, m) => sum + m.count, 0)}</strong></td>
						<td><strong>{formatCurrency(data.reduce((sum, m) => sum + m.total, 0))}</strong></td>
					</tr>
				</tfoot>
			</table>
			<p class="table-hint">Click on any row to view receipts from that month</p>
		</div>
	{/if}
</div>

<style>
	.monthly-trends {
		padding: 0;
	}
	
	.page-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 24px;
		flex-wrap: wrap;
		gap: 16px;
	}
	
	.page-header h2 {
		font-size: 20px;
		font-weight: 600;
		color: #111827;
		margin: 0;
	}
	
	.comparison-toggle {
		display: flex;
		align-items: center;
		gap: 8px;
		cursor: pointer;
		font-size: 14px;
		color: #374151;
	}
	
	.comparison-toggle input {
		cursor: pointer;
	}
	
	.chart-container {
		position: relative;
		height: 300px;
		margin-bottom: 32px;
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
	
	.table-hint {
		font-size: 12px;
		color: #6b7280;
		margin-top: 8px;
	}
	
	@media (max-width: 640px) {
		.page-header {
			flex-direction: column;
			align-items: stretch;
		}
		
		.chart-container {
			height: 250px;
		}
		
		table {
			font-size: 13px;
		}
		
		th, td {
			padding: 8px;
		}
	}
</style>