import { api, ApiClientError } from '$lib/api/client';
import type { UserSocietyAssignment } from '$lib/api/society';
import { deactivateCurrentDevice } from '$lib/push';
import { toast } from '$lib/toast.svelte';

export interface UserProfile {
	id: string;
	phone: string;
	first_name?: string;
	last_name?: string;
	display_name?: string;
	email?: string;
	profile_photo_url?: string | null;
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
	networkError = $state<boolean>(false);
	private refreshToken: string | null = null;
	private refreshPromise: Promise<string | null> | null = null;

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

	private getSessionStorage(): Storage | null {
		if (typeof window !== 'undefined' && window.sessionStorage) {
			return window.sessionStorage;
		}
		if (typeof sessionStorage !== 'undefined') {
			return sessionStorage;
		}
		return null;
	}

	async init(): Promise<void> {
		const storage = this.getSessionStorage();
		if (!storage) {
			this.isLoading = false;
			return;
		}

		const saved = storage.getItem('chs_refresh_token');
		if (!saved) {
			this.isLoading = false;
			return;
		}

		this.refreshToken = saved;
		this.isLoading = true;
		this.networkError = false;

		try {
			const token = await this.refreshSession();
			if (token) {
				await this.fetchCurrentUser();
			}
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				if (err.code === 'NETWORK_ERROR' || err.statusCode === 0 || err.statusCode >= 500) {
					this.networkError = true;
				}
			}
		} finally {
			this.isLoading = false;
		}
	}

	dismissSessionReplacedNotice() {
		this.sessionReplaced = false;
	}

	updateProfile(data: Partial<UserProfile>) {
		if (this.user) {
			this.user = {
				...this.user,
				...data
			};
		}
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
		this.networkError = false;
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
			if (res.session_replaced) {
				toast.info('Signed in successfully. Your oldest session was signed out.');
			}
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
		// In-flight promise deduplication: ensure concurrent requests reuse the same refresh cycle
		if (this.refreshPromise) {
			return this.refreshPromise;
		}

		if (!this.refreshToken) {
			this.clearSession();
			return null;
		}

		this.refreshPromise = (async () => {
			try {
				const res = await api.post<RefreshResult>('/auth/refresh', {
					refresh_token: this.refreshToken
				});
				this.setTokens(res.access_token, res.refresh_token);
				this.networkError = false;
				return res.access_token;
			} catch (err: unknown) {
				if (err instanceof ApiClientError) {
					// Authoritative rejection: session revoked or token invalid/reused
					if (err.statusCode === 401) {
						const wasReplaced = err.code === 'SESSION_REVOKED';
						if (wasReplaced) {
							this.sessionReplaced = true;
							toast.warning(
								'Your session has ended because this account was logged into elsewhere.'
							);
						}
						this.clearSession(wasReplaced);
						return null;
					}
					// Network drop or temporary server error: DO NOT clear session
					if (err.code === 'NETWORK_ERROR' || err.statusCode === 0 || err.statusCode >= 500) {
						this.networkError = true;
						return null;
					}
				}
				this.clearSession();
				return null;
			} finally {
				this.refreshPromise = null;
			}
		})();

		return this.refreshPromise;
	}

	async fetchCurrentUser(): Promise<void> {
		try {
			const user = await api.get<UserProfile>('/auth/me');
			this.user = user;
			this.isAuthenticated = true;
			this.networkError = false;
		} catch (err: unknown) {
			if (err instanceof ApiClientError) {
				if (err.statusCode === 401) {
					this.clearSession();
				} else if (err.code === 'NETWORK_ERROR' || err.statusCode === 0 || err.statusCode >= 500) {
					this.networkError = true;
				}
			}
			throw err;
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
		const storage = this.getSessionStorage();
		if (storage) {
			storage.setItem('chs_refresh_token', refreshToken);
		}
	}

	private clearSession(preserveReplaced = false) {
		api.setAccessToken(null);
		this.refreshToken = null;
		this.user = null;
		this.isAuthenticated = false;
		if (!preserveReplaced) {
			this.sessionReplaced = false;
		}
		const storage = this.getSessionStorage();
		if (storage) {
			storage.removeItem('chs_refresh_token');
		}
	}
}

export const auth = new AuthStore();
