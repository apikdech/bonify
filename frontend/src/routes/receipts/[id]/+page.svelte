<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { api, type Receipt, type Tag } from '$lib/api';
	import { toastStore } from '$lib/stores';
	import {
		ArrowLeft,
		Edit,
		Trash2,
		Check,
		X,
		Receipt as ReceiptIcon,
		Image as ImageIcon,
		Tag as TagIcon,
		Plus,
		Bot,
		RotateCcw,
		Loader2,
		AlertCircle,
		FileText,
		MessageCircle,
		Smartphone,
		Calendar,
		CreditCard,
		Coins,
		CheckCircle,
		XCircle,
		Clock,
		AlertTriangle
	} from 'lucide-svelte';

	// ============ State ============
	
	// Data
	let receipt: Receipt | null = null;
	let allTags: Tag[] = [];
	
	// Loading states
	let isLoading = true;
	let isLoadingTags = false;
	let isProcessing = false;
	
	// Error state
	let error: string | null = null;
	
	// UI state
	let isEditMode = false;
	let showDeleteConfirm = false;
	let showTagDropdown = false;
	let tagDropdownRef: HTMLDivElement;
	
	// Edit form state (reused from new receipt page)
	let editShopName = '';
	let editReceiptDate = '';
	let editCurrency = 'IDR';
	let editPaymentMethod = 'cash';
	let editNotes = '';
	let editSelectedTagIds: string[] = [];
	let editItems: Array<{
		id: string;
		name: string;
		quantity: number;
		unitPrice: number;
		discount: number;
	}> = [];
	let editFees: Array<{
		id: string;
		label: string;
		type: 'tax' | 'service' | 'delivery' | 'tip' | 'other';
		amount: number;
	}> = [];
	let editFormErrors: Record<string, string> = {};
	
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
	
	$: receiptId = $page.params.id;
	
	// Edit mode calculations
	$: editItemSubtotals = editItems.map(item => ({
		...item,
		subtotal: item.quantity * item.unitPrice - item.discount
	}));
	$: editSubtotal = editItemSubtotals.reduce((sum, item) => sum + Math.max(0, item.subtotal), 0);
	$: editTotalFees = editFees.reduce((sum, fee) => sum + (fee.amount || 0), 0);
	$: editGrandTotal = editSubtotal + editTotalFees;
	
	// ============ Helper Functions ============
	
	function formatCurrency(amount: number, currency: string = 'IDR'): string {
		const locale = currency === 'IDR' ? 'id-ID' : 'en-US';
		const fractionDigits = currency === 'IDR' || currency === 'JPY' ? 0 : 2;
		
		return new Intl.NumberFormat(locale, {
			style: 'currency',
			currency: currency,
			minimumFractionDigits: fractionDigits,
			maximumFractionDigits: fractionDigits
		}).format(amount);
	}
	
	function formatDate(dateString: string | undefined): string {
		if (!dateString) return '-';
		const date = new Date(dateString);
		return date.toLocaleDateString('en-US', {
			weekday: 'short',
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	}
	
	function getStatusColor(status: string): string {
		switch (status) {
			case 'confirmed':
				return 'bg-green-100 text-green-800 border-green-200';
			case 'pending':
				return 'bg-yellow-100 text-yellow-800 border-yellow-200';
			case 'rejected':
				return 'bg-red-100 text-red-800 border-red-200';
			default:
				return 'bg-gray-100 text-gray-800 border-gray-200';
		}
	}
	
	function getStatusIcon(status: string) {
		switch (status) {
			case 'confirmed':
				return CheckCircle;
			case 'pending':
				return Clock;
			case 'rejected':
				return XCircle;
			default:
				return AlertCircle;
		}
	}
	
	function getStatusLabel(status: string): string {
		switch (status) {
			case 'confirmed':
				return 'Confirmed';
			case 'pending':
				return 'Pending Review';
			case 'rejected':
				return 'Rejected';
			default:
				return status;
		}
	}
	
	function getSourceIcon(source: string) {
		switch (source) {
			case 'telegram':
				return MessageCircle;
			case 'discord':
				return Smartphone;
			case 'manual':
				return FileText;
			case 'ocr':
				return Bot;
			default:
				return FileText;
		}
	}
	
	function getSourceLabel(source: string): string {
		switch (source) {
			case 'manual':
				return 'Manual Entry';
			case 'ocr':
				return 'OCR Scan';
			case 'telegram':
				return 'Telegram Bot';
			case 'discord':
				return 'Discord Bot';
			default:
				return source;
		}
	}
	
	function getTagById(tagId: string): Tag | undefined {
		return allTags.find(t => t.id === tagId);
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
	
	function getConfidenceColor(score: number): string {
		if (score >= 0.9) return 'bg-green-100 text-green-800';
		if (score >= 0.7) return 'bg-yellow-100 text-yellow-800';
		return 'bg-red-100 text-red-800';
	}
	
	function generateId(): string {
		return `temp-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
	}
	
	function formatNumberInput(value: number): string {
		if (isNaN(value) || value === 0) return '';
		return value.toString();
	}
	
	function parseNumberInput(value: string): number {
		const parsed = parseFloat(value);
		return isNaN(parsed) ? 0 : parsed;
	}
	
	// ============ Data Fetching ============
	
	async function fetchReceipt() {
		isLoading = true;
		error = null;
		
		try {
			receipt = await api.receipts.get(receiptId);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load receipt';
			toastStore.error('Failed to load receipt');
		} finally {
			isLoading = false;
		}
	}
	
	async function fetchTags() {
		isLoadingTags = true;
		try {
			allTags = await api.tags.list();
		} catch (err) {
			console.error('Failed to fetch tags:', err);
		} finally {
			isLoadingTags = false;
		}
	}
	
	// ============ Action Handlers ============
	
	async function handleConfirm() {
		if (!receipt) return;
		
		isProcessing = true;
		try {
			await api.receipts.confirm(receipt.id);
			await fetchReceipt();
			toastStore.success('Receipt confirmed successfully');
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to confirm receipt';
			toastStore.error(message);
		} finally {
			isProcessing = false;
		}
	}
	
	async function handleReject() {
		if (!receipt) return;
		
		isProcessing = true;
		try {
			await api.receipts.reject(receipt.id);
			await fetchReceipt();
			toastStore.success('Receipt rejected');
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to reject receipt';
			toastStore.error(message);
		} finally {
			isProcessing = false;
		}
	}
	
	async function handleDelete() {
		if (!receipt) return;
		
		isProcessing = true;
		try {
			await api.receipts.delete(receipt.id);
			toastStore.success('Receipt deleted successfully');
			goto('/receipts');
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to delete receipt';
			toastStore.error(message);
			isProcessing = false;
			showDeleteConfirm = false;
		}
	}
	
	async function handleAddTag(tagId: string) {
		if (!receipt) return;
		
		const newTags = [...receipt.tags, tagId];
		
		try {
			await api.receipts.update(receipt.id, { tags: newTags });
			await fetchReceipt();
			toastStore.success('Tag added');
			showTagDropdown = false;
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to add tag';
			toastStore.error(message);
		}
	}
	
	async function handleRemoveTag(tagId: string) {
		if (!receipt) return;
		
		const newTags = receipt.tags.filter(id => id !== tagId);
		
		try {
			await api.receipts.update(receipt.id, { tags: newTags });
			await fetchReceipt();
			toastStore.success('Tag removed');
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to remove tag';
			toastStore.error(message);
		}
	}
	
	function handleRerunOCR() {
		toastStore.info('OCR reprocessing is not yet implemented');
	}
	
	// ============ Edit Mode Functions ============
	
	function enterEditMode() {
		if (!receipt) return;
		
		// Populate edit form with current receipt data
		editShopName = receipt.title;
		editReceiptDate = receipt.receipt_date || receipt.created_at.split('T')[0];
		editCurrency = receipt.currency;
		editPaymentMethod = receipt.payment_method || 'cash';
		editNotes = receipt.notes || '';
		editSelectedTagIds = [...receipt.tags];
		
		// Populate items
		editItems = (receipt.items || []).map(item => ({
			id: item.id || generateId(),
			name: item.name,
			quantity: item.quantity,
			unitPrice: item.unit_price,
			discount: (item.quantity * item.unit_price) - item.total_price
		}));
		
		// Populate fees
		editFees = (receipt.fees || []).map(fee => ({
			id: fee.id || generateId(),
			label: fee.name,
			type: 'other',
			amount: fee.amount
		}));
		
		isEditMode = true;
	}
	
	function cancelEditMode() {
		isEditMode = false;
		editFormErrors = {};
	}
	
	// Edit mode item management
	function addEditItem() {
		editItems = [...editItems, {
			id: generateId(),
			name: '',
			quantity: 1,
			unitPrice: 0,
			discount: 0
		}];
	}
	
	function removeEditItem(id: string) {
		editItems = editItems.filter(item => item.id !== id);
	}
	
	function updateEditItem(id: string, field: string, value: string | number) {
		editItems = editItems.map(item => {
			if (item.id === id) {
				return { ...item, [field]: value };
			}
			return item;
		});
	}
	
	// Edit mode fee management
	function addEditFee() {
		editFees = [...editFees, {
			id: generateId(),
			label: '',
			type: 'tax',
			amount: 0
		}];
	}
	
	function removeEditFee(id: string) {
		editFees = editFees.filter(fee => fee.id !== id);
	}
	
	function updateEditFee(id: string, field: string, value: string | number) {
		editFees = editFees.map(fee => {
			if (fee.id === id) {
				return { ...fee, [field]: value };
			}
			return fee;
		});
	}
	
	// Edit mode tag management
	function toggleEditTag(tagId: string) {
		if (editSelectedTagIds.includes(tagId)) {
			editSelectedTagIds = editSelectedTagIds.filter(id => id !== tagId);
		} else {
			editSelectedTagIds = [...editSelectedTagIds, tagId];
		}
	}
	
	function validateEditForm(): boolean {
		editFormErrors = {};
		
		if (!editShopName.trim()) {
			editFormErrors.shopName = 'Shop name is required';
		}
		
		if (!editReceiptDate) {
			editFormErrors.receiptDate = 'Date is required';
		}
		
		if (editItems.length === 0) {
			editFormErrors.items = 'At least one item is required';
		}
		
		editItems.forEach((item, index) => {
			if (!item.name.trim()) {
				editFormErrors[`item_${index}_name`] = 'Item name is required';
			}
			if (item.quantity <= 0) {
				editFormErrors[`item_${index}_quantity`] = 'Quantity must be at least 1';
			}
			if (item.unitPrice < 0) {
				editFormErrors[`item_${index}_price`] = 'Price cannot be negative';
			}
		});
		
		return Object.keys(editFormErrors).length === 0;
	}
	
	async function handleEditSubmit() {
		if (!receipt || !validateEditForm()) {
			toastStore.error('Please fix the errors in the form');
			return;
		}
		
		isProcessing = true;
		
		try {
			// Prepare items data
			const receiptItems = editItems.map(item => ({
				name: item.name.trim(),
				quantity: item.quantity,
				unit_price: item.unitPrice,
				total_price: item.quantity * item.unitPrice - item.discount,
				category: undefined
			}));
			
			// Prepare fees data
			const receiptFees = editFees.map(fee => ({
				name: fee.label.trim() || fee.type,
				amount: fee.amount
			}));
			
			await api.receipts.update(receipt.id, {
				title: editShopName.trim(),
				currency: editCurrency,
				total: editGrandTotal,
				receipt_date: editReceiptDate,
				payment_method: editPaymentMethod,
				notes: editNotes.trim() || undefined,
				subtotal: editSubtotal,
				items: receiptItems,
				fees: receiptFees,
				tags: editSelectedTagIds
			});
			
			await fetchReceipt();
			isEditMode = false;
			toastStore.success('Receipt updated successfully');
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to update receipt';
			toastStore.error(message);
		} finally {
			isProcessing = false;
		}
	}
	
	// ============ Lifecycle ============
	
	onMount(() => {
		fetchReceipt();
		fetchTags();
		
		// Close tag dropdown when clicking outside
		const handleClickOutside = (event: MouseEvent) => {
			if (tagDropdownRef && !tagDropdownRef.contains(event.target as Node)) {
				showTagDropdown = false;
			}
		};
		
		document.addEventListener('click', handleClickOutside);
		
		return () => {
			document.removeEventListener('click', handleClickOutside);
		};
	});
</script>

<div class="p-4 md:p-8 max-w-5xl mx-auto">
	<!-- Loading State -->
	{#if isLoading}
		<div class="flex items-center justify-center py-20">
			<Loader2 class="w-8 h-8 text-primary animate-spin" />
			<span class="ml-3 text-gray-600">Loading receipt...</span>
		</div>
	{:else if error}
		<!-- Error State -->
		<div class="bg-red-50 border border-red-200 rounded-lg p-6 text-center">
			<AlertCircle class="w-12 h-12 text-red-600 mx-auto mb-3" />
			<h3 class="text-lg font-medium text-red-900 mb-2">Failed to load receipt</h3>
			<p class="text-red-700 mb-4">{error}</p>
			<button
				on:click={fetchReceipt}
				class="inline-flex items-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors"
			>
				<RotateCcw class="w-4 h-4" />
				Retry
			</button>
		</div>
	{:else if receipt}
		<!-- Header -->
		<div class="mb-8">
			<div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4 mb-4">
				<div class="flex items-center gap-4">
					<button
						on:click={() => goto('/receipts')}
						class="p-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg transition-colors"
					>
						<ArrowLeft class="w-5 h-5" />
					</button>
					<div>
						<div class="flex items-center gap-3 flex-wrap">
							<h1 class="text-2xl md:text-3xl font-bold text-gray-900">
								{receipt.title || 'Untitled Receipt'}
							</h1>
							<!-- Status Badge -->
							<span class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-sm font-medium border {getStatusColor(receipt.status)}">
								<svelte:component this={getStatusIcon(receipt.status)} class="w-4 h-4" />
								{getStatusLabel(receipt.status)}
							</span>
							<!-- Source Badge -->
							<span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium bg-gray-100 text-gray-700">
								<svelte:component this={getSourceIcon(receipt.source)} class="w-3.5 h-3.5" />
								{getSourceLabel(receipt.source)}
							</span>
						</div>
					</div>
				</div>
				
				{#if !isEditMode}
					<div class="flex items-center gap-2">
						<button
							on:click={enterEditMode}
							disabled={isProcessing}
							class="flex items-center gap-2 px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
						>
							<Edit class="w-4 h-4" />
							<span>Edit</span>
						</button>
						<button
							on:click={() => showDeleteConfirm = true}
							disabled={isProcessing}
							class="flex items-center gap-2 px-4 py-2 text-red-600 hover:bg-red-50 rounded-lg transition-colors"
						>
							<Trash2 class="w-4 h-4" />
							<span>Delete</span>
						</button>
					</div>
				{/if}
			</div>
		</div>
		
		<!-- Edit Mode -->
		{#if isEditMode}
			<!-- Edit Form -->
			<form on:submit|preventDefault={handleEditSubmit} class="space-y-6">
				<!-- Basic Information -->
				<section class="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
					<div class="px-6 py-4 border-b border-gray-200 bg-gray-50">
						<div class="flex items-center gap-2">
							<ReceiptIcon class="w-5 h-5 text-primary" />
							<h2 class="text-lg font-semibold text-gray-900">Edit Receipt</h2>
						</div>
					</div>
					
					<div class="p-6 space-y-6">
						<!-- Shop Name -->
						<div>
							<label for="edit-shop-name" class="block text-sm font-medium text-gray-700 mb-2">
								Shop Name <span class="text-red-500">*</span>
							</label>
							<input
								type="text"
								id="edit-shop-name"
								bind:value={editShopName}
								class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
								class:border-red-500={editFormErrors.shopName}
								disabled={isProcessing}
							/>
							{#if editFormErrors.shopName}
								<p class="mt-1 text-sm text-red-600">{editFormErrors.shopName}</p>
							{/if}
						</div>
						
						<!-- Date and Currency -->
						<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
							<div>
								<label for="edit-receipt-date" class="block text-sm font-medium text-gray-700 mb-2">
									Date <span class="text-red-500">*</span>
								</label>
								<input
									type="date"
									id="edit-receipt-date"
									bind:value={editReceiptDate}
									class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
									class:border-red-500={editFormErrors.receiptDate}
									disabled={isProcessing}
								/>
								{#if editFormErrors.receiptDate}
									<p class="mt-1 text-sm text-red-600">{editFormErrors.receiptDate}</p>
								{/if}
							</div>
							
							<div>
								<label for="edit-currency" class="block text-sm font-medium text-gray-700 mb-2">Currency</label>
								<select
									id="edit-currency"
									bind:value={editCurrency}
									class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
									disabled={isProcessing}
								>
									{#each currencies as curr}
										<option value={curr.value}>{curr.label}</option>
									{/each}
								</select>
							</div>
						</div>
						
						<!-- Payment Method -->
						<div>
							<label for="edit-payment-method" class="block text-sm font-medium text-gray-700 mb-2">Payment Method</label>
							<select
								id="edit-payment-method"
								bind:value={editPaymentMethod}
								class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
								disabled={isProcessing}
							>
								{#each paymentMethods as method}
									<option value={method.value}>{method.label}</option>
								{/each}
							</select>
						</div>
						
						<!-- Notes -->
						<div>
							<label for="edit-notes" class="block text-sm font-medium text-gray-700 mb-2">Notes</label>
							<textarea
								id="edit-notes"
								bind:value={editNotes}
								rows="3"
								class="w-full px-4 py-3 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary resize-none"
								disabled={isProcessing}
							></textarea>
						</div>
						
						<!-- Tags -->
						<div>
							<label class="block text-sm font-medium text-gray-700 mb-2 flex items-center gap-2">
								<TagIcon class="w-4 h-4" />
								Tags
							</label>
							{#if isLoadingTags}
								<div class="flex items-center gap-2 text-gray-500">
									<Loader2 class="w-4 h-4 animate-spin" />
									<span class="text-sm">Loading tags...</span>
								</div>
							{:else if allTags.length === 0}
								<p class="text-sm text-gray-500">No tags available.</p>
							{:else}
								<div class="flex flex-wrap gap-2">
									{#each allTags as tag}
										<button
											type="button"
											on:click={() => toggleEditTag(tag.id)}
											disabled={isProcessing}
											class="inline-flex items-center px-3 py-1.5 rounded-full text-sm font-medium border transition-all {editSelectedTagIds.includes(tag.id) ? getTagColorClass(tag.color) : 'bg-white text-gray-600 border-gray-300 hover:bg-gray-50'}"
										>
											{tag.name}
											{#if editSelectedTagIds.includes(tag.id)}
												<Check class="w-3 h-3 ml-1.5" />
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
					<div class="px-6 py-4 border-b border-gray-200 bg-gray-50 flex items-center justify-between">
						<h2 class="text-lg font-semibold text-gray-900">Items</h2>
						<button
							type="button"
							on:click={addEditItem}
							disabled={isProcessing}
							class="flex items-center gap-2 px-4 py-2 text-sm font-medium text-primary hover:bg-primary/10 rounded-lg transition-colors"
						>
							<Plus class="w-4 h-4" />
							Add Item
						</button>
					</div>
					
					<div class="p-6">
						{#if editFormErrors.items}
							<div class="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
								{editFormErrors.items}
							</div>
						{/if}
						
						{#if editItems.length === 0}
							<p class="text-center py-4 text-gray-500">No items. Click "Add Item" to add one.</p>
						{:else}
							<div class="space-y-4">
								{#each editItems as item, index (item.id)}
									<div class="p-4 border border-gray-200 rounded-lg bg-gray-50/50">
										<div class="grid grid-cols-1 md:grid-cols-12 gap-4">
											<div class="md:col-span-4">
												<label class="block text-xs font-medium text-gray-600 mb-1">Name</label>
												<input
													type="text"
													value={item.name}
													on:input={(e) => updateEditItem(item.id, 'name', e.currentTarget.value)}
													class="w-full px-3 py-2 text-sm border border-gray-200 rounded-md"
													class:border-red-500={editFormErrors[`item_${index}_name`]}
													disabled={isProcessing}
												/>
											</div>
											<div class="md:col-span-2">
												<label class="block text-xs font-medium text-gray-600 mb-1">Qty</label>
												<input
													type="number"
													value={item.quantity}
													on:input={(e) => updateEditItem(item.id, 'quantity', parseInt(e.currentTarget.value) || 0)}
													min="1"
													class="w-full px-3 py-2 text-sm border border-gray-200 rounded-md"
													disabled={isProcessing}
												/>
											</div>
											<div class="md:col-span-2">
												<label class="block text-xs font-medium text-gray-600 mb-1">Unit Price</label>
												<input
													type="number"
													value={formatNumberInput(item.unitPrice)}
													on:input={(e) => updateEditItem(item.id, 'unitPrice', parseNumberInput(e.currentTarget.value))}
													min="0"
													step="0.01"
													class="w-full px-3 py-2 text-sm border border-gray-200 rounded-md"
													disabled={isProcessing}
												/>
											</div>
											<div class="md:col-span-2">
												<label class="block text-xs font-medium text-gray-600 mb-1">Discount</label>
												<input
													type="number"
													value={formatNumberInput(item.discount)}
													on:input={(e) => updateEditItem(item.id, 'discount', parseNumberInput(e.currentTarget.value))}
													min="0"
													step="0.01"
													class="w-full px-3 py-2 text-sm border border-gray-200 rounded-md"
													disabled={isProcessing}
												/>
											</div>
											<div class="md:col-span-2 flex items-end justify-between gap-2">
												<div class="flex-1">
													<label class="block text-xs font-medium text-gray-600 mb-1">Subtotal</label>
													<div class="text-sm font-medium text-gray-900 py-2">
														{formatCurrency(editItemSubtotals.find(i => i.id === item.id)?.subtotal || 0, editCurrency)}
													</div>
												</div>
												<button
													type="button"
													on:click={() => removeEditItem(item.id)}
													disabled={isProcessing}
													class="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-md"
												>
													<Trash2 class="w-4 h-4" />
												</button>
											</div>
										</div>
									</div>
								{/each}
							</div>
							
							<div class="mt-4 pt-4 border-t border-gray-200 flex justify-between items-center">
								<span class="text-sm font-medium text-gray-600">Items Subtotal:</span>
								<span class="text-lg font-semibold text-gray-900">{formatCurrency(editSubtotal, editCurrency)}</span>
							</div>
						{/if}
					</div>
				</section>
				
				<!-- Fees Section -->
				<section class="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
					<div class="px-6 py-4 border-b border-gray-200 bg-gray-50 flex items-center justify-between">
						<h2 class="text-lg font-semibold text-gray-900">Fees & Taxes</h2>
						<button
							type="button"
							on:click={addEditFee}
							disabled={isProcessing}
							class="flex items-center gap-2 px-4 py-2 text-sm font-medium text-primary hover:bg-primary/10 rounded-lg transition-colors"
						>
							<Plus class="w-4 h-4" />
							Add Fee
						</button>
					</div>
					
					<div class="p-6">
						{#if editFees.length === 0}
							<p class="text-center py-4 text-gray-500">No additional fees.</p>
						{:else}
							<div class="space-y-3">
								{#each editFees as fee (fee.id)}
									<div class="grid grid-cols-1 md:grid-cols-12 gap-3 items-end">
										<div class="md:col-span-4">
											<input
												type="text"
												value={fee.label}
												on:input={(e) => updateEditFee(fee.id, 'label', e.currentTarget.value)}
												placeholder="Fee label"
												class="w-full px-3 py-2 text-sm border border-gray-200 rounded-md"
												disabled={isProcessing}
											/>
										</div>
										<div class="md:col-span-3">
											<select
												value={fee.type}
												on:change={(e) => updateEditFee(fee.id, 'type', e.currentTarget.value)}
												class="w-full px-3 py-2 text-sm border border-gray-200 rounded-md"
												disabled={isProcessing}
											>
												{#each feeTypes as type}
													<option value={type.value}>{type.label}</option>
												{/each}
											</select>
										</div>
										<div class="md:col-span-3">
											<input
												type="number"
												value={formatNumberInput(fee.amount)}
												on:input={(e) => updateEditFee(fee.id, 'amount', parseNumberInput(e.currentTarget.value))}
												min="0"
												step="0.01"
												class="w-full px-3 py-2 text-sm border border-gray-200 rounded-md"
												disabled={isProcessing}
											/>
										</div>
										<div class="md:col-span-2">
											<button
												type="button"
												on:click={() => removeEditFee(fee.id)}
												disabled={isProcessing}
												class="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-md"
											>
												<Trash2 class="w-4 h-4" />
											</button>
										</div>
									</div>
								{/each}
								
								<div class="mt-4 pt-3 border-t border-gray-200 flex justify-between items-center">
									<span class="text-sm font-medium text-gray-600">Total Fees:</span>
									<span class="font-semibold text-gray-900">{formatCurrency(editTotalFees, editCurrency)}</span>
								</div>
							</div>
						{/if}
					</div>
				</section>
				
				<!-- Edit Totals -->
				<section class="bg-primary/5 rounded-xl border-2 border-primary/20 p-6">
					<div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
						<div class="space-y-2">
							<div class="flex justify-between md:justify-start md:gap-8 text-sm">
								<span class="text-gray-600">Subtotal:</span>
								<span class="font-medium text-gray-900">{formatCurrency(editSubtotal, editCurrency)}</span>
							</div>
							<div class="flex justify-between md:justify-start md:gap-8 text-sm">
								<span class="text-gray-600">Fees & Taxes:</span>
								<span class="font-medium text-gray-900">{formatCurrency(editTotalFees, editCurrency)}</span>
							</div>
						</div>
						
						<div class="flex items-center gap-3">
							<span class="text-lg font-medium text-gray-700">Grand Total:</span>
							<span class="text-2xl md:text-3xl font-bold text-primary">
								{formatCurrency(editGrandTotal, editCurrency)}
							</span>
						</div>
					</div>
				</section>
				
				<!-- Edit Action Buttons -->
				<div class="flex flex-col sm:flex-row gap-4">
					<button
						type="submit"
						disabled={isProcessing}
						class="flex items-center justify-center gap-2 px-8 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors font-medium"
					>
						{#if isProcessing}
							<Loader2 class="w-5 h-5 animate-spin" />
							<span>Saving...</span>
						{:else}
							<Check class="w-5 h-5" />
							<span>Save Changes</span>
						{/if}
					</button>
					
					<button
						type="button"
						on:click={cancelEditMode}
						disabled={isProcessing}
						class="flex items-center justify-center gap-2 px-6 py-3 border border-gray-200 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors font-medium"
					>
						<X class="w-5 h-5" />
						<span>Cancel</span>
					</button>
				</div>
			</form>
		{:else}
			<!-- View Mode -->
			<div class="space-y-6">
				<!-- Receipt Info Card -->
				<section class="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
					<div class="p-6">
						<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
							<!-- Date -->
							<div class="flex items-start gap-3">
								<div class="w-10 h-10 rounded-lg bg-gray-100 flex items-center justify-center flex-shrink-0">
									<Calendar class="w-5 h-5 text-gray-500" />
								</div>
								<div>
									<p class="text-sm text-gray-500">Date</p>
									<p class="font-medium text-gray-900">{formatDate(receipt.receipt_date || receipt.created_at)}</p>
								</div>
							</div>
							
							<!-- Currency -->
							<div class="flex items-start gap-3">
								<div class="w-10 h-10 rounded-lg bg-gray-100 flex items-center justify-center flex-shrink-0">
									<Coins class="w-5 h-5 text-gray-500" />
								</div>
								<div>
									<p class="text-sm text-gray-500">Currency</p>
									<p class="font-medium text-gray-900">{receipt.currency}</p>
								</div>
							</div>
							
							<!-- Payment Method -->
							<div class="flex items-start gap-3">
								<div class="w-10 h-10 rounded-lg bg-gray-100 flex items-center justify-center flex-shrink-0">
									<CreditCard class="w-5 h-5 text-gray-500" />
								</div>
								<div>
									<p class="text-sm text-gray-500">Payment Method</p>
									<p class="font-medium text-gray-900 capitalize">{receipt.payment_method || 'Not specified'}</p>
								</div>
							</div>
							
							<!-- Total Amount -->
							<div class="flex items-start gap-3">
								<div class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
									<ReceiptIcon class="w-5 h-5 text-primary" />
								</div>
								<div>
									<p class="text-sm text-gray-500">Total Amount</p>
									<p class="text-xl font-bold text-primary">{formatCurrency(receipt.total, receipt.currency)}</p>
								</div>
							</div>
						</div>
						
						<!-- Notes -->
						{#if receipt.notes}
							<div class="mt-6 pt-6 border-t border-gray-100">
								<p class="text-sm text-gray-500 mb-1">Notes</p>
								<p class="text-gray-700 whitespace-pre-wrap">{receipt.notes}</p>
							</div>
						{/if}
					</div>
				</section>
				
				<!-- Image Section -->
				{#if receipt.image_url}
					<section class="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
						<div class="px-6 py-4 border-b border-gray-200 bg-gray-50">
							<div class="flex items-center gap-2">
								<ImageIcon class="w-5 h-5 text-primary" />
								<h2 class="text-lg font-semibold text-gray-900">Receipt Image</h2>
							</div>
						</div>
						<div class="p-6">
							<div class="relative">
								<img
									src={receipt.image_url}
									alt="Receipt"
									class="max-w-full max-h-96 object-contain rounded-lg border border-gray-200"
								/>
								
								<!-- AI Badge -->
								{#if receipt.source === 'ocr' && receipt.ocr_confidence !== undefined}
									<div class="absolute top-4 left-4">
										<span class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm font-medium {getConfidenceColor(receipt.ocr_confidence)}">
											<Bot class="w-4 h-4" />
											Parsed by AI ({Math.round(receipt.ocr_confidence * 100)}% confidence)
										</span>
									</div>
								{/if}
								
								<!-- Re-run OCR Button -->
								{#if receipt.source === 'ocr'}
									<button
										on:click={handleRerunOCR}
										class="absolute top-4 right-4 flex items-center gap-2 px-3 py-1.5 bg-white/90 backdrop-blur-sm text-gray-700 rounded-lg text-sm font-medium hover:bg-white transition-colors shadow-sm"
									>
										<RotateCcw class="w-4 h-4" />
										Re-run OCR
									</button>
								{/if}
							</div>
						</div>
					</section>
				{/if}
				
				<!-- Items Table -->
				<section class="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
					<div class="px-6 py-4 border-b border-gray-200 bg-gray-50">
						<h2 class="text-lg font-semibold text-gray-900">Items</h2>
					</div>
					<div class="overflow-x-auto">
						<table class="w-full">
							<thead class="bg-gray-50">
								<tr>
									<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
									<th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Qty</th>
									<th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Unit Price</th>
									<th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Discount</th>
									<th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Subtotal</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-gray-200">
								{#if receipt.items && receipt.items.length > 0}
									{#each receipt.items as item}
										<tr class="hover:bg-gray-50">
											<td class="px-6 py-4 text-sm font-medium text-gray-900">{item.name}</td>
											<td class="px-6 py-4 text-sm text-gray-600 text-right">{item.quantity}</td>
											<td class="px-6 py-4 text-sm text-gray-600 text-right">{formatCurrency(item.unit_price, receipt.currency)}</td>
											<td class="px-6 py-4 text-sm text-gray-600 text-right">
												{#if item.quantity * item.unit_price - item.total_price > 0}
													{formatCurrency(item.quantity * item.unit_price - item.total_price, receipt.currency)}
												{:else}
													-
												{/if}
											</td>
											<td class="px-6 py-4 text-sm font-medium text-gray-900 text-right">{formatCurrency(item.total_price, receipt.currency)}</td>
										</tr>
									{/each}
									<tr class="bg-gray-50 font-medium">
										<td colspan="4" class="px-6 py-3 text-sm text-gray-600 text-right">Items Subtotal:</td>
										<td class="px-6 py-3 text-sm font-semibold text-gray-900 text-right">{formatCurrency(receipt.subtotal || receipt.items.reduce((sum, item) => sum + item.total_price, 0), receipt.currency)}</td>
									</tr>
								{:else}
									<tr>
										<td colspan="5" class="px-6 py-8 text-center text-gray-500">
											No items on this receipt
										</td>
									</tr>
								{/if}
							</tbody>
						</table>
					</div>
				</section>
				
				<!-- Fees Section -->
				<section class="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
					<div class="px-6 py-4 border-b border-gray-200 bg-gray-50">
						<h2 class="text-lg font-semibold text-gray-900">Fees & Taxes</h2>
					</div>
					<div class="overflow-x-auto">
						<table class="w-full">
							<thead class="bg-gray-50">
								<tr>
									<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Label</th>
									<th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Type</th>
									<th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Amount</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-gray-200">
								{#if receipt.fees && receipt.fees.length > 0}
									{#each receipt.fees as fee}
										<tr class="hover:bg-gray-50">
											<td class="px-6 py-4 text-sm font-medium text-gray-900">{fee.name}</td>
											<td class="px-6 py-4 text-sm text-gray-600 capitalize">-</td>
											<td class="px-6 py-4 text-sm font-medium text-gray-900 text-right">{formatCurrency(fee.amount, receipt.currency)}</td>
										</tr>
									{/each}
									<tr class="bg-gray-50 font-medium">
										<td colspan="2" class="px-6 py-3 text-sm text-gray-600 text-right">Total Fees:</td>
										<td class="px-6 py-3 text-sm font-semibold text-gray-900 text-right">{formatCurrency(receipt.fees.reduce((sum, fee) => sum + fee.amount, 0), receipt.currency)}</td>
									</tr>
								{:else}
									<tr>
										<td colspan="3" class="px-6 py-8 text-center text-gray-500">
											No additional fees or taxes
										</td>
									</tr>
								{/if}
							</tbody>
						</table>
					</div>
				</section>
				
				<!-- Tags Section -->
				<section class="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
					<div class="px-6 py-4 border-b border-gray-200 bg-gray-50 flex items-center justify-between">
						<div class="flex items-center gap-2">
							<TagIcon class="w-5 h-5 text-primary" />
							<h2 class="text-lg font-semibold text-gray-900">Tags</h2>
						</div>
					</div>
					<div class="p-6">
						<div class="flex flex-wrap items-center gap-2">
							{#if receipt.tags && receipt.tags.length > 0}
								{#each receipt.tags as tagId}
									{@const tag = getTagById(tagId)}
									{#if tag}
										<span class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm font-medium border {getTagColorClass(tag.color)}">
											{tag.name}
											<button
												on:click={() => handleRemoveTag(tagId)}
												disabled={isProcessing}
												class="p-0.5 hover:bg-black/10 rounded-full transition-colors"
												aria-label="Remove tag"
											>
												<X class="w-3 h-3" />
											</button>
										</span>
									{/if}
								{/each}
							{:else}
								<span class="text-gray-500 text-sm">No tags assigned</span>
							{/if}
							
							<!-- Add Tag Dropdown -->
							<div class="relative" bind:this={tagDropdownRef}>
								<button
									on:click={() => showTagDropdown = !showTagDropdown}
									disabled={isProcessing || isLoadingTags}
									class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm font-medium bg-gray-100 text-gray-700 hover:bg-gray-200 transition-colors"
								>
									<Plus class="w-3.5 h-3.5" />
									Add tag
								</button>
								
								{#if showTagDropdown}
									<div class="absolute top-full left-0 mt-1 w-48 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-10">
										{#if isLoadingTags}
											<div class="px-4 py-2 text-sm text-gray-500 flex items-center gap-2">
												<Loader2 class="w-4 h-4 animate-spin" />
												Loading...
											</div>
										{:else if allTags.length === 0}
											<div class="px-4 py-2 text-sm text-gray-500">No tags available</div>
										{:else}
											{#each allTags.filter(t => !receipt?.tags.includes(t.id)) as tag}
												<button
													on:click={() => handleAddTag(tag.id)}
													class="w-full px-4 py-2 text-left text-sm hover:bg-gray-50 flex items-center gap-2"
												>
													<span class="w-3 h-3 rounded-full" style="background-color: {tag.color || '#6b7280'}"></span>
													{tag.name}
												</button>
											{/each}
											{#if allTags.filter(t => !receipt?.tags.includes(t.id)).length === 0}
												<div class="px-4 py-2 text-sm text-gray-500">All tags already added</div>
											{/if}
										{/if}
									</div>
								{/if}
							</div>
						</div>
					</div>
				</section>
				
				<!-- Totals Section -->
				<section class="bg-primary/5 rounded-xl border-2 border-primary/20 p-6">
					<div class="space-y-3">
						<div class="flex justify-between items-center text-sm">
							<span class="text-gray-600">Subtotal:</span>
							<span class="font-medium text-gray-900">
								{formatCurrency(receipt.subtotal || (receipt.items ? receipt.items.reduce((sum, item) => sum + item.total_price, 0) : 0), receipt.currency)}
							</span>
						</div>
						<div class="flex justify-between items-center text-sm">
							<span class="text-gray-600">Fees & Taxes:</span>
							<span class="font-medium text-gray-900">
								{formatCurrency(receipt.fees ? receipt.fees.reduce((sum, fee) => sum + fee.amount, 0) : 0, receipt.currency)}
							</span>
						</div>
						<div class="flex justify-between items-center pt-3 border-t border-primary/20">
							<span class="text-lg font-medium text-gray-900">Grand Total:</span>
							<span class="text-3xl font-bold text-primary">{formatCurrency(receipt.total, receipt.currency)}</span>
						</div>
					</div>
				</section>
				
				<!-- Action Buttons (Confirm/Reject) -->
				{#if receipt.status === 'pending_review'}
					<section class="flex flex-col sm:flex-row gap-4">
						<button
							on:click={handleConfirm}
							disabled={isProcessing}
							class="flex-1 flex items-center justify-center gap-2 px-6 py-3 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors font-medium"
						>
							{#if isProcessing}
								<Loader2 class="w-5 h-5 animate-spin" />
								<span>Processing...</span>
							{:else}
								<Check class="w-5 h-5" />
								<span>Confirm Receipt</span>
							{/if}
						</button>
						
						<button
							on:click={handleReject}
							disabled={isProcessing}
							class="flex-1 flex items-center justify-center gap-2 px-6 py-3 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors font-medium"
						>
							{#if isProcessing}
								<Loader2 class="w-5 h-5 animate-spin" />
								<span>Processing...</span>
							{:else}
								<X class="w-5 h-5" />
								<span>Reject Receipt</span>
							{/if}
						</button>
					</section>
				{/if}
			</div>
		{/if}
		
		<!-- Delete Confirmation Modal -->
		{#if showDeleteConfirm}
			<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
				<div class="bg-white rounded-xl shadow-xl max-w-md w-full p-6">
					<div class="flex items-center gap-3 mb-4">
						<div class="w-12 h-12 rounded-full bg-red-100 flex items-center justify-center">
							<AlertTriangle class="w-6 h-6 text-red-600" />
						</div>
						<h3 class="text-lg font-semibold text-gray-900">Delete Receipt?</h3>
					</div>
					<p class="text-gray-600 mb-6">
						Are you sure you want to delete this receipt? This action cannot be undone.
					</p>
					<div class="flex flex-col sm:flex-row gap-3">
						<button
							on:click={handleDelete}
							disabled={isProcessing}
							class="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors font-medium"
						>
							{#if isProcessing}
								<Loader2 class="w-4 h-4 animate-spin" />
							{:else}
								<Trash2 class="w-4 h-4" />
							{/if}
							<span>Delete</span>
						</button>
						<button
							on:click={() => showDeleteConfirm = false}
							disabled={isProcessing}
							class="flex-1 flex items-center justify-center gap-2 px-4 py-2 border border-gray-200 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors font-medium"
						>
							Cancel
						</button>
					</div>
				</div>
			</div>
		{/if}
	{/if}
</div>
