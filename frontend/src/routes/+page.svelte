<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/auth';
	import { api } from '$lib/api';
	import { pendingCountStore } from '$lib/stores';
	import { 
		Plus, 
		ArrowUpRight, 
		ArrowDownRight, 
		Receipt, 
		AlertCircle, 
		CheckCircle,
		RefreshCw,
		ArrowRight
	} from 'lucide-svelte';
	import type { Receipt as ReceiptType } from '$lib/api';

	// State variables
	let monthlyTotal: number = 0;
	let lastMonthTotal: number = 0;
	let pendingCount: number = 0;
	let recentReceipts: ReceiptType[] = [];
	let isLoading: boolean = true;
	let error: string | null = null;

	// Helper function to format currency
	function formatCurrency(amount: number, currency: string = 'IDR'): string {
		return new Intl.NumberFormat('id-ID', {
			style: 'currency',
			currency: currency,
			minimumFractionDigits: 0
		}).format(amount);
	}

	// Helper function to format date
	function formatDate(dateString: string): string {
		const date = new Date(dateString);
		return date.toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		});
	}

	// Helper function to get status badge color
	function getStatusColor(status: string): string {
		switch (status) {
			case 'confirmed':
				return 'bg-green-100 text-green-800';
			case 'pending_review':
				return 'bg-yellow-100 text-yellow-800';
			case 'rejected':
				return 'bg-red-100 text-red-800';
			default:
				return 'bg-gray-100 text-gray-800';
		}
	}

	// Calculate percentage change
	function getPercentageChange(current: number, previous: number): number {
		if (previous === 0) return current > 0 ? 100 : 0;
		return ((current - previous) / previous) * 100;
	}

	// Fetch all data
	async function fetchData() {
		isLoading = true;
		error = null;

		try {

			// Get current month date range
			const now = new Date();
			const currentMonthStart = new Date(now.getFullYear(), now.getMonth(), 1);
			const currentMonthEnd = new Date(now.getFullYear(), now.getMonth() + 1, 0);

			// Get last month date range
			const lastMonthStart = new Date(now.getFullYear(), now.getMonth() - 1, 1);
			const lastMonthEnd = new Date(now.getFullYear(), now.getMonth(), 0);

			// Fetch current month receipts
			const currentMonthResponse = await api.receipts.list({
				status: 'confirmed',
				start_date: currentMonthStart.toISOString().split('T')[0],
				end_date: currentMonthEnd.toISOString().split('T')[0],
				limit: 1000
			});

			// Fetch last month receipts
			const lastMonthResponse = await api.receipts.list({
				status: 'confirmed',
				start_date: lastMonthStart.toISOString().split('T')[0],
				end_date: lastMonthEnd.toISOString().split('T')[0],
				limit: 1000
			});

			// Calculate totals
			monthlyTotal = currentMonthResponse.data.reduce((sum, receipt) => sum + receipt.total, 0);
			lastMonthTotal = lastMonthResponse.data.reduce((sum, receipt) => sum + receipt.total, 0);

			// Fetch pending count
			const pendingResponse = await api.receipts.list({
				status: 'pending_review',
				limit: 1
			});
			pendingCount = pendingResponse.total;

			// Fetch recent receipts (last 5)
			const recentResponse = await api.receipts.list({
				sort_by: 'created_at',
				sort_order: 'desc',
				limit: 5
			});
			recentReceipts = recentResponse.data;

			// Update the pending store
			pendingCountStore.set(pendingCount);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load dashboard data';
		} finally {
			isLoading = false;
		}
	}

	onMount(() => {
		fetchData();
	});

	// Compute percentage change
	$: percentageChange = getPercentageChange(monthlyTotal, lastMonthTotal);
	$: isIncrease = percentageChange > 0;
	$: hasChange = percentageChange !== 0;
</script>

