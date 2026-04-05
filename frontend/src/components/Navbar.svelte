<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/stores';
	import { Home, Receipt, Inbox, BarChart3, Tag, LogOut } from 'lucide-svelte';
	import auth from '$lib/auth';
	import { pendingCountStore } from '$lib/stores';

	// Navigation items configuration
	const navItems = [
		{ path: '/', label: 'Home', icon: Home },
		{ path: '/receipts', label: 'Receipts', icon: Receipt },
		{ path: '/queue', label: 'Queue', icon: Inbox, showBadge: true },
		{ path: '/analytics', label: 'Analytics', icon: BarChart3 },
		{ path: '/tags', label: 'Tags', icon: Tag }
	];

	// Mobile nav items (subset for space)
	const mobileNavItems = [
		{ path: '/', label: 'Home', icon: Home },
		{ path: '/receipts', label: 'Receipts', icon: Receipt },
		{ path: '/queue', label: 'Queue', icon: Inbox, showBadge: true },
		{ path: '/analytics', label: 'Analytics', icon: BarChart3 }
	];

	$: currentPath = $page.url.pathname;
	$: isAuthenticated = $auth.isAuthenticated;
	$: user = $auth.user;
	$: pendingCount = $pendingCountStore;

	function isActive(path: string): boolean {
		if (path === '/') {
			return currentPath === '/';
		}
		return currentPath.startsWith(path);
	}

	async function handleLogout() {
		await auth.logout();
	}

	onMount(() => {
		// Start polling for pending count when authenticated
		if (isAuthenticated) {
			pendingCountStore.startPolling(30000); // Poll every 30 seconds
		}
	});

	onDestroy(() => {
		// Stop polling when component is destroyed
		pendingCountStore.stopPolling();
	});
</script>

{#if isAuthenticated}
	<!-- Desktop Sidebar -->
	<aside
		class="fixed left-0 top-0 z-40 hidden h-screen w-64 flex-col border-r border-gray-200 bg-white md:flex"
	>
		<!-- Logo Section -->
		<div class="flex h-16 items-center border-b border-gray-200 px-6">
			<div class="flex items-center gap-2">
				<Receipt class="h-6 w-6 text-primary" />
				<span class="text-lg font-semibold text-gray-900">Receipt Manager</span>
			</div>
		</div>

		<!-- Navigation Links -->
		<nav class="flex-1 overflow-y-auto px-4 py-4">
			<ul class="space-y-1">
				{#each navItems as item}
					{@const Icon = item.icon}
					{@const active = isActive(item.path)}
					<li>
						<a
							href={item.path}
							class="flex items-center gap-3 rounded-lg px-4 py-3 text-sm font-medium transition-colors duration-200 {active
								? 'bg-primary text-white'
								: 'text-gray-700 hover:bg-gray-100'}"
						>
							<div class="relative">
								<Icon class="h-5 w-5" />
								{#if item.showBadge && pendingCount > 0}
									<span
										class="absolute -right-1.5 -top-1.5 flex h-4 w-4 items-center justify-center rounded-full bg-red-500 text-[10px] font-bold text-white"
									>
										{pendingCount > 99 ? '99+' : pendingCount}
									</span>
								{/if}
							</div>
							<span>{item.label}</span>
						</a>
					</li>
				{/each}
			</ul>
		</nav>

		<!-- User Section -->
		<div class="border-t border-gray-200 p-4">
			<div class="flex items-center gap-3 rounded-lg bg-gray-50 px-4 py-3">
				<div class="flex-1 min-w-0">
					<p class="truncate text-sm font-medium text-gray-900">
						{user?.name || user?.email || 'User'}
					</p>
					<p class="truncate text-xs text-gray-500 capitalize">
						{user?.role || 'User'}
					</p>
				</div>
				<button
					on:click={handleLogout}
					class="rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-200 hover:text-gray-700"
					title="Logout"
				>
					<LogOut class="h-4 w-4" />
				</button>
			</div>
		</div>
	</aside>

	<!-- Mobile Bottom Navigation -->
	<nav class="fixed bottom-0 left-0 right-0 z-50 border-t border-gray-200 bg-white md:hidden">
		<div class="flex items-center justify-around px-2 py-2">
			{#each mobileNavItems as item}
				{@const Icon = item.icon}
				{@const active = isActive(item.path)}
				<a
					href={item.path}
					class="relative flex flex-col items-center gap-1 rounded-lg px-3 py-2 transition-colors duration-200 {active
						? 'text-primary'
						: 'text-gray-500 hover:text-gray-700'}"
				>
					<div class="relative">
						<Icon class="h-5 w-5" />
						{#if item.showBadge && pendingCount > 0}
							<span
								class="absolute -right-2 -top-2 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white"
							>
								{pendingCount > 99 ? '99+' : pendingCount}
							</span>
						{/if}
					</div>
					<span class="text-[10px] font-medium">{item.label}</span>
				</a>
			{/each}
		</div>
	</nav>

	<!-- Spacer for mobile bottom nav -->
	<div class="h-16 md:hidden"></div>
{/if}
