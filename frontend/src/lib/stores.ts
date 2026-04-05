// Global stores for Receipt Manager
import { writable, derived, get } from 'svelte/store';
import { goto } from '$app/navigation';
import { page } from '$app/stores';
import api from './api';

// ============ Date Range Store ============

export interface DateRange {
	startDate: Date | null;
	endDate: Date | null;
}

function createDateRangeStore() {
	const { subscribe, set, update } = writable<DateRange>({
		startDate: null,
		endDate: null
	});

	return {
		subscribe,
		set,
		update,
		setRange: (startDate: Date | null, endDate: Date | null) => {
			set({ startDate, endDate });
		},
		setLastDays: (days: number) => {
			const endDate = new Date();
			const startDate = new Date();
			startDate.setDate(startDate.getDate() - days);
			set({ startDate, endDate });
		},
		setThisMonth: () => {
			const now = new Date();
			const startDate = new Date(now.getFullYear(), now.getMonth(), 1);
			const endDate = new Date(now.getFullYear(), now.getMonth() + 1, 0);
			set({ startDate, endDate });
		},
		setThisYear: () => {
			const now = new Date();
			const startDate = new Date(now.getFullYear(), 0, 1);
			const endDate = new Date(now.getFullYear(), 11, 31);
			set({ startDate, endDate });
		},
		clear: () => {
			set({ startDate: null, endDate: null });
		}
	};
}

export const dateRangeStore = createDateRangeStore();

// ============ Analytics Date Range Store with URL Sync ============

export type DateRangePreset = 'this_week' | 'this_month' | 'last_month' | 'this_quarter' | 'this_year' | 'custom';

export interface AnalyticsDateRange {
	from: Date;
	to: Date;
	preset: DateRangePreset;
}

