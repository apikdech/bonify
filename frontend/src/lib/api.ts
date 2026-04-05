// API Client for Receipt Manager
const API_BASE = '/api/v1';

// ============ Interfaces ============

export interface TokenPair {
	access_token: string;
	refresh_token: string;
}

export interface User {
	id: string;
	name: string;
	email: string;
	role: string;
	home_currency: string;
	notify_on_parse: boolean;
	notify_on_pending_review: boolean;
	notify_budget_alerts: boolean;
}

export interface ReceiptItem {
	id?: string;
	name: string;
	quantity: number;
	unit_price: number;
	total_price: number;
	category?: string;
}

export interface ReceiptFee {
	id?: string;
	name: string;
	amount: number;
}

export interface Tag {
	id: string;
	name: string;
	color?: string;
	user_id: string;
}

export interface Receipt {
	id: string;
	user_id: string;
	title: string;
	source: 'manual' | 'ocr';
	image_url?: string;
	ocr_confidence?: number;
	currency: string;
	payment_method?: string;
	subtotal?: number;
	total: number;
	status: 'pending_review' | 'confirmed' | 'rejected';
	notes?: string;
	receipt_date?: string;
	created_at: string;
	items: ReceiptItem[];
	fees: ReceiptFee[];
	tags: string[];
}

export interface CreateReceiptRequest {
	title: string;
	currency: string;
	total: number;
	receipt_date?: string;
	payment_method?: string;
	notes?: string;
	subtotal?: number;
	items?: ReceiptItem[];
	fees?: ReceiptFee[];
	tags?: string[];
	source?: 'manual' | 'ocr';
	image_url?: string;
}

export interface ReceiptSplitParticipant {
	id?: string;
	name: string;
	amount: number;
	paid: boolean;
}

export interface ReceiptSplit {
	id: string;
	receipt_id: string;
	split_type: 'even' | 'custom' | 'items';
	participants: ReceiptSplitParticipant[];
	created_at: string;
	updated_at: string;
}

export interface CreateSplitRequest {
	split_type: 'even' | 'custom' | 'items';
	participants: Omit<ReceiptSplitParticipant, 'id'>[];
}

export interface UpdateSplitRequest {
	split_type?: 'even' | 'custom' | 'items';
	participants?: Omit<ReceiptSplitParticipant, 'id'>[];
}

export interface UpdateReceiptRequest {
	title?: string;
	currency?: string;
	total?: number;
	receipt_date?: string;
	payment_method?: string;
	notes?: string;
	subtotal?: number;
	items?: ReceiptItem[];
	fees?: ReceiptFee[];
	tags?: string[];
}

export interface ReceiptListParams {
	page?: number;
	limit?: number;
	status?: 'pending_review' | 'confirmed' | 'rejected';
	start_date?: string;
	end_date?: string;
	from?: string;
	to?: string;
	sort_by?: 'created_at' | 'receipt_date' | 'total';
	sort_order?: 'asc' | 'desc';
	q?: string;
	search?: string;
	tag_id?: string;
	source?: 'manual' | 'ocr' | 'telegram' | 'discord' | 'api';
}

export interface ReceiptListResponse {
	data: Receipt[];
	total: number;
	page: number;
	limit: number;
}

export interface CreateTagRequest {
	name: string;
	color?: string;
}

export interface UpdateTagRequest {
	name?: string;
	color?: string;
}

// ============ Budget Interfaces ============

export interface Budget {
	id: string;
	user_id: string;
	tag_id: string | null;
	month: string;  // YYYY-MM format
	amount_limit: number;
}

export interface BudgetStatus {
	budget_id: string;
	tag_id: string | null;
	month: string;
	amount_limit: number;
	spent: number;
	percentage: number;
	remaining: number;
}

export interface CreateBudgetRequest {
	tag_id?: string | null;
	month: string;
	amount_limit: number;
}

export interface UpdateBudgetRequest {
	tag_id?: string | null;
	month?: string;
	amount_limit?: number;
}

export interface BudgetListParams {
	month?: string;
}

// ============ User Interfaces ============

export interface UpdateUserRequest {
	name?: string;
	email?: string;
	notify_on_parse?: boolean;
	notify_on_pending_review?: boolean;
	notify_budget_alerts?: boolean;
}

// ============ Analytics Interfaces ============

export interface AnalyticsSummary {
	total_spend: number;
	receipt_count: number;
	avg_per_receipt: number;
}

export interface AnalyticsSummaryResponse {
	data: AnalyticsSummary;
	warnings?: ConversionWarning[];
}

export interface ConversionWarning {
	currency: string;
	message: string;
}

export interface MonthData {
	month: string;
	total: number;
	count: number;
}

export interface MonthlyTrendsResponse {
	data: MonthData[];
	warnings?: ConversionWarning[];
}

export interface TagSpend {
	tag_id: string;
	name: string;
	color: string;
	total: number;
	count: number;
	percentage: number;
}

export interface ByTagResponse {
	data: TagSpend[];
	warnings?: ConversionWarning[];
}

