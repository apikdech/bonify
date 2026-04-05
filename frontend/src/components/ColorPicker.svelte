<script lang="ts">
	import { createEventDispatcher } from 'svelte';

	// Props
	export let selectedColor: string = '#3b82f6';

	// Dispatch events
	const dispatch = createEventDispatcher<{ select: string }>();

	// Preset colors
	const presetColors = [
		'#ef4444',
		'#f97316',
		'#f59e0b',
		'#22c55e',
		'#10b981',
		'#14b8a6',
		'#06b6d4',
		'#3b82f6',
		'#6366f1',
		'#8b5cf6',
		'#a855f7',
		'#ec4899',
		'#f43f5e',
		'#64748b',
		'#6b7280',
		'#71717a'
	];

	// Hex validation regex
	const hexRegex = /^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$/;

	// Computed
	$: isValidHex = hexRegex.test(selectedColor);

	function handleColorSelect(color: string) {
		selectedColor = color;
		dispatch('select', color);
	}

	function handleHexInput(event: Event) {
		const target = event.target as HTMLInputElement;
		let value = target.value.trim();

		// Auto-add # if missing
		if (value && !value.startsWith('#')) {
			value = '#' + value;
		}

		selectedColor = value.toLowerCase();

		if (hexRegex.test(selectedColor)) {
			dispatch('select', selectedColor);
		}
	}
</script>

<div class="space-y-4">
	<!-- Color Grid -->
	<div class="grid grid-cols-8 gap-2">
		{#each presetColors as color}
			<button
				type="button"
				on:click={() => handleColorSelect(color)}
				class="w-8 h-8 rounded-lg transition-all duration-200 hover:scale-110 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary {selectedColor === color ? 'ring-2 ring-offset-2 ring-primary scale-110' : ''}"
				style="background-color: {color}"
				aria-label="Select color {color}"
				title={color}
			></button>
		{/each}
	</div>

	<!-- Hex Input and Preview -->
	<div class="flex items-center gap-3">
		<!-- Live Preview -->
		<div
			class="w-12 h-12 rounded-lg border-2 border-gray-200 flex-shrink-0 transition-colors duration-200"
			style="background-color: {isValidHex ? selectedColor : '#e5e7eb'}"
		></div>

		<!-- Hex Input -->
		<div class="flex-1">
			<label for="color-hex" class="block text-xs font-medium text-gray-700 mb-1"> Hex Code </label>
			<input
				type="text"
				id="color-hex"
				value={selectedColor}
				on:input={handleHexInput}
				placeholder="#3b82f6"
				class="w-full px-3 py-2 border rounded-lg text-sm font-mono transition-colors focus:outline-none focus:ring-2 focus:ring-primary/20 {isValidHex ? 'border-gray-200 focus:border-primary' : 'border-red-300 focus:border-red-500 focus:ring-red-500/20'}"
			/>
			{#if !isValidHex && selectedColor}
				<p class="text-xs text-red-500 mt-1">Invalid hex code</p>
			{/if}
		</div>
	</div>
</div>
