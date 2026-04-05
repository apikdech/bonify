<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { api, type ReceiptItem, type ReceiptFee, type Tag, type CreateReceiptRequest } from '$lib/api';
	import { toastStore } from '$lib/stores';
	import {
		ArrowLeft,
		Plus,
		Trash2,
		Save,
		X,
		Calculator,
		Receipt as ReceiptIcon,
		Tag as TagIcon,
		Loader2,
		AlertCircle
	} from 'lucide-svelte';

	// ============ State ============
	
	// Form mode
	let isEditMode = false;
	let receiptId: string | null = null;
	
	// Loading states
	let isSubmitting = false;
	let isLoadingTags = true;
	let isLoadingReceipt = false;
	
	// Error state
	let formErrors: Record<string, string> = {};
	
	// Available tags
	let availableTags: Tag[] = [];
	
	// Form data
	let shopName = '';
	let receiptDate = new Date().toISOString().split('T')[0]; // Default to today
	let currency = 'IDR';
	let paymentMethod = 'cash';
	let notes = '';
	let selectedTagIds: string[] = [];
	
	// Line items
	let items: Array<{
		id: string;
		name: string;
		quantity: number;
		unitPrice: number;
		discount: number;
	}> = [];
	
	// Fees
	let fees: Array<{
		id: string;
		label: string;
		type: 'tax' | 'service' | 'delivery' | 'tip' | 'other';
		amount: number;
	}> = [];
	
	// ============ Constants ============
	
	const currencies = [
		{ value: 'IDR', label: 'IDR - Indonesian Rupiah' },
		{ value: 'USD', label: 'USD - US Dollar' },
		{ value: 'SGD', label: 'SGD - Singapore Dollar' },
		{ value: 'MYR', label: 'MYR - Malaysian Ringgit' },
		{ value: 'EUR', label: 'EUR - Euro' },
		{ value: 'GBP', label: 'GBP - British Pound' },
		{ value: 'JPY', label: 'JPY - Japanese Yen' },
		{ value: 'AUD', label: 'AUD - Australian Dollar' }
	];
	
	const paymentMethods = [
		{ value: 'cash', label: 'Cash' },
		{ value: 'card', label: 'Card' },
		{ value: 'qris', label: 'QRIS' },
		{ value: 'transfer', label: 'Bank Transfer' },
		{ value: 'ewallet', label: 'E-Wallet' },
		{ value: 'other', label: 'Other' }
	];
	
	const feeTypes = [
		{ value: 'tax', label: 'Tax' },
		{ value: 'service', label: 'Service Charge' },
		{ value: 'delivery', label: 'Delivery' },
		{ value: 'tip', label: 'Tip' },
		{ value: 'other', label: 'Other' }
	];
	
	// ============ Computed ============
	
	// Calculate item subtotals
	$: itemSubtotals = items.map(item => ({
		...item,
		subtotal: item.quantity * item.unitPrice - item.discount
	}));
	
	// Calculate subtotal (sum of all item subtotals)
	$: subtotal = itemSubtotals.reduce((sum, item) => sum + Math.max(0, item.subtotal), 0);
	
	// Calculate total fees
	$: totalFees = fees.reduce((sum, fee) => sum + (fee.amount || 0), 0);
	
	// Calculate grand total
	$: grandTotal = subtotal + totalFees;
	
	// Form validity
	$: isFormValid = shopName.trim().length > 0 && items.length > 0 && items.every(item => 
		item.name.trim().length > 0 && item.quantity > 0 && item.unitPrice >= 0
	);
	
	// ============ Helper Functions ============
	
	function generateId(): string {
		return `temp-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
	}
	
	function formatCurrency(amount: number, curr: string = currency): string {
		if (isNaN(amount)) return '-';
		
		const locale = curr === 'IDR' ? 'id-ID' : 'en-US';
		const fractionDigits = curr === 'IDR' || curr === 'JPY' ? 0 : 2;
		
		return new Intl.NumberFormat(locale, {
			style: 'currency',
			currency: curr,
			minimumFractionDigits: fractionDigits,
			maximumFractionDigits: fractionDigits
		}).format(amount);
	}
	
	function formatNumberInput(value: number): string {
		if (isNaN(value) || value === 0) return '';
		return value.toString();
	}
	
	function parseNumberInput(value: string): number {
		const parsed = parseFloat(value);
		return isNaN(parsed) ? 0 : parsed;
	}
	
	// ============ Item Management ============
	
	function addItem() {
		items = [...items, {
			id: generateId(),
			name: '',
			quantity: 1,
			unitPrice: 0,
			discount: 0
		}];
	}
	
	function removeItem(id: string) {
		items = items.filter(item => item.id !== id);
	}
	
	function updateItem(id: string, field: string, value: string | number) {
		items = items.map(item => {
			if (item.id === id) {
				return { ...item, [field]: value };
			}
			return item;
		});
	}
	
	// ============ Fee Management ============
	
	function addFee() {
		fees = [...fees, {
			id: generateId(),
			label: '',
			type: 'tax',
			amount: 0
		}];
	}
	
	function removeFee(id: string) {
		fees = fees.filter(fee => fee.id !== id);
	}
	
	function updateFee(id: string, field: string, value: string | number) {
		fees = fees.map(fee => {
			if (fee.id === id) {
				return { ...fee, [field]: value };
			}
			return fee;
		});
	}
	
	// ============ Tag Management ============
	
	function toggleTag(tagId: string) {
		if (selectedTagIds.includes(tagId)) {
			selectedTagIds = selectedTagIds.filter(id => id !== tagId);
		} else {
			selectedTagIds = [...selectedTagIds, tagId];
		}
	}
	
	function getTagColorClass(color: string | undefined): string {
		if (!color) return 'bg-gray-100 text-gray-700 border-gray-300';
		
		const colorMap: Record<string, string> = {
			'#ef4444': 'bg-red-100 text-red-700 border-red-300',
			'#f97316': 'bg-orange-100 text-orange-700 border-orange-300',
			'#f59e0b': 'bg-amber-100 text-amber-700 border-amber-300',
			'#84cc16': 'bg-lime-100 text-lime-700 border-lime-300',
			'#22c55e': 'bg-green-100 text-green-700 border-green-300',
			'#10b981': 'bg-emerald-100 text-emerald-700 border-emerald-300',
			'#14b8a6': 'bg-teal-100 text-teal-700 border-teal-300',
			'#06b6d4': 'bg-cyan-100 text-cyan-700 border-cyan-300',
			'#0ea5e9': 'bg-sky-100 text-sky-700 border-sky-300',
			'#3b82f6': 'bg-blue-100 text-blue-700 border-blue-300',
			'#6366f1': 'bg-indigo-100 text-indigo-700 border-indigo-300',
			'#8b5cf6': 'bg-violet-100 text-violet-700 border-violet-300',
			'#a855f7': 'bg-purple-100 text-purple-700 border-purple-300',
			'#d946ef': 'bg-fuchsia-100 text-fuchsia-700 border-fuchsia-300',
			'#ec4899': 'bg-pink-100 text-pink-700 border-pink-300',
			'#f43f5e': 'bg-rose-100 text-rose-700 border-rose-300',
			'#6b7280': 'bg-gray-100 text-gray-700 border-gray-300'
		};
		
		return colorMap[color] || 'bg-gray-100 text-gray-700 border-gray-300';
	}
	
	// ============ Validation ============
	
	function validateForm(): boolean {
		formErrors = {};
		
		if (!shopName.trim()) {
			formErrors.shopName = 'Shop name is required';
		}
		
		if (!receiptDate) {
			formErrors.receiptDate = 'Date is required';
		}
		
		if (items.length === 0) {
			formErrors.items = 'At least one item is required';
		}
		
		items.forEach((item, index) => {
			if (!item.name.trim()) {
				formErrors[`item_${index}_name`] = 'Item name is required';
			}
			if (item.quantity <= 0) {
				formErrors[`item_${index}_quantity`] = 'Quantity must be at least 1';
			}
			if (item.unitPrice < 0) {
				formErrors[`item_${index}_price`] = 'Price cannot be negative';
			}
		});
		
		return Object.keys(formErrors).length === 0;
	}
	
	// ============ Form Actions ============
	
	async function handleSubmit() {
		if (!validateForm()) {
			toastStore.error('Please fix the errors in the form');
			return;
		}
		
		isSubmitting = true;
		
		try {
			// Prepare items data
			const receiptItems: ReceiptItem[] = items.map(item => ({
				name: item.name.trim(),
				quantity: item.quantity,
				unit_price: item.unitPrice,
				total_price: item.quantity * item.unitPrice - item.discount,
				category: undefined
			}));
			
			// Prepare fees data
			const receiptFees: ReceiptFee[] = fees.map(fee => ({
				name: fee.label.trim() || fee.type,
				amount: fee.amount
			}));
			
			const receiptData: CreateReceiptRequest = {
				title: shopName.trim(),
				currency: currency,
				total: grandTotal,
				receipt_date: receiptDate,
				payment_method: paymentMethod,
				notes: notes.trim() || undefined,
				subtotal: subtotal,
				items: receiptItems,
				fees: receiptFees,
				tags: selectedTagIds,
				source: 'manual'
			};
			
			if (isEditMode && receiptId) {
				// Update existing receipt
				await api.receipts.update(receiptId, receiptData);
				toastStore.success('Receipt updated successfully');
				goto(`/receipts/${receiptId}`);
			} else {
				// Create new receipt
				const newReceipt = await api.receipts.create(receiptData);
				toastStore.success('Receipt created successfully');
				goto(`/receipts/${newReceipt.id}`);
			}
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to save receipt';
			toastStore.error(message);
			console.error('Error saving receipt:', err);
		} finally {
			isSubmitting = false;
		}
	}
	
	async function handleDelete() {
		if (!isEditMode || !receiptId) return;
		
		if (!confirm('Are you sure you want to delete this receipt? This action cannot be undone.')) {
			return;
		}
		
		isSubmitting = true;
		
		try {
			await api.receipts.delete(receiptId);
			toastStore.success('Receipt deleted successfully');
			goto('/receipts');
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to delete receipt';
			toastStore.error(message);
			console.error('Error deleting receipt:', err);
		} finally {
			isSubmitting = false;
		}
	}
	
	function handleCancel() {
		if (isEditMode && receiptId) {
			goto(`/receipts/${receiptId}`);
		} else {
			goto('/receipts');
		}
	}
	
	// ============ Data Fetching ============
	
	async function fetchTags() {
		isLoadingTags = true;
		try {
			availableTags = await api.tags.list();
		} catch (err) {
			console.error('Failed to fetch tags:', err);
			toastStore.error('Failed to load tags');
		} finally {
			isLoadingTags = false;
		}
	}
	
	async function loadReceiptForEdit(id: string) {
		isLoadingReceipt = true;
		
		try {
			const receipt = await api.receipts.get(id);
			
			// Populate form data
			shopName = receipt.title;
			receiptDate = receipt.receipt_date || receipt.created_at.split('T')[0];
			currency = receipt.currency;
			paymentMethod = receipt.payment_method || 'cash';
			notes = receipt.notes || '';
			selectedTagIds = receipt.tags || [];
			
			// Populate items
			items = (receipt.items || []).map(item => ({
				id: item.id || generateId(),
				name: item.name,
				quantity: item.quantity,
				unitPrice: item.unit_price,
				discount: (item.quantity * item.unit_price) - item.total_price
			}));
			
			// Populate fees
			fees = (receipt.fees || []).map(fee => ({
				id: fee.id || generateId(),
				label: fee.name,
				type: 'other',
				amount: fee.amount
			}));
			
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to load receipt';
			toastStore.error(message);
			goto('/receipts');
		} finally {
			isLoadingReceipt = false;
		}
	}
	
	// ============ Lifecycle ============
	
	onMount(() => {
		fetchTags();
		
		// Check if we're in edit mode
		const path = $page.url.pathname;
		const match = path.match(/\/receipts\/([^/]+)\/edit/);
		
		if (match) {
			isEditMode = true;
			receiptId = match[1];
			loadReceiptForEdit(receiptId);
		} else {
			// Add one empty item for new receipts
			addItem();
		}
	});
</script>

<div class="p-4 md:p-8 max-w-5xl mx-auto">
	<!-- Loading State -->
	{#if isLoadingReceipt}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="w-8 h-8 text-primary animate-spin" />
			<span class="ml-3 text-gray-600">Loading receipt...</span>
		</div>
	{:else}
		<!-- Header -->
		<div class="mb-8 flex items-center justify-between">
			<div class="flex items-center gap-4">
				<button
					on:click={handleCancel}
					class="p-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg transition-colors"
					disabled={isSubmitting}
				>
					<ArrowLeft class="w-5 h-5" />
				</button>
				<div>
					<h1 class="text-2xl md:text-3xl font-bold text-gray-900">
						{isEditMode ? 'Edit Receipt' : 'New Receipt'}
					</h1>
					<p class="text-gray-600 mt-1">
						{isEditMode ? 'Update receipt details' : 'Create a new receipt manually'}
					</p>
				</div>
			</div>
			
			{#if isEditMode}
				<button
					on:click={handleDelete}
					disabled={isSubmitting}
					class="flex items-center gap-2 px-4 py-2 text-red-600 hover:text-red-700 hover:bg-red-50 rounded-lg transition-colors"
				>
					<Trash2 class="w-4 h-4" />
					<span class="hidden sm:inline">Delete</span>
				</button>
			{/if}
		</div>
		
		<!-- Form -->
		<form on:submit|preventDefault={handleSubmit} class="space-y-8">
			<!-- Basic Information Section -->
			<section class="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
				<div class="px-6 py-4 border-b border-gray-200 bg-gray-50">
					<div class="flex items-center gap-2">
						<ReceiptIcon class="w-5 h-5 text-primary" />
						<h2 class="text-lg font-semibold text-gray-900">Basic Information</h2>
					</div>
				</div>
				
				<div class="p-6 space-y-6">
					<!-- Shop Name -->
					<div>
						<label for="shop-name" class="block text-sm font-medium text-gray-700 mb-2">
							Shop Name <span class="text-red-500">*</span>
						</label>
						<input
							type="text"
							id="shop-name"
							bind:value={shopName}
							placeholder="e.g., Starbucks, Amazon, Local Market"
							class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-colors"
							class:border-red-500={formErrors.shopName}
							disabled={isSubmitting}
						/>
						{#if formErrors.shopName}
							<p class="mt-1 text-sm text-red-600 flex items-center gap-1">
								<AlertCircle class="w-4 h-4" />
								{formErrors.shopName}
							</p>
						{/if}
					</div>
					
					<!-- Date and Currency Row -->
					<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
						<!-- Date -->
						<div>
							<label for="receipt-date" class="block text-sm font-medium text-gray-700 mb-2">
								Date <span class="text-red-500">*</span>
							</label>
							<input
								type="date"
								id="receipt-date"
								bind:value={receiptDate}
								class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-colors"
								class:border-red-500={formErrors.receiptDate}
								disabled={isSubmitting}
							/>
							{#if formErrors.receiptDate}
								<p class="mt-1 text-sm text-red-600 flex items-center gap-1">
									<AlertCircle class="w-4 h-4" />
									{formErrors.receiptDate}
								</p>
							{/if}
						</div>
						
						<!-- Currency -->
						<div>
							<label for="currency" class="block text-sm font-medium text-gray-700 mb-2">
								Currency
							</label>
							<select
								id="currency"
								bind:value={currency}
								class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-colors"
								disabled={isSubmitting}
							>
								{#each currencies as curr}
									<option value={curr.value}>{curr.label}</option>
								{/each}
							</select>
						</div>
					</div>
					
					<!-- Payment Method -->
					<div>
						<label for="payment-method" class="block text-sm font-medium text-gray-700 mb-2">
							Payment Method
						</label>
						<select
							id="payment-method"
							bind:value={paymentMethod}
							class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-colors"
							disabled={isSubmitting}
						>
							{#each paymentMethods as method}
								<option value={method.value}>{method.label}</option>
							{/each}
						</select>
					</div>
					
					<!-- Notes -->
					<div>
						<label for="notes" class="block text-sm font-medium text-gray-700 mb-2">
							Notes <span class="text-gray-400 font-normal">(Optional)</span>
						</label>
						<textarea
							id="notes"
							bind:value={notes}
							placeholder="Add any additional notes about this receipt..."
							rows="3"
							class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-colors resize-none"
							disabled={isSubmitting}
						></textarea>
					</div>
					
					<!-- Tags -->
					<div>
						<label class="block text-sm font-medium text-gray-700 mb-2 flex items-center gap-2">
							<TagIcon class="w-4 h-4" />
							Tags <span class="text-gray-400 font-normal">(Optional)</span>
						</label>
						
						{#if isLoadingTags}
							<div class="flex items-center gap-2 text-gray-500">
								<Loader2 class="w-4 h-4 animate-spin" />
								<span class="text-sm">Loading tags...</span>
							</div>
						{:else if availableTags.length === 0}
							<p class="text-sm text-gray-500">No tags available. Create tags in settings.</p>
						{:else}
							<div class="flex flex-wrap gap-2">
								{#each availableTags as tag}
									<button
										type="button"
										on:click={() => toggleTag(tag.id)}
										disabled={isSubmitting}
										class="inline-flex items-center px-3 py-1.5 rounded-full text-sm font-medium border transition-all {selectedTagIds.includes(tag.id) ? getTagColorClass(tag.color) : 'bg-white text-gray-600 border-gray-300 hover:bg-gray-50'}"
									>
										{tag.name}
										{#if selectedTagIds.includes(tag.id)}
											<X class="w-3 h-3 ml-1.5" />
										{/if}
									</button>
								{/each}
							</div>
						{/if}
					</div>
				</div>
			</section>
			
			<!-- Items Section -->
			<section class="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
				<div class="px-6 py-4 border-b border-gray-200 bg-gray-50">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-2">
							<Calculator class="w-5 h-5 text-primary" />
							<h2 class="text-lg font-semibold text-gray-900">Items</h2>
						</div>
						<button
							type="button"
							on:click={addItem}
							disabled={isSubmitting}
							class="flex items-center gap-2 px-4 py-2 text-sm font-medium text-primary hover:bg-primary/10 rounded-lg transition-colors"
						>
							<Plus class="w-4 h-4" />
							Add Item
						</button>
					</div>
				</div>
				
				<div class="p-6">
					{#if formErrors.items}
						<div class="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg flex items-center gap-2 text-red-700">
							<AlertCircle class="w-4 h-4" />
							<span class="text-sm">{formErrors.items}</span>
						</div>
					{/if}
					
					{#if items.length === 0}
						<div class="text-center py-8 text-gray-500">
							<p>No items added yet.</p>
							<button
								type="button"
								on:click={addItem}
								disabled={isSubmitting}
								class="mt-2 text-primary hover:underline"
							>
								Add your first item
							</button>
						</div>
					{:else}
						<div class="space-y-4">
							{#each items as item, index (item.id)}
								<div class="p-4 border border-gray-200 rounded-lg bg-gray-50/50">
									<div class="grid grid-cols-1 md:grid-cols-12 gap-4">
										<!-- Item Name -->
										<div class="md:col-span-4">
											<label class="block text-xs font-medium text-gray-600 mb-1">Name</label>
											<input
												type="text"
												value={item.name}
												on:input={(e) => updateItem(item.id, 'name', e.currentTarget.value)}
												placeholder="Item name"
												class="w-full px-3 py-2 text-sm border border-gray-200 rounded-md focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
												class:border-red-500={formErrors[`item_${index}_name`]}
												disabled={isSubmitting}
											/>
											{#if formErrors[`item_${index}_name`]}
												<p class="mt-1 text-xs text-red-600">{formErrors[`item_${index}_name`]}</p>
											{/if}
										</div>
										
										<!-- Quantity -->
										<div class="md:col-span-2">
											<label class="block text-xs font-medium text-gray-600 mb-1">Qty</label>
											<input
												type="number"
												value={item.quantity}
												on:input={(e) => updateItem(item.id, 'quantity', parseInt(e.currentTarget.value) || 0)}
												min="1"
												class="w-full px-3 py-2 text-sm border border-gray-200 rounded-md focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
												class:border-red-500={formErrors[`item_${index}_quantity`]}
												disabled={isSubmitting}
											/>
											{#if formErrors[`item_${index}_quantity`]}
												<p class="mt-1 text-xs text-red-600">{formErrors[`item_${index}_quantity`]}</p>
											{/if}
										</div>
										
										<!-- Unit Price -->
										<div class="md:col-span-2">
											<label class="block text-xs font-medium text-gray-600 mb-1">Unit Price</label>
											<input
												type="number"
												value={formatNumberInput(item.unitPrice)}
												on:input={(e) => updateItem(item.id, 'unitPrice', parseNumberInput(e.currentTarget.value))}
												min="0"
												step="0.01"
												placeholder="0.00"
												class="w-full px-3 py-2 text-sm border border-gray-200 rounded-md focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
												class:border-red-500={formErrors[`item_${index}_price`]}
												disabled={isSubmitting}
											/>
											{#if formErrors[`item_${index}_price`]}
												<p class="mt-1 text-xs text-red-600">{formErrors[`item_${index}_price`]}</p>
											{/if}
										</div>
										
										<!-- Discount -->
										<div class="md:col-span-2">
											<label class="block text-xs font-medium text-gray-600 mb-1">Discount</label>
											<input
												type="number"
												value={formatNumberInput(item.discount)}
												on:input={(e) => updateItem(item.id, 'discount', parseNumberInput(e.currentTarget.value))}
												min="0"
												step="0.01"
												placeholder="0.00"
												class="w-full px-3 py-2 text-sm border border-gray-200 rounded-md focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
												disabled={isSubmitting}
											/>
										</div>
										
										<!-- Subtotal and Remove -->
										<div class="md:col-span-2 flex items-end justify-between gap-2">
											<div>
												<label class="block text-xs font-medium text-gray-600 mb-1">Subtotal</label>
												<div class="text-sm font-medium text-gray-900 py-2">
													{formatCurrency(itemSubtotals.find(i => i.id === item.id)?.subtotal || 0)}
												</div>
											</div>
											<button
												type="button"
												on:click={() => removeItem(item.id)}
												disabled={isSubmitting || items.length === 1}
												class="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-md transition-colors disabled:opacity-50"
												title="Remove item"
											>
												<Trash2 class="w-4 h-4" />
											</button>
										</div>
									</div>
								</div>
							{/each}
						</div>
						
						<!-- Items Subtotal -->
						<div class="mt-4 pt-4 border-t border-gray-200 flex justify-between items-center">
							<span class="text-sm font-medium text-gray-600">Items Subtotal:</span>
							<span class="text-lg font-semibold text-gray-900">{formatCurrency(subtotal)}</span>
						</div>
					{/if}
				</div>
			</section>
			
			<!-- Fees Section -->
			<section class="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
				<div class="px-6 py-4 border-b border-gray-200 bg-gray-50">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-2">
							<Calculator class="w-5 h-5 text-primary" />
							<h2 class="text-lg font-semibold text-gray-900">Fees & Taxes</h2>
						</div>
						<button
							type="button"
							on:click={addFee}
							disabled={isSubmitting}
							class="flex items-center gap-2 px-4 py-2 text-sm font-medium text-primary hover:bg-primary/10 rounded-lg transition-colors"
						>
							<Plus class="w-4 h-4" />
							Add Fee
						</button>
					</div>
				</div>
				
				<div class="p-6">
					{#if fees.length === 0}
						<div class="text-center py-6 text-gray-500">
							<p class="text-sm">No additional fees or taxes.</p>
						</div>
					{:else}
						<div class="space-y-3">
							{#each fees as fee (fee.id)}
								<div class="grid grid-cols-1 md:grid-cols-12 gap-3 items-end">
									<!-- Fee Label -->
									<div class="md:col-span-4">
										<input
											type="text"
											value={fee.label}
											on:input={(e) => updateFee(fee.id, 'label', e.currentTarget.value)}
											placeholder="e.g., PPN 11%, Service Charge"
											class="w-full px-3 py-2 text-sm border border-gray-200 rounded-md focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
											disabled={isSubmitting}
										/>
									</div>
									
									<!-- Fee Type -->
									<div class="md:col-span-3">
										<select
											value={fee.type}
											on:change={(e) => updateFee(fee.id, 'type', e.currentTarget.value)}
											class="w-full px-3 py-2 text-sm border border-gray-200 rounded-md focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
											disabled={isSubmitting}
										>
											{#each feeTypes as type}
												<option value={type.value}>{type.label}</option>
											{/each}
										</select>
									</div>
									
									<!-- Fee Amount -->
									<div class="md:col-span-3">
										<input
											type="number"
											value={formatNumberInput(fee.amount)}
											on:input={(e) => updateFee(fee.id, 'amount', parseNumberInput(e.currentTarget.value))}
											min="0"
											step="0.01"
											placeholder="0.00"
											class="w-full px-3 py-2 text-sm border border-gray-200 rounded-md focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
											disabled={isSubmitting}
										/>
									</div>
									
									<!-- Remove Button -->
									<div class="md:col-span-2">
										<button
											type="button"
											on:click={() => removeFee(fee.id)}
											disabled={isSubmitting}
											class="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-md transition-colors"
											title="Remove fee"
										>
											<Trash2 class="w-4 h-4" />
										</button>
									</div>
								</div>
							{/each}
							
							<!-- Fees Total -->
							<div class="mt-4 pt-3 border-t border-gray-200 flex justify-between items-center">
								<span class="text-sm font-medium text-gray-600">Total Fees:</span>
								<span class="text-base font-semibold text-gray-900">{formatCurrency(totalFees)}</span>
							</div>
						</div>
					{/if}
				</div>
			</section>
			
			<!-- Totals Section -->
			<section class="bg-primary/5 rounded-xl border-2 border-primary/20 p-6">
				<div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
					<div class="space-y-2">
						<div class="flex justify-between md:justify-start md:gap-8 text-sm">
							<span class="text-gray-600">Subtotal:</span>
							<span class="font-medium text-gray-900">{formatCurrency(subtotal)}</span>
						</div>
						<div class="flex justify-between md:justify-start md:gap-8 text-sm">
							<span class="text-gray-600">Fees & Taxes:</span>
							<span class="font-medium text-gray-900">{formatCurrency(totalFees)}</span>
						</div>
					</div>
					
					<div class="flex items-center gap-3 pt-4 md:pt-0 border-t md:border-t-0 border-primary/20">
						<span class="text-lg font-medium text-gray-700">Grand Total:</span>
						<span class="text-2xl md:text-3xl font-bold text-primary">
							{formatCurrency(grandTotal)}
						</span>
					</div>
				</div>
			</section>
			
			<!-- Action Buttons -->
			<div class="flex flex-col sm:flex-row gap-4 pt-4">
				<button
					type="submit"
					disabled={isSubmitting || !isFormValid}
					class="flex-1 sm:flex-none flex items-center justify-center gap-2 px-8 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed font-medium"
				>
					{#if isSubmitting}
						<Loader2 class="w-5 h-5 animate-spin" />
						<span>Saving...</span>
					{:else}
						<Save class="w-5 h-5" />
						<span>{isEditMode ? 'Update Receipt' : 'Save Receipt'}</span>
					{/if}
				</button>
				
				<button
					type="button"
					on:click={handleCancel}
					disabled={isSubmitting}
					class="flex items-center justify-center gap-2 px-6 py-3 border border-gray-200 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors font-medium"
				>
					<X class="w-5 h-5" />
					<span>Cancel</span>
				</button>
			</div>
		</form>
	{/if}
</div>
