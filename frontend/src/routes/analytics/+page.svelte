<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { analyticsDateRangeStore, toastStore } from '$lib/stores';
	import api from '$lib/api';
	import type { AnalyticsSummary, Insights, BudgetStatus } from '$lib/api';
	
	let summary: AnalyticsSummary | null = null;
	let insights: Insights | null = null;
	let loading = true;
	let error: string | null = null;
	let warnings: string[] = [];
	let exporting = false;
	
	// Budget alerts state
	let budgetAlerts: BudgetStatus[] = [];
	let loadingBudgetAlerts = false;
	
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
			
			// Fetch both summary and insights in parallel
			const [summaryRes, insightsRes] = await Promise.all([
				api.analytics.summary(params),
				api.analytics.insights(params)
			]);
			
			summary = summaryRes.data;
			insights = insightsRes.data;
			
			// Collect all warnings
			if (summaryRes.warnings) {
				warnings.push(...summaryRes.warnings.map((w: { message: string }) => w.message));
			}
			if (insightsRes.warnings) {
				warnings.push(...insightsRes.warnings.map((w: { message: string }) => w.message));
			}
			
			// Fetch budget alerts
			await loadBudgetAlerts();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load analytics';
		} finally {
			loading = false;
		}
	}
	
	async function loadBudgetAlerts() {
		loadingBudgetAlerts = true;
		try {
			// Get current month in YYYY-MM format
			const now = new Date();
			const currentMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
			
			const statuses = await api.budgets.status(currentMonth);
			
			// Filter budgets over threshold (>=80%)
			budgetAlerts = statuses.filter((status: BudgetStatus) => status.percentage >= 80);
		} catch (err) {
			// Silently fail - budget alerts are not critical
			console.error('Failed to load budget alerts:', err);
			budgetAlerts = [];
		} finally {
			loadingBudgetAlerts = false;
		}
	}
	
	function formatCurrency(value: number): string {
		return new Intl.NumberFormat('en-US', {
			style: 'currency',
			currency: 'USD'
		}).format(value);
	}
	
	async function exportCSV() {
		exporting = true;
		
		try {
			const { from, to } = $analyticsDateRangeStore;
			const blob = await api.receipts.export({
				from: from.toISOString(),
				to: to.toISOString(),
				format: 'csv'
			});
			
			// Trigger download
			const url = window.URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `receipts_${from.toISOString().split('T')[0]}_${to.toISOString().split('T')[0]}.csv`;
			document.body.appendChild(a);
			a.click();
			document.body.removeChild(a);
			window.URL.revokeObjectURL(url);
			
			toastStore.success('CSV exported successfully');
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Export failed';
			toastStore.error(message);
		} finally {
			exporting = false;
		}
	}
	
	function getDayName(day: string): string {
		const days: Record<string, string> = {
			'0': 'Sunday',
			'1': 'Monday',
			'2': 'Tuesday',
			'3': 'Wednesday',
			'4': 'Thursday',
			'5': 'Friday',
			'6': 'Saturday'
		};
		return days[day] || day;
	}
	
	function getTrendIcon(percentage: number): string {
		if (percentage > 0) return '↑';
		if (percentage < 0) return '↓';
		return '→';
	}
	
	function getTrendClass(percentage: number): string {
		if (percentage > 0) return 'positive';
		if (percentage < 0) return 'negative';
		return 'neutral';
	}
	
	function getBudgetAlertClass(percentage: number): string {
		if (percentage >= 100) return 'critical';
		if (percentage >= 80) return 'warning';
		return '';
	}
	
	function getBudgetAlertMessage(alert: BudgetStatus): string {
		if (alert.percentage >= 100) {
			return `Budget exceeded: You've spent ${formatCurrency(alert.spent)} of your ${formatCurrency(alert.amount_limit)} limit (${alert.percentage.toFixed(1)}%)`;
		}
		return `Budget warning: You've spent ${formatCurrency(alert.spent)} of your ${formatCurrency(alert.amount_limit)} limit (${alert.percentage.toFixed(1)}%)`;
	}
	
	$: dateRange = $analyticsDateRangeStore;
	
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

