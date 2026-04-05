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
}

// Export singleton instance
export const api = new APIClient();
export default api;