export interface ShopSpend {
	name: string;
	total: number;
	visit_count: number;
	avg_ticket: number;
	last_visit: string;
}

export interface ByShopResponse {
	data: ShopSpend[];
	warnings?: ConversionWarning[];
}

export interface BiggestReceipt {
	id: string;
	title: string;
	total: number;
	date: string;
}

export interface MostVisitedShop {
	name: string;
	visits: number;
}

export interface MoMChange {
	percentage: number;
	absolute: number;
}

export interface BusiestDay {
	day: string;
	total: number;
}

export interface Insights {
	biggest_receipt: BiggestReceipt | null;
	most_visited_shop: MostVisitedShop | null;
	mom_change: MoMChange | null;
	busiest_day: BusiestDay | null;
}

export interface InsightsResponse {
	data: Insights;
	warnings?: ConversionWarning[];
}

export interface AnalyticsParams {
	from: string;
	to: string;
}

export interface MonthlyTrendsParams {
	months?: number;
}

// ============ API Client Class ============

class APIClient {
	private token: string | null = null;

	setToken(token: string | null): void {
		this.token = token;
	}

	private async fetch(endpoint: string, options: RequestInit = {}): Promise<Response> {
		const url = `${API_BASE}${endpoint}`;
		const headers: Record<string, string> = {
			'Content-Type': 'application/json',
			...((options.headers as Record<string, string>) || {})
		};

		if (this.token) {
			headers['Authorization'] = `Bearer ${this.token}`;
		}

		const response = await fetch(url, {
			...options,
			headers
		});

		if (!response.ok) {
			const errorData = await response.json().catch(() => ({}));
			const error = new Error(errorData.message || `HTTP ${response.status}: ${response.statusText}`);
			(error as Error & { status: number }).status = response.status;
			throw error;
		}

		return response;
	}

	// ============ Auth Methods ============

	auth = {
		login: async (email: string, password: string): Promise<TokenPair> => {
			const response = await this.fetch('/auth/login', {
				method: 'POST',
				body: JSON.stringify({ email, password })
			});
			return response.json();
		},

		refresh: async (refreshToken: string): Promise<TokenPair> => {
			const response = await this.fetch('/auth/refresh', {
				method: 'POST',
				body: JSON.stringify({ refresh_token: refreshToken })
			});
			return response.json();
		},

		logout: async (refreshToken: string): Promise<void> => {
			await this.fetch('/auth/logout', {
				method: 'POST',
				body: JSON.stringify({ refresh_token: refreshToken })
			});
		},

		me: async (): Promise<User> => {
			const response = await this.fetch('/auth/me', {
				method: 'GET'
			});
			return response.json();
		}
	};

	// ============ Receipts Methods ============

	receipts = {
		list: async (params: ReceiptListParams = {}): Promise<ReceiptListResponse> => {
			const queryParams = new URLSearchParams();
			if (params.page) queryParams.set('page', String(params.page));
			if (params.limit) queryParams.set('limit', String(params.limit));
			if (params.status) queryParams.set('status', params.status);
			if (params.start_date) queryParams.set('from', params.start_date);
			if (params.end_date) queryParams.set('to', params.end_date);
			if (params.from) queryParams.set('from', params.from);
			if (params.to) queryParams.set('to', params.to);
			if (params.sort_by) queryParams.set('sort_by', params.sort_by);
			if (params.sort_order) queryParams.set('sort_order', params.sort_order);
			if (params.q) queryParams.set('q', params.q);
			if (params.search) queryParams.set('q', params.search);
			if (params.tag_id) queryParams.set('tag_id', params.tag_id);
			if (params.source) queryParams.set('source', params.source);

			const query = queryParams.toString();
			const endpoint = query ? `/receipts?${query}` : '/receipts';
			const response = await this.fetch(endpoint, { method: 'GET' });
			return response.json();
		},

		get: async (id: string): Promise<Receipt> => {
			const response = await this.fetch(`/receipts/${id}`, { method: 'GET' });
			return response.json();
		},

		create: async (data: CreateReceiptRequest): Promise<Receipt> => {
			const response = await this.fetch('/receipts', {
				method: 'POST',
				body: JSON.stringify(data)
			});
			return response.json();
		},

		update: async (id: string, data: UpdateReceiptRequest): Promise<Receipt> => {
			const response = await this.fetch(`/receipts/${id}`, {
				method: 'PATCH',
				body: JSON.stringify(data)
			});
			return response.json();
		},

		delete: async (id: string): Promise<void> => {
			await this.fetch(`/receipts/${id}`, { method: 'DELETE' });
		},

		confirm: async (id: string): Promise<void> => {
			await this.fetch(`/receipts/${id}/confirm`, { method: 'PATCH' });
		},

		reject: async (id: string): Promise<void> => {
			await this.fetch(`/receipts/${id}/reject`, { method: 'PATCH' });
		},

		export: async (params: { from: string; to: string; format: string }): Promise<Blob> => {
			const queryParams = new URLSearchParams();
			queryParams.set('from', params.from);
			queryParams.set('to', params.to);
			queryParams.set('format', params.format);
			const response = await this.fetch(`/receipts/export?${queryParams}`, { method: 'GET' });
			return response.blob();
		},

		// ============ Splits Methods ============
		getSplits: async (id: string): Promise<ReceiptSplit> => {
			const response = await this.fetch(`/receipts/${id}/splits`, { method: 'GET' });
			return response.json();
		},

		createSplit: async (id: string, data: CreateSplitRequest): Promise<ReceiptSplit> => {
			const response = await this.fetch(`/receipts/${id}/splits`, {
				method: 'POST',
				body: JSON.stringify(data)
			});
			return response.json();
		},

		updateSplit: async (id: string, data: UpdateSplitRequest): Promise<ReceiptSplit> => {
			const response = await this.fetch(`/receipts/${id}/splits`, {
				method: 'PUT',
				body: JSON.stringify(data)
			});
			return response.json();
		},

		deleteSplit: async (id: string): Promise<void> => {
			await this.fetch(`/receipts/${id}/splits`, { method: 'DELETE' });
		}
	};