<div class="analytics-home">
	{#if loading}
		<div class="loading-state">
			<div class="spinner"></div>
			<p>Loading analytics...</p>
		</div>
	{:else if error}
		<div class="error-state">
			<p>{error}</p>
			<button on:click={loadData}>Try Again</button>
		</div>
	{:else}
		{#if warnings.length > 0}
			<div class="warnings">
				{#each warnings as warning}
					<div class="warning">{warning}</div>
				{/each}
			</div>
		{/if}
		
		{#if budgetAlerts.length > 0}
			<div class="budget-alerts">
				{#each budgetAlerts as alert}
					<div class="budget-alert {getBudgetAlertClass(alert.percentage)}">
						<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
							<line x1="12" y1="9" x2="12" y2="13"></line>
							<line x1="12" y1="17" x2="12.01" y2="17"></line>
						</svg>
						<span>{getBudgetAlertMessage(alert)}</span>
						<a href="/settings/budgets" class="alert-link">View Budgets</a>
					</div>
				{/each}
			</div>
		{/if}
		
		<section class="page-header">
			<h2>Overview</h2>
			<button 
				class="export-button" 
				on:click={exportCSV} 
				disabled={exporting}
			>
				{#if exporting}
					<div class="button-spinner"></div>
					<span>Exporting...</span>
				{:else}
					<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
						<polyline points="7 10 12 15 17 10"></polyline>
						<line x1="12" y1="15" x2="12" y2="3"></line>
					</svg>
					<span>Export CSV</span>
				{/if}
			</button>
		</section>
		
		<section class="summary-cards">
			<div class="summary-card">
				<div class="card-icon total">
					<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<line x1="12" y1="1" x2="12" y2="23"></line>
						<path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"></path>
					</svg>
				</div>
				<div class="card-content">
					<span class="card-label">Total Spend</span>
					<span class="card-value">{summary ? formatCurrency(summary.total_spend) : '-'}</span>
				</div>
			</div>
			
			<div class="summary-card">
				<div class="card-icon count">
					<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
						<polyline points="14 2 14 8 20 8"></polyline>
						<line x1="16" y1="13" x2="8" y2="13"></line>
						<line x1="16" y1="17" x2="8" y2="17"></line>
						<polyline points="10 9 9 9 8 9"></polyline>
					</svg>
				</div>
				<div class="card-content">
					<span class="card-label">Receipts</span>
					<span class="card-value">{summary ? summary.receipt_count : '-'}</span>
				</div>
			</div>
			
			<div class="summary-card">
				<div class="card-icon avg">
					<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M18 20V10"></path>
						<path d="M12 20V4"></path>
						<path d="M6 20v-6"></path>
					</svg>
				</div>
				<div class="card-content">
					<span class="card-label">Avg per Receipt</span>
					<span class="card-value">{summary ? formatCurrency(summary.avg_per_receipt) : '-'}</span>
				</div>
			</div>
			
			<div class="summary-card">
				<div class="card-icon daily">
					<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect>
						<line x1="16" y1="2" x2="16" y2="6"></line>
						<line x1="8" y1="2" x2="8" y2="6"></line>
						<line x1="3" y1="10" x2="21" y2="10"></line>
					</svg>
				</div>
				<div class="card-content">
					<span class="card-label">Daily Average</span>
					{#if summary}
						{@const days = Math.ceil((dateRange.to.getTime() - dateRange.from.getTime()) / (1000 * 60 * 60 * 24)) || 1}
						<span class="card-value">{formatCurrency(summary.total_spend / days)}</span>
					{:else}
						<span class="card-value">-</span>
					{/if}
				</div>
			</div>
		</section>
		
		{#if insights}
			<section class="insights">
				<h3>Insights</h3>
				<div class="insights-grid">
					{#if insights.biggest_receipt}
						<div class="insight-card" on:click={() => goto(`/receipts/${insights?.biggest_receipt?.id}`)}>
							<div class="insight-icon">
								<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<polyline points="23 6 13.5 15.5 8.5 10.5 1 18"></polyline>
									<polyline points="17 6 23 6 23 12"></polyline>
								</svg>
							</div>
							<div class="insight-content">
								<span class="insight-label">Biggest Receipt</span>
								<span class="insight-value">{formatCurrency(insights.biggest_receipt.total)}</span>
								<span class="insight-detail">{insights.biggest_receipt.title}</span>
							</div>
						</div>
					{/if}
					
					{#if insights.most_visited_shop}
						<div class="insight-card" on:click={() => goto(`/receipts?q=${encodeURIComponent(insights?.most_visited_shop?.name || '')}`)}>
							<div class="insight-icon">
								<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path>
									<polyline points="9 22 9 12 15 12 15 22"></polyline>
								</svg>
							</div>
							<div class="insight-content">
								<span class="insight-label">Most Visited</span>
								<span class="insight-value">{insights.most_visited_shop.name}</span>
								<span class="insight-detail">{insights.most_visited_shop.visits} visits</span>
							</div>
						</div>
					{/if}
					
					{#if insights.mom_change}
						<div class="insight-card">
							<div class="insight-icon trend {getTrendClass(insights.mom_change.percentage)}">
								<span class="trend-icon">{getTrendIcon(insights.mom_change.percentage)}</span>
							</div>
							<div class="insight-content">
								<span class="insight-label">vs Last Month</span>
								<span class="insight-value {getTrendClass(insights.mom_change.percentage)}">
									{insights.mom_change.percentage > 0 ? '+' : ''}{insights.mom_change.percentage.toFixed(1)}%
								</span>
								<span class="insight-detail">
									{insights.mom_change.absolute > 0 ? '+' : ''}{formatCurrency(insights.mom_change.absolute)}
								</span>
							</div>
						</div>
					{/if}
					
					{#if insights.busiest_day}
						<div class="insight-card">
							<div class="insight-icon">
								<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<circle cx="12" cy="12" r="10"></circle>
									<polyline points="12 6 12 12 16 14"></polyline>
								</svg>
							</div>
							<div class="insight-content">
								<span class="insight-label">Busiest Day</span>
								<span class="insight-value">{getDayName(insights.busiest_day.day)}</span>
								<span class="insight-detail">{formatCurrency(insights.busiest_day.total)}</span>
							</div>
						</div>
					{/if}
				</div>
			</section>
		{/if}
		
		<section class="quick-links">
			<h3>Detailed Reports</h3>
			<div class="links-grid">
				<a href="/analytics/monthly" class="quick-link">
					<div class="link-icon">
						<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect>
							<line x1="16" y1="2" x2="16" y2="6"></line>
							<line x1="8" y1="2" x2="8" y2="6"></line>
							<line x1="3" y1="10" x2="21" y2="10"></line>
						</svg>
					</div>
					<span>Monthly Trends</span>
				</a>
				
				<a href="/analytics/by-tag" class="quick-link">
					<div class="link-icon">
						<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"></path>
							<line x1="7" y1="7" x2="7.01" y2="7"></line>
						</svg>
					</div>
					<span>By Category</span>
				</a>
				
				<a href="/analytics/by-shop" class="quick-link">
					<div class="link-icon">
						<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path>
							<polyline points="9 22 9 12 15 12 15 22"></polyline>
						</svg>
					</div>
					<span>By Shop</span>
				</a>
			</div>
		</section>
	{/if}
</div>

<style>
	.analytics-home {
		padding: 0;
	}
	
	.loading-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 48px 24px;
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
	
	.error-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 48px 24px;
		text-align: center;
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
	
	.warnings {
		margin-bottom: 24px;
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
	
	.budget-alerts {
		margin-bottom: 24px;
	}
	
	.budget-alert {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 12px;
		border-radius: 6px;
		font-size: 14px;
		margin-bottom: 8px;
	}
	
	.budget-alert.warning {
		background: #fef3c7;
		border: 1px solid #f59e0b;
		color: #92400e;
	}
	
	.budget-alert.critical {
		background: #fee2e2;
		border: 1px solid #ef4444;
		color: #dc2626;
	}
	
	.alert-link {
		margin-left: auto;
		color: inherit;
		text-decoration: underline;
		font-weight: 500;
	}
	
	.alert-link:hover {
		opacity: 0.8;
	}
	
	.summary-cards {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 16px;
		margin-bottom: 32px;
	}
	
	.summary-card {
		display: flex;
		align-items: center;
		gap: 12px;
		background: #f9fafb;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		padding: 16px;
	}
	
	.card-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 48px;
		height: 48px;
		border-radius: 8px;
		flex-shrink: 0;
	}
	
	.card-icon.total {
		background: #dbeafe;
		color: #2563eb;
	}
	
	.card-icon.count {
		background: #d1fae5;
		color: #059669;
	}
	
	.card-icon.avg {
		background: #ede9fe;
		color: #7c3aed;
	}
	
	.card-icon.daily {
		background: #fce7f3;
		color: #db2777;
	}
	
	.card-content {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	
	.card-label {
		font-size: 12px;
		color: #6b7280;
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}
	
	.card-value {
		font-size: 20px;
		font-weight: 600;
		color: #111827;
	}
	
	.insights {
		margin-bottom: 32px;
	}
	
	.insights h3,
	.quick-links h3 {
		font-size: 16px;
		font-weight: 600;
		color: #374151;
		margin: 0 0 16px 0;
	}
	
	.insights-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
		gap: 16px;
	}
	
	.insight-card {
		display: flex;
		align-items: center;
		gap: 12px;
		background: white;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		padding: 16px;
		cursor: pointer;
		transition: all 0.2s;
	}
	
	.insight-card:hover {
		border-color: #3b82f6;
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
	}
	
	.insight-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 40px;
		height: 40px;
		background: #f3f4f6;
		border-radius: 8px;
		color: #6b7280;
		flex-shrink: 0;
	}
	
	.insight-icon.trend {
		background: #f3f4f6;
	}
	
	.insight-icon.trend.positive {
		background: #d1fae5;
		color: #059669;
	}
	
	.insight-icon.trend.negative {
		background: #fee2e2;
		color: #dc2626;
	}
	
	.trend-icon {
		font-size: 18px;
		font-weight: 700;
	}
	
	.insight-content {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}
	
	.insight-label {
		font-size: 12px;
		color: #6b7280;
	}
	
	.insight-value {
		font-size: 18px;
		font-weight: 600;
		color: #111827;
	}
	
	.insight-value.positive {
		color: #059669;
	}
	
	.insight-value.negative {
		color: #dc2626;
	}
	
	.insight-detail {
		font-size: 13px;
		color: #6b7280;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	
	.quick-links {
		margin-bottom: 32px;
	}
	
	.links-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
		gap: 12px;
	}
	
	.quick-link {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		padding: 20px;
		background: white;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		text-decoration: none;
		color: #374151;
		transition: all 0.2s;
	}
	
	.quick-link:hover {
		border-color: #3b82f6;
		background: #eff6ff;
	}
	
	.link-icon {
		color: #3b82f6;
	}
	
	.quick-link span {
		font-size: 14px;
		font-weight: 500;
	}
	
	.page-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 24px;
	}
	
	.page-header h2 {
		font-size: 18px;
		font-weight: 600;
		color: #111827;
		margin: 0;
	}
	
	.export-button {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 16px;
		background: #3b82f6;
		color: white;
		border: none;
		border-radius: 6px;
		cursor: pointer;
		font-size: 14px;
		font-weight: 500;
		transition: all 0.2s;
	}
	
	.export-button:hover:not(:disabled) {
		background: #2563eb;
	}
	
	.export-button:disabled {
		opacity: 0.7;
		cursor: not-allowed;
	}
	
	.button-spinner {
		width: 14px;
		height: 14px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: white;
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}
	
	@media (max-width: 640px) {
		.page-header {
			flex-direction: column;
			align-items: flex-start;
			gap: 12px;
		}
		
		.summary-cards {
			grid-template-columns: repeat(2, 1fr);
		}
		
		.summary-card {
			flex-direction: column;
			text-align: center;
		}
		
		.card-icon {
			width: 40px;
			height: 40px;
		}
		
		.card-value {
			font-size: 16px;
		}
		
		.insights-grid {
			grid-template-columns: 1fr;
		}
	}
</style>