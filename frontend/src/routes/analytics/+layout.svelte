<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import DateRangePicker from '$components/DateRangePicker.svelte';
	import { analyticsDateRangeStore } from '$lib/stores';
	
	onMount(() => {
		// Initialize date range from URL on mount
		analyticsDateRangeStore.initializeFromURL();
	});
	
	const navItems = [
		{ href: '/analytics', label: 'Overview', exact: true },
		{ href: '/analytics/monthly', label: 'Monthly' },
		{ href: '/analytics/by-tag', label: 'By Tag' },
		{ href: '/analytics/by-shop', label: 'By Shop' }
	];
	
	$: currentPath = $page.url.pathname;
</script>

<div class="analytics-layout">
	<header class="analytics-header">
		<h1>Analytics</h1>
		<DateRangePicker />
	</header>
	
	<nav class="analytics-nav">
		{#each navItems as item}
			<a
				href={item.href}
				class="nav-link"
				class:active={item.exact ? currentPath === item.href : currentPath.startsWith(item.href)}
			>
				{item.label}
			</a>
		{/each}
	</nav>
	
	<main class="analytics-content">
		<slot />
	</main>
</div>

<style>
	.analytics-layout {
		max-width: 1200px;
		margin: 0 auto;
		padding: 24px 16px;
	}
	
	.analytics-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 24px;
		flex-wrap: wrap;
		gap: 16px;
	}
	
	.analytics-header h1 {
		font-size: 28px;
		font-weight: 600;
		color: #111827;
		margin: 0;
	}
	
	.analytics-nav {
		display: flex;
		gap: 8px;
		margin-bottom: 24px;
		border-bottom: 1px solid #e5e7eb;
		padding-bottom: 1px;
		flex-wrap: wrap;
	}
	
	.nav-link {
		padding: 12px 16px;
		color: #6b7280;
		text-decoration: none;
		font-weight: 500;
		border-bottom: 2px solid transparent;
		transition: all 0.2s;
		white-space: nowrap;
	}
	
	.nav-link:hover {
		color: #374151;
	}
	
	.nav-link.active {
		color: #3b82f6;
		border-bottom-color: #3b82f6;
	}
	
	.analytics-content {
		background: white;
		border-radius: 8px;
		padding: 24px;
		border: 1px solid #e5e7eb;
	}
	
	@media (max-width: 640px) {
		.analytics-header {
			flex-direction: column;
			align-items: stretch;
		}
		
		.analytics-header h1 {
			font-size: 24px;
		}
		
		.analytics-nav {
			gap: 4px;
		}
		
		.nav-link {
			padding: 10px 12px;
			font-size: 14px;
		}
		
		.analytics-content {
			padding: 16px;
		}
	}
</style>