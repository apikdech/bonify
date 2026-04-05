<script lang="ts">
	import { onMount } from 'svelte';
	import { Bell, Save, Check, Loader2 } from 'lucide-svelte';
	import { api, type User } from '$lib/api';
	import { toastStore } from '$lib/stores';

	// ============ State ============

	// User data
	let user: User | null = null;

	// Notification settings
	let notifyOnParse = false;
	let notifyOnPendingReview = false;
	let notifyBudgetAlerts = false;

	// Loading states
	let isLoading = true;
	let isSaving = false;
	let hasChanges = false;

	// ============ Computed ============

	$: canSave = hasChanges && !isSaving;

	// ============ Data Fetching ============

	async function loadUser() {
		isLoading = true;
		try {
			user = await api.auth.me();
			// Initialize toggle values from user data
			notifyOnParse = user.notify_on_parse ?? false;
			notifyOnPendingReview = user.notify_on_pending_review ?? false;
			notifyBudgetAlerts = user.notify_budget_alerts ?? false;
			hasChanges = false;
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to load settings';
			toastStore.error(message);
		} finally {
			isLoading = false;
		}
	}

	// ============ Save Settings ============

	async function saveSettings() {
		if (!canSave || !user) return;

		isSaving = true;
		try {
			await api.user.update({
				notify_on_parse: notifyOnParse,
				notify_on_pending_review: notifyOnPendingReview,
				notify_budget_alerts: notifyBudgetAlerts
			});
			toastStore.success('Notification settings saved');
			hasChanges = false;
			// Refresh user data to sync with server
			await loadUser();
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to save settings';
			toastStore.error(message);
		} finally {
			isSaving = false;
		}
	}

	// ============ Event Handlers ============

	function handleToggleChange() {
		if (!user) return;
		hasChanges =
			notifyOnParse !== user.notify_on_parse ||
			notifyOnPendingReview !== user.notify_on_pending_review ||
			notifyBudgetAlerts !== user.notify_budget_alerts;
	}

	// ============ Lifecycle ============

	onMount(() => {
		loadUser();
	});
</script>

<div class="p-4 md:p-8 max-w-3xl mx-auto">
	<!-- Header -->
	<div class="mb-8">
		<h1 class="text-3xl font-bold text-gray-900 mb-2">Settings</h1>
		<p class="text-gray-600">Manage your account preferences and notification settings</p>
	</div>

	<!-- Loading State -->
	{#if isLoading}
		<div class="bg-white rounded-xl border border-gray-200 p-8 animate-pulse">
			<div class="flex items-center gap-4 mb-6">
				<div class="w-12 h-12 rounded-full bg-gray-200"></div>
				<div class="space-y-2">
					<div class="h-4 bg-gray-200 rounded w-48"></div>
					<div class="h-3 bg-gray-200 rounded w-32"></div>
				</div>
			</div>
			<div class="space-y-4">
				<div class="h-12 bg-gray-200 rounded-lg"></div>
				<div class="h-12 bg-gray-200 rounded-lg"></div>
				<div class="h-12 bg-gray-200 rounded-lg"></div>
			</div>
		</div>

		<!-- Notification Settings Card -->
	{:else}
		<div class="bg-white rounded-xl border border-gray-200 p-6">
			<!-- Card Header -->
			<div class="flex items-start gap-4 mb-6 pb-6 border-b border-gray-200">
				<div class="w-12 h-12 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
					<Bell class="w-6 h-6 text-primary" />
				</div>
				<div>
					<h2 class="text-xl font-semibold text-gray-900">Notifications</h2>
					<p class="text-gray-500 text-sm mt-1">
						Choose when you'd like to be notified about your receipts and budgets
					</p>
				</div>
			</div>

			<!-- Toggle Options -->
			<div class="space-y-6">
				<!-- Toggle: Notify when receipt is parsed -->
				<div class="flex items-center justify-between">
					<div class="flex-1 pr-4">
						<h3 class="font-medium text-gray-900">Receipt parsed</h3>
						<p class="text-sm text-gray-500 mt-1">
							Get notified when a new receipt is successfully processed
						</p>
					</div>
					<label class="relative inline-flex items-center cursor-pointer">
						<input
							type="checkbox"
							bind:checked={notifyOnParse}
							on:change={handleToggleChange}
							class="sr-only peer"
						/>
						<div
							class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary/20 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"
						></div>
					</label>
				</div>

				<!-- Toggle: Notify when review is needed -->
				<div class="flex items-center justify-between">
					<div class="flex-1 pr-4">
						<h3 class="font-medium text-gray-900">Review needed (24h+)</h3>
						<p class="text-sm text-gray-500 mt-1">
							Get notified when receipts have been pending review for more than 24 hours
						</p>
					</div>
					<label class="relative inline-flex items-center cursor-pointer">
						<input
							type="checkbox"
							bind:checked={notifyOnPendingReview}
							on:change={handleToggleChange}
							class="sr-only peer"
						/>
						<div
							class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary/20 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"
						></div>
					</label>
				</div>

				<!-- Toggle: Notify when approaching budget limit -->
				<div class="flex items-center justify-between">
					<div class="flex-1 pr-4">
						<h3 class="font-medium text-gray-900">Budget limit approaching</h3>
						<p class="text-sm text-gray-500 mt-1">
							Get notified when you're approaching your monthly budget limits
						</p>
					</div>
					<label class="relative inline-flex items-center cursor-pointer">
						<input
							type="checkbox"
							bind:checked={notifyBudgetAlerts}
							on:change={handleToggleChange}
							class="sr-only peer"
						/>
						<div
							class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary/20 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"
						></div>
					</label>
				</div>
			</div>

			<!-- Action Buttons -->
			<div class="flex items-center justify-between mt-8 pt-6 border-t border-gray-200">
				<div class="text-sm text-gray-500">
					{#if hasChanges}
						<span class="flex items-center gap-1 text-amber-600">
							<span class="w-2 h-2 rounded-full bg-amber-500"></span>
							Unsaved changes
						</span>
					{:else}
						<span class="flex items-center gap-1 text-green-600">
							<Check class="w-4 h-4" />
							All changes saved
						</span>
					{/if}
				</div>
				<button
					on:click={saveSettings}
					disabled={!canSave}
					class="flex items-center gap-2 px-6 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{#if isSaving}
						<Loader2 class="w-5 h-5 animate-spin" />
						<span>Saving...</span>
					{:else}
						<Save class="w-5 h-5" />
						<span>Save Changes</span>
					{/if}
				</button>
			</div>
		</div>
	{/if}
</div>