<div class="p-4 md:p-8 max-w-7xl mx-auto">
	<!-- Header -->
	<div class="mb-8">
		<h1 class="text-3xl font-bold text-gray-900 mb-2">Dashboard</h1>
		<p class="text-gray-600">Overview of your receipt activity</p>
	</div>

	<!-- Error State -->
	{#if error}
		<div class="bg-red-50 border border-red-200 rounded-lg p-4 mb-6 flex items-center justify-between">
			<div class="flex items-center gap-3">
				<AlertCircle class="w-5 h-5 text-red-600" />
				<span class="text-red-800">{error}</span>
			</div>
			<button
				on:click={fetchData}
				class="flex items-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors"
			>
				<RefreshCw class="w-4 h-4" />
				Retry
			</button>
		</div>
	{/if}

	<!-- Quick Actions -->
	<div class="mb-6">
		<button
			on:click={() => goto('/receipts/new')}
			class="flex items-center gap-2 px-6 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors shadow-sm"
		>
			<Plus class="w-5 h-5" />
			<span class="font-medium">Add Receipt</span>
		</button>
	</div>

	<!-- Stats Grid -->
	<div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
		<!-- Monthly Spend Card -->
		<div class="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
			{#if isLoading}
				<div class="space-y-4">
					<div class="h-4 bg-gray-200 rounded w-1/3 animate-pulse"></div>
					<div class="h-12 bg-gray-200 rounded w-2/3 animate-pulse"></div>
					<div class="h-4 bg-gray-200 rounded w-1/2 animate-pulse"></div>
				</div>
			{:else}
				<div class="flex items-center gap-2 mb-4">
					<Receipt class="w-5 h-5 text-gray-500" />
					<h2 class="text-sm font-medium text-gray-600 uppercase tracking-wide">This Month's Spend</h2>
				</div>
				<div class="mb-4">
					<div class="text-4xl font-bold text-gray-900">
						{formatCurrency(monthlyTotal, $auth.user?.home_currency || 'IDR')}
					</div>
				</div>
				{#if hasChange}
					<div class="flex items-center gap-2">
						{#if isIncrease}
							<div class="flex items-center gap-1 text-red-600">
								<ArrowUpRight class="w-4 h-4" />
								<span class="font-medium">{percentageChange.toFixed(1)}%</span>
							</div>
							<span class="text-gray-500 text-sm">vs last month</span>
						{:else}
							<div class="flex items-center gap-1 text-green-600">
								<ArrowDownRight class="w-4 h-4" />
								<span class="font-medium">{Math.abs(percentageChange).toFixed(1)}%</span>
							</div>
							<span class="text-gray-500 text-sm">vs last month</span>
						{/if}
					</div>
				{:else}
					<div class="flex items-center gap-2 text-gray-500 text-sm">
						<span>No change from last month</span>
					</div>
				{/if}
			{/if}
		</div>

		<!-- Pending Review Card -->
		<div class="bg-white rounded-xl shadow-sm border border-gray-200 p-6 {pendingCount > 0 ? 'border-yellow-300' : ''}">
			{#if isLoading}
				<div class="space-y-4">
					<div class="h-4 bg-gray-200 rounded w-1/3 animate-pulse"></div>
					<div class="h-12 bg-gray-200 rounded w-1/3 animate-pulse"></div>
				</div>
			{:else}
				<div class="flex items-center justify-between mb-4">
					<div class="flex items-center gap-2">
						{#if pendingCount > 0}
							<AlertCircle class="w-5 h-5 text-yellow-600" />
						{:else}
							<CheckCircle class="w-5 h-5 text-green-600" />
						{/if}
						<h2 class="text-sm font-medium text-gray-600 uppercase tracking-wide">Pending Review</h2>
					</div>
					{#if pendingCount > 0}
						<span class="bg-yellow-100 text-yellow-800 text-xs font-semibold px-2.5 py-0.5 rounded-full">
							{pendingCount} pending
						</span>
					{/if}
				</div>
				<div class="mb-4">
					{#if pendingCount > 0}
						<div class="text-4xl font-bold text-gray-900">{pendingCount}</div>
						<p class="text-gray-500 text-sm mt-1">receipts need your attention</p>
					{:else}
						<div class="flex items-center gap-3">
							<CheckCircle class="w-8 h-8 text-green-600" />
							<div>
								<div class="text-xl font-semibold text-gray-900">You're all caught up!</div>
								<p class="text-gray-500 text-sm">No receipts pending review</p>
							</div>
						</div>
					{/if}
				</div>
				{#if pendingCount > 0}
					<a
						href="/queue"
						class="inline-flex items-center gap-2 text-primary hover:text-primary/80 font-medium transition-colors"
						on:click|preventDefault={() => goto('/queue')}
					>
						Review now
						<ArrowRight class="w-4 h-4" />
					</a>
				{/if}
			{/if}
		</div>
	</div>

	<!-- Recent Receipts Feed -->
	<div class="bg-white rounded-xl shadow-sm border border-gray-200">
		<div class="p-6 border-b border-gray-200 flex items-center justify-between">
			<h2 class="text-lg font-semibold text-gray-900">Recent Receipts</h2>
			<a
				href="/receipts"
				class="text-primary hover:text-primary/80 font-medium text-sm transition-colors flex items-center gap-1"
				on:click|preventDefault={() => goto('/receipts')}
			>
				View all
				<ArrowRight class="w-4 h-4" />
			</a>
		</div>
		
		{#if isLoading}
			<div class="p-6 space-y-4">
				{#each Array(5) as _}
					<div class="flex items-center justify-between py-3">
						<div class="flex items-center gap-4">
							<div class="w-10 h-10 bg-gray-200 rounded-lg animate-pulse"></div>
							<div class="space-y-2">
								<div class="h-4 bg-gray-200 rounded w-32 animate-pulse"></div>
								<div class="h-3 bg-gray-200 rounded w-20 animate-pulse"></div>
							</div>
						</div>
						<div class="h-4 bg-gray-200 rounded w-16 animate-pulse"></div>
					</div>
				{/each}
			</div>
		{:else if recentReceipts.length === 0}
			<div class="p-12 text-center">
				<Receipt class="w-12 h-12 text-gray-300 mx-auto mb-4" />
				<p class="text-gray-500 mb-2">No receipts yet</p>
				<button
					on:click={() => goto('/receipts/new')}
					class="text-primary hover:text-primary/80 font-medium"
				>
					Add your first receipt
				</button>
			</div>
		{:else}
			<div class="divide-y divide-gray-100">
				{#each recentReceipts as receipt}
					<button
						on:click={() => goto(`/receipts/${receipt.id}`)}
						class="w-full p-4 flex items-center justify-between hover:bg-gray-50 transition-colors text-left"
					>
						<div class="flex items-center gap-4">
							<div class="w-10 h-10 rounded-lg bg-gray-100 flex items-center justify-center flex-shrink-0">
								<Receipt class="w-5 h-5 text-gray-500" />
							</div>
							<div>
								<div class="font-medium text-gray-900">{receipt.title}</div>
								<div class="text-sm text-gray-500">
									{formatDate(receipt.receipt_date || receipt.created_at)}
								</div>
							</div>
						</div>
						<div class="flex items-center gap-4">
							<span class="font-medium text-gray-900">
								{formatCurrency(receipt.total, receipt.currency)}
							</span>
							<span class="px-2.5 py-0.5 rounded-full text-xs font-medium {getStatusColor(receipt.status)}">
								{receipt.status}
							</span>
						</div>
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>
