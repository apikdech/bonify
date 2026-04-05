// Global stores for Receipt Manager
import { writable, derived } from 'svelte/store';
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
