<script lang="ts">
	import { onMount } from 'svelte';
	import { analyticsDateRangeStore, type DateRangePreset } from '$lib/stores';
	
	let isOpen = false;
	let pickerRef: HTMLDivElement;
	
	const presets: { value: DateRangePreset; label: string }[] = [
		{ value: 'this_week', label: 'This Week' },
		{ value: 'this_month', label: 'This Month' },
		{ value: 'last_month', label: 'Last Month' },
		{ value: 'this_quarter', label: 'This Quarter' },
		{ value: 'this_year', label: 'This Year' },
		{ value: 'custom', label: 'Custom' }
	];
	
	let customFrom: string = '';
	let customTo: string = '';
	
	onMount(() => {
		// Initialize from URL params if present
		analyticsDateRangeStore.initializeFromURL();
		
		// Subscribe to store changes to update custom date inputs
		const unsubscribe = analyticsDateRangeStore.subscribe(range => {
			if (range.preset === 'custom') {
				customFrom = range.from.toISOString().split('T')[0];
				customTo = range.to.toISOString().split('T')[0];
			}
		});
		
		return unsubscribe;
	});
	
	function handlePresetClick(preset: DateRangePreset) {
		analyticsDateRangeStore.setPreset(preset);
		if (preset !== 'custom') {
			isOpen = false;
		}
	}
	
	function handleCustomDateChange() {
		if (customFrom && customTo) {
			const from = new Date(customFrom);
			const to = new Date(customTo);
			if (!isNaN(from.getTime()) && !isNaN(to.getTime())) {
				analyticsDateRangeStore.setCustomRange(from, to);
			}
		}
	}
	
	function formatDate(date: Date): string {
		return date.toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		});
	}
	
	function handleClickOutside(event: MouseEvent) {
		if (pickerRef && !pickerRef.contains(event.target as Node)) {
			isOpen = false;
		}
	}
	
	$: currentLabel = presets.find(p => p.value === $analyticsDateRangeStore.preset)?.label || 'Custom';
</script>

<svelte:window on:click={handleClickOutside} />

<div class="date-range-picker" bind:this={pickerRef}>
	<button
		class="picker-trigger"
		on:click={() => isOpen = !isOpen}
		type="button"
	>
		<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
			<rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect>
			<line x1="16" y1="2" x2="16" y2="6"></line>
			<line x1="8" y1="2" x2="8" y2="6"></line>
			<line x1="3" y1="10" x2="21" y2="10"></line>
		</svg>
		<span class="current-range">
			{#if $analyticsDateRangeStore.preset === 'custom'}
				{formatDate($analyticsDateRangeStore.from)} - {formatDate($analyticsDateRangeStore.to)}
			{:else}
				{currentLabel}
			{/if}
		</span>
		<svg class="chevron" class:open={isOpen} xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
			<polyline points="6 9 12 15 18 9"></polyline>
		</svg>
	</button>
	
	{#if isOpen}
		<div class="picker-dropdown">
			<div class="preset-buttons">
				{#each presets as preset}
					<button
						class="preset-btn"
						class:active={$analyticsDateRangeStore.preset === preset.value}
						on:click={() => handlePresetClick(preset.value)}
						type="button"
					>
						{preset.label}
					</button>
				{/each}
			</div>
			
			{#if $analyticsDateRangeStore.preset === 'custom'}
				<div class="custom-date-inputs">
					<div class="date-input-group">
						<label for="date-from">From</label>
						<input
							type="date"
							id="date-from"
							bind:value={customFrom}
							on:change={handleCustomDateChange}
						/>
					</div>
					<div class="date-input-group">
						<label for="date-to">To</label>
						<input
							type="date"
							id="date-to"
							bind:value={customTo}
							on:change={handleCustomDateChange}
						/>
					</div>
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.date-range-picker {
		position: relative;
		display: inline-block;
	}
	
	.picker-trigger {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 12px;
		background: white;
		border: 1px solid #d1d5db;
		border-radius: 6px;
		cursor: pointer;
		font-size: 14px;
		color: #374151;
		transition: all 0.2s;
	}
	
	.picker-trigger:hover {
		border-color: #9ca3af;
		background: #f9fafb;
	}
	
	.current-range {
		white-space: nowrap;
	}
	
	.chevron {
		transition: transform 0.2s;
	}
	
	.chevron.open {
		transform: rotate(180deg);
	}
	
	.picker-dropdown {
		position: absolute;
		top: 100%;
		left: 0;
		margin-top: 4px;
		background: white;
		border: 1px solid #d1d5db;
		border-radius: 8px;
		box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
		padding: 12px;
		min-width: 280px;
		z-index: 50;
	}
	
	.preset-buttons {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 8px;
		margin-bottom: 12px;
	}
	
	.preset-btn {
		padding: 8px 12px;
		border: 1px solid #e5e7eb;
		border-radius: 6px;
		background: white;
		cursor: pointer;
		font-size: 13px;
		color: #374151;
		transition: all 0.2s;
	}
	
	.preset-btn:hover {
		background: #f3f4f6;
		border-color: #d1d5db;
	}
	
	.preset-btn.active {
		background: #3b82f6;
		color: white;
		border-color: #3b82f6;
	}
	
	.custom-date-inputs {
		display: flex;
		gap: 12px;
		padding-top: 12px;
		border-top: 1px solid #e5e7eb;
	}
	
	.date-input-group {
		flex: 1;
	}
	
	.date-input-group label {
		display: block;
		font-size: 12px;
		color: #6b7280;
		margin-bottom: 4px;
	}
	
	.date-input-group input {
		width: 100%;
		padding: 6px 8px;
		border: 1px solid #d1d5db;
		border-radius: 4px;
		font-size: 13px;
	}
	
	.date-input-group input:focus {
		outline: none;
		border-color: #3b82f6;
		box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
	}
	
	@media (max-width: 640px) {
		.preset-buttons {
			grid-template-columns: 1fr;
		}
		
		.custom-date-inputs {
			flex-direction: column;
			gap: 8px;
		}
	}
</style>