function startOfDay(date: Date): Date {
	return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

function endOfDay(date: Date): Date {
	return new Date(date.getFullYear(), date.getMonth(), date.getDate(), 23, 59, 59, 999);
}

function startOfMonth(date: Date): Date {
	return new Date(date.getFullYear(), date.getMonth(), 1);
}

function endOfMonth(date: Date): Date {
	return new Date(date.getFullYear(), date.getMonth() + 1, 0, 23, 59, 59, 999);
}

function startOfQuarter(date: Date): Date {
	const quarter = Math.floor(date.getMonth() / 3);
	return new Date(date.getFullYear(), quarter * 3, 1);
}

function getPresetDates(preset: DateRangePreset): { from: Date; to: Date } {
	const now = new Date();
	
	switch (preset) {
		case 'this_week': {
			const dayOfWeek = now.getDay();
			const startOfWeek = new Date(now);
			startOfWeek.setDate(now.getDate() - dayOfWeek);
			return { from: startOfDay(startOfWeek), to: endOfDay(now) };
		}
		case 'this_month':
			return { from: startOfDay(startOfMonth(now)), to: endOfDay(now) };
		case 'last_month': {
			const lastMonth = new Date(now.getFullYear(), now.getMonth() - 1, 1);
			return { from: startOfDay(startOfMonth(lastMonth)), to: endOfDay(endOfMonth(lastMonth)) };
		}
		case 'this_quarter':
			return { from: startOfDay(startOfQuarter(now)), to: endOfDay(now) };
		case 'this_year':
			return { from: startOfDay(new Date(now.getFullYear(), 0, 1)), to: endOfDay(now) };
		default:
			return { from: startOfDay(startOfMonth(now)), to: endOfDay(now) };
	}
}

function createAnalyticsDateRangeStore() {
	const { subscribe, set, update } = writable<AnalyticsDateRange>({
		from: startOfMonth(new Date()),
		to: endOfDay(new Date()),
		preset: 'this_month'
	});

	let initialized = false;

	function updateURL(range: AnalyticsDateRange) {
		if (typeof window === 'undefined') return;
		
		const url = new URL(window.location.href);
		const fromStr = range.from.toISOString().split('T')[0];
		const toStr = range.to.toISOString().split('T')[0];
		
		url.searchParams.set('from', fromStr);
		url.searchParams.set('to', toStr);
		url.searchParams.set('preset', range.preset);
		
		goto(url.toString(), { replaceState: true, keepFocus: true });
	}

	return {
		subscribe,
		set: (range: AnalyticsDateRange) => {
			set(range);
			updateURL(range);
		},
		update,
		setPreset: (preset: DateRangePreset) => {
			if (preset === 'custom') {
				update(current => {
					const newRange = { ...current, preset };
					updateURL(newRange);
					return newRange;
				});
			} else {
				const { from, to } = getPresetDates(preset);
				const newRange = { from, to, preset };
				set(newRange);
				updateURL(newRange);
			}
		},
		setCustomRange: (from: Date, to: Date) => {
			const newRange = { from: startOfDay(from), to: endOfDay(to), preset: 'custom' as DateRangePreset };
			set(newRange);
			updateURL(newRange);
		},
		initializeFromURL: () => {
			if (initialized || typeof window === 'undefined') return;
			
			const url = new URL(window.location.href);
			const fromParam = url.searchParams.get('from');
			const toParam = url.searchParams.get('to');
			const presetParam = url.searchParams.get('preset') as DateRangePreset | null;
			
			if (fromParam && toParam) {
				const from = new Date(fromParam);
				const to = new Date(toParam);
				
				if (!isNaN(from.getTime()) && !isNaN(to.getTime())) {
					const preset = presetParam || 'custom';
					set({ from, to, preset });
				}
			} else if (presetParam && presetParam !== 'custom') {
				const { from, to } = getPresetDates(presetParam);
				set({ from, to, preset: presetParam });
			}
			
			initialized = true;
		},
		getQueryParams: () => {
			const state = get({ subscribe });
			return {
				from: state.from.toISOString(),
				to: state.to.toISOString()
			};
		}
	};
}

export const analyticsDateRangeStore = createAnalyticsDateRangeStore();

// ============ Pending Count Store ============

function createPendingCountStore() {
	const { subscribe, set, update } = writable<number>(0);

	let intervalId: ReturnType<typeof setInterval> | null = null;

	async function fetchCount() {
		try {
			const response = await api.receipts.list({
				status: 'pending_review',
				limit: 1
			});
			set(response.total);
		} catch {
			// Ignore errors, keep previous value
		}
	}

	return {
		subscribe,
		fetch: fetchCount,
		startPolling: (intervalMs: number = 30000) => {
			fetchCount();
			if (intervalId) clearInterval(intervalId);
			intervalId = setInterval(fetchCount, intervalMs);
		},
		stopPolling: () => {
			if (intervalId) {
				clearInterval(intervalId);
				intervalId = null;
			}
		},
		increment: () => update((n) => n + 1),
		decrement: () => update((n) => Math.max(0, n - 1)),
		set: (value: number) => set(value)
	};
}

export const pendingCountStore = createPendingCountStore();

// ============ Utility Stores ============

// Store for tracking mobile menu state
export const mobileMenuOpen = writable(false);

// Store for tracking toast notifications
export interface Toast {
	id: string;
	message: string;
	type: 'success' | 'error' | 'info' | 'warning';
	duration?: number;
}

function createToastStore() {
	const { subscribe, update } = writable<Toast[]>([]);

	let toastId = 0;

	function add(toast: Omit<Toast, 'id'>): string {
		const id = `toast-${++toastId}`;
		const newToast: Toast = { ...toast, id };

		update((toasts) => [...toasts, newToast]);

		// Auto-remove after duration
		const duration = toast.duration ?? 3000;
		setTimeout(() => {
			remove(id);
		}, duration);

		return id;
	}

	function remove(id: string) {
		update((toasts) => toasts.filter((t) => t.id !== id));
	}

	return {
		subscribe,
		add,
		remove,
		success: (message: string, duration?: number) =>
			add({ message, type: 'success', duration }),
		error: (message: string, duration?: number) =>
			add({ message, type: 'error', duration }),
		info: (message: string, duration?: number) =>
			add({ message, type: 'info', duration }),
		warning: (message: string, duration?: number) =>
			add({ message, type: 'warning', duration })
	};
}

export const toastStore = createToastStore();

// ============ Derived Stores ============

// Derived store to get formatted date range string
export const dateRangeString = derived(dateRangeStore, ($range) => {
	if (!$range.startDate || !$range.endDate) return 'All time';

	const format = (date: Date) =>
		date.toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		});

	return `${format($range.startDate)} - ${format($range.endDate)}`;
});
