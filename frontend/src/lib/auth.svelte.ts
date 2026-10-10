import { api, ApiClientError } from '$lib/api/client';

export interface UserProfile {
	id: string;
	phone: string;
	status: string;
}

export interface AuthVerifyResult {
	access_token: string;
	refresh_token: string;
	expires_in: number;
	user: UserProfile;
}

export interface RefreshResult {
	access_token: string;
	refresh_token: string;
	expires_in: number;
}

// In-memory reactive authentication state for SvelteKit 5
class AuthStore {
	user = $state<UserProfile | null>(null);
	isAuthenticated = $state<boolean>(false);
	isLoading = $state<boolean>(true);
	isLoginOpen = $state<boolean>(false);
	error = $state<string | null>(null);
	private refreshToken: string | null = null;

	openLogin() {
		this.isLoginOpen = true;
	}

	closeLogin() {
		this.isLoginOpen = false;
	}

	constructor() {
		// Register transparent token refresh handler with central API client
		api.setRefreshHandler(() => this.refreshSession());
	}

	init() {
		if (typeof window !== 'undefined') {
			const saved = sessionStorage.getItem('chs_refresh_token');
			if (saved) {
				this.refreshToken = saved;
				this.refreshSession()
					.then((token) => {
						if (token) {
							return this.fetchCurrentUser();
						}
					})
					.catch(() => {
						this.clearSession();
					})
					.finally(() => {
						this.isLoading = false;
					});
				return;
			}
		}
		this.isLoading = false;
	}

	async sendOTP(phone: string): Promise<number> {
		this.error = null;
		const res = await api.post<{ resend_after_seconds: number }>('/auth/otp/send', {
			phone
		});
		return res.resend_after_seconds;
	}

	async verifyOTP(phone: string, otp: string): Promise<void> {
		this.error = null;
		this.isLoading = true;
		try {
			const res = await api.post<AuthVerifyResult>('/auth/otp/verify', {
				phone,
				otp
			});

			this.setTokens(res.access_token, res.refresh_token);
			this.user = res.user;
			this.isAuthenticated = true;
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				this.error = err.message;
			} else {
				this.error = 'Verification failed. Please try again.';
			}
			throw err;
		} finally {
			this.isLoading = false;
		}
	}

	async refreshSession(): Promise<string | null> {
		if (!this.refreshToken) {
			this.clearSession();
			return null;
		}

		try {
			const res = await api.post<RefreshResult>('/auth/refresh', {
				refresh_token: this.refreshToken
			});
			this.setTokens(res.access_token, res.refresh_token);
			return res.access_token;
		} catch {
			this.clearSession();
			return null;
		}
	}

	async fetchCurrentUser(): Promise<void> {
		try {
			const user = await api.get<UserProfile>('/auth/me');
			this.user = user;
			this.isAuthenticated = true;
		} catch {
			this.clearSession();
		}
	}

	async logout(): Promise<void> {
		try {
			await api.post('/auth/logout');
		} catch {
			// Ignore network failure on logout
		} finally {
			this.clearSession();
		}
	}

	async logoutAll(): Promise<void> {
		try {
			await api.post('/auth/logout-all');
		} catch {
			// Ignore network failure
		} finally {
			this.clearSession();
		}
	}

	private setTokens(accessToken: string, refreshToken: string) {
		api.setAccessToken(accessToken);
		this.refreshToken = refreshToken;
		if (typeof window !== 'undefined') {
			sessionStorage.setItem('chs_refresh_token', refreshToken);
		}
	}

	private clearSession() {
		api.setAccessToken(null);
		this.refreshToken = null;
		this.user = null;
		this.isAuthenticated = false;
		if (typeof window !== 'undefined') {
			sessionStorage.removeItem('chs_refresh_token');
		}
	}
}

export const auth = new AuthStore();