	// ============ Tags Methods ============

	tags = {
		list: async (): Promise<Tag[]> => {
			const response = await this.fetch('/tags', { method: 'GET' });
			return response.json();
		},

		create: async (data: CreateTagRequest): Promise<Tag> => {
			const response = await this.fetch('/tags', {
				method: 'POST',
				body: JSON.stringify(data)
			});
			return response.json();
		},

		update: async (id: string, data: UpdateTagRequest): Promise<Tag> => {
			const response = await this.fetch(`/tags/${id}`, {
				method: 'PATCH',
				body: JSON.stringify(data)
			});
			return response.json();
		},

		delete: async (id: string): Promise<void> => {
			await this.fetch(`/tags/${id}`, { method: 'DELETE' });
		}
	};

	// ============ Budget Methods ============

	budgets = {
		list: async (params: BudgetListParams = {}): Promise<Budget[]> => {
			const queryParams = new URLSearchParams();
			if (params.month) queryParams.set('month', params.month);
			
			const query = queryParams.toString();
			const endpoint = query ? `/budgets?${query}` : '/budgets';
			const response = await this.fetch(endpoint, { method: 'GET' });
			return response.json();
		},

		create: async (data: CreateBudgetRequest): Promise<Budget> => {
			const response = await this.fetch('/budgets', {
				method: 'POST',
				body: JSON.stringify(data)
			});
			return response.json();
		},

		update: async (id: string, data: UpdateBudgetRequest): Promise<Budget> => {
			const response = await this.fetch(`/budgets/${id}`, {
				method: 'PATCH',
				body: JSON.stringify(data)
			});
			return response.json();
		},

		delete: async (id: string): Promise<void> => {
			await this.fetch(`/budgets/${id}`, { method: 'DELETE' });
		},

		status: async (month: string): Promise<BudgetStatus[]> => {
			const queryParams = new URLSearchParams();
			queryParams.set('month', month);
			const response = await this.fetch(`/budgets/status?${queryParams}`, { method: 'GET' });
			return response.json();
		}
	};

	// ============ User Methods ============

	user = {
		update: async (data: UpdateUserRequest): Promise<User> => {
			const response = await this.fetch('/users/me', {
				method: 'PATCH',
				body: JSON.stringify(data)
			});
			return response.json();
		}
	};

	// ============ Analytics Methods ============

	analytics = {
		summary: async (params: AnalyticsParams): Promise<AnalyticsSummaryResponse> => {
			const queryParams = new URLSearchParams();
			queryParams.set('from', params.from);
			queryParams.set('to', params.to);
			const response = await this.fetch(`/analytics/summary?${queryParams}`, { method: 'GET' });
			return response.json();
		},

		monthly: async (params: MonthlyTrendsParams = {}): Promise<MonthlyTrendsResponse> => {
			const queryParams = new URLSearchParams();
			if (params.months) queryParams.set('months', String(params.months));
			const response = await this.fetch(`/analytics/monthly?${queryParams}`, { method: 'GET' });
			return response.json();
		},

		byTag: async (params: AnalyticsParams): Promise<ByTagResponse> => {
			const queryParams = new URLSearchParams();
			queryParams.set('from', params.from);
			queryParams.set('to', params.to);
			const response = await this.fetch(`/analytics/by-tag?${queryParams}`, { method: 'GET' });
			return response.json();
		},

		byShop: async (params: AnalyticsParams): Promise<ByShopResponse> => {
			const queryParams = new URLSearchParams();
			queryParams.set('from', params.from);
			queryParams.set('to', params.to);
			const response = await this.fetch(`/analytics/by-shop?${queryParams}`, { method: 'GET' });
			return response.json();
		},

		insights: async (params: AnalyticsParams): Promise<InsightsResponse> => {
			const queryParams = new URLSearchParams();
			queryParams.set('from', params.from);
			queryParams.set('to', params.to);
			const response = await this.fetch(`/analytics/insights?${queryParams}`, { method: 'GET' });
			return response.json();
		}
	};
}

// Export singleton instance
export const api = new APIClient();
export default api;
