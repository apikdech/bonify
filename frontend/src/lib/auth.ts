// Auth Store for Receipt Manager
import { writable, get } from 'svelte/store';
import { goto } from '$app/navigation';
import api, { type User, type TokenPair } from './api';

// ============ Types ============

export interface AuthState {
	user: User | null;
	accessToken: string | null;
	refreshToken: string | null;
	isAuthenticated: boolean;
	isLoading: boolean;
}

const STORAGE_KEYS = {
	accessToken: 'receipt_manager_access_token',
	refreshToken: 'receipt_manager_refresh_token',
	user: 'receipt_manager_user'
};

// ============ Auth Store Factory ============

export function createAuthStore() {
	// Initialize store with default state
	const { subscribe, set: _set, update } = writable<AuthState>({
		user: null,
		accessToken: null,
		refreshToken: null,
		isAuthenticated: false,
		isLoading: true
	});

	// ============ Helper Functions ============

	function saveToStorage(state: Partial<AuthState>): void {
		if (typeof localStorage === 'undefined') return;

		if (state.accessToken) {
			localStorage.setItem(STORAGE_KEYS.accessToken, state.accessToken);
		}
		if (state.refreshToken) {
			localStorage.setItem(STORAGE_KEYS.refreshToken, state.refreshToken);
		}
		if (state.user) {
			localStorage.setItem(STORAGE_KEYS.user, JSON.stringify(state.user));
		}
	}

	function clearStorage(): void {
		if (typeof localStorage === 'undefined') return;

		localStorage.removeItem(STORAGE_KEYS.accessToken);
		localStorage.removeItem(STORAGE_KEYS.refreshToken);
		localStorage.removeItem(STORAGE_KEYS.user);
	}

	function loadFromStorage(): Partial<AuthState> {
		if (typeof localStorage === 'undefined') return {};

		const accessToken = localStorage.getItem(STORAGE_KEYS.accessToken);
		const refreshToken = localStorage.getItem(STORAGE_KEYS.refreshToken);
		const userJson = localStorage.getItem(STORAGE_KEYS.user);

		const state: Partial<AuthState> = {};

		if (accessToken) state.accessToken = accessToken;
		if (refreshToken) state.refreshToken = refreshToken;
		if (userJson) {
			try {
				state.user = JSON.parse(userJson);
			} catch {
				// Invalid JSON, ignore
			}
		}

		return state;
	}

	function isTokenExpired(token: string): boolean {
		try {
			const base64Url = token.split('.')[1];
			const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
			const jsonPayload = decodeURIComponent(
				atob(base64)
					.split('')
					.map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
					.join('')
			);
			const { exp } = JSON.parse(jsonPayload);
			return exp * 1000 < Date.now();
		} catch {
			return true;
		}
	}

	async function tryRefreshToken(refreshToken: string): Promise<TokenPair | null> {
		try {
			const tokens = await api.auth.refresh(refreshToken);
			return tokens;
		} catch {
			return null;
		}
	}

	// ============ Public Methods ============

	async function checkAuth(): Promise<void> {
		update((state) => ({ ...state, isLoading: true }));

		const stored = loadFromStorage();

		if (!stored.accessToken || !stored.refreshToken) {
			update((state) => ({
				...state,
				isLoading: false,
				isAuthenticated: false
			}));
			return;
		}

		// Check if access token is expired
		let accessToken = stored.accessToken;
		let needsRefresh = isTokenExpired(accessToken);

		// Try to refresh if expired
		if (needsRefresh) {
			const newTokens = await tryRefreshToken(stored.refreshToken);
			if (newTokens) {
				accessToken = newTokens.access_token;
				saveToStorage({
					accessToken: newTokens.access_token,
					refreshToken: newTokens.refresh_token
				});
				api.setToken(accessToken);
			} else {
				// Refresh failed, clear storage
				clearStorage();
				update((state) => ({
					...state,
					user: null,
					accessToken: null,
					refreshToken: null,
					isAuthenticated: false,
					isLoading: false
				}));
				return;
			}
		}

		// Set the token on the API client
		api.setToken(accessToken);

		// Validate token by fetching user
		try {
			const user = await api.auth.me();
			saveToStorage({ user });
			update((state) => ({
				...state,
				user,
				accessToken,
				refreshToken: stored.refreshToken ?? null,
				isAuthenticated: true,
				isLoading: false
			}));
		} catch {
			// Token invalid, try refresh
			const newTokens = await tryRefreshToken(stored.refreshToken);
			if (newTokens) {
				api.setToken(newTokens.access_token);
				try {
					const user = await api.auth.me();
					saveToStorage({
						user,
						accessToken: newTokens.access_token,
						refreshToken: newTokens.refresh_token
					});
					update((state) => ({
						...state,
						user,
						accessToken: newTokens.access_token,
						refreshToken: newTokens.refresh_token,
						isAuthenticated: true,
						isLoading: false
					}));
				} catch {
					clearStorage();
					update((state) => ({
						...state,
						user: null,
						accessToken: null,
						refreshToken: null,
						isAuthenticated: false,
						isLoading: false
					}));
				}
			} else {
				clearStorage();
				update((state) => ({
					...state,
					user: null,
					accessToken: null,
					refreshToken: null,
					isAuthenticated: false,
					isLoading: false
				}));
			}
		}
	}

	async function login(email: string, password: string): Promise<boolean> {
		update((state) => ({ ...state, isLoading: true }));

		try {
			const tokens = await api.auth.login(email, password);
			api.setToken(tokens.access_token);

			const user = await api.auth.me();

			saveToStorage({
				user,
				accessToken: tokens.access_token,
				refreshToken: tokens.refresh_token
			});

			update((state) => ({
				...state,
				user,
				accessToken: tokens.access_token,
				refreshToken: tokens.refresh_token,
				isAuthenticated: true,
				isLoading: false
			}));

			await goto('/');
			return true;
		} catch (error) {
			update((state) => ({ ...state, isLoading: false }));
			throw error;
		}
	}

	async function logout(): Promise<void> {
		const currentState = get({ subscribe });

		if (currentState.refreshToken) {
			try {
				await api.auth.logout(currentState.refreshToken);
			} catch {
				// Ignore logout errors
			}
		}

		clearStorage();
		api.setToken(null);

		update((state) => ({
			...state,
			user: null,
			accessToken: null,
			refreshToken: null,
			isAuthenticated: false,
			isLoading: false
		}));

		await goto('/login');
	}

	function setLoading(loading: boolean): void {
		update((state) => ({ ...state, isLoading: loading }));
	}

	// Initialize checkAuth on creation (but only in browser)
	if (typeof window !== 'undefined') {
		checkAuth();
	}

	return {
		subscribe,
		checkAuth,
		login,
		logout,
		setLoading
	};
}

// Export singleton instance
export const auth = createAuthStore();
export default auth;
