import { api, ApiClientError } from '$lib/api/client';
import type { UserSocietyAssignment } from '$lib/api/society';
import { deactivateCurrentDevice } from '$lib/push';

export interface UserProfile {
	id: string;
	phone: string;
	status: string;
	platform_roles?: string[];
	society_assignments?: UserSocietyAssignment[];
}

export interface AuthVerifyResult {
	access_token: string;
	refresh_token: string;
	expires_in: number;
	user: UserProfile;
	session_replaced?: boolean;
}

export interface RefreshResult {
	access_token: string;
	refresh_token: string;
	expires_in: number;
}

export interface SendOTPResult {
	resend_after_seconds: number;
	otp?: string;
}

// In-memory reactive authentication state for SvelteKit 5
class AuthStore {
	user = $state<UserProfile | null>(null);
	isAuthenticated = $state<boolean>(false);
	isLoading = $state<boolean>(true);
	isLoginOpen = $state<boolean>(false);
	error = $state<string | null>(null);
	sessionReplaced = $state<boolean>(false);
	private refreshToken: string | null = null;

	get isSuperAdmin(): boolean {
		return Boolean(this.user?.platform_roles?.includes('SUPER_ADMIN'));
	}

	get isPlatformAdmin(): boolean {
		return Boolean(
			this.user?.platform_roles?.includes('SUPER_ADMIN') ||
			this.user?.platform_roles?.includes('ADMIN')
		);
	}

	get isSocietyAdmin(): boolean {
		return Boolean(this.user?.society_assignments && this.user.society_assignments.length > 0);
	}

	openLogin() {
		this.isLoginOpen = true;
		if (typeof window !== 'undefined' && window.location.pathname !== '/login') {
			window.location.href = '/login';
		}
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

	dismissSessionReplacedNotice() {
		this.sessionReplaced = false;
	}

	async sendOTP(phone: string): Promise<SendOTPResult> {
		this.error = null;
		const cleanPhone = phone.replace(/\D/g, '');
		const normalized = cleanPhone.length === 10 ? '91' + cleanPhone : cleanPhone;
		const res = await api.post<SendOTPResult>('/auth/otp/send', {
			phone: normalized
		});
		return res;
	}

	async verifyOTP(phone: string, otp: string): Promise<AuthVerifyResult> {
		this.error = null;
		this.isLoading = true;
		const cleanPhone = phone.replace(/\D/g, '');
		const normalized = cleanPhone.length === 10 ? '91' + cleanPhone : cleanPhone;
		try {
			const res = await api.post<AuthVerifyResult>('/auth/otp/verify', {
				phone: normalized,
				otp: otp.trim()
			});

			this.setTokens(res.access_token, res.refresh_token);
			this.user = res.user;
			this.sessionReplaced = Boolean(res.session_replaced);
			this.isAuthenticated = true;
			return res;
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
		} catch (err: unknown) {
			if (err instanceof ApiClientError && err.statusCode === 401) {
				this.clearSession();
			}
		}
	}

	async logout(): Promise<void> {
		try {
			await deactivateCurrentDevice();
		} catch {
			// Ignore device deactivation failure
		}
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
		this.sessionReplaced = false;
		if (typeof window !== 'undefined') {
			sessionStorage.removeItem('chs_refresh_token');
		}
	}
}

export const auth = new AuthStore();
