import { env } from '$env/dynamic/public';

export interface ApiSuccessResponse<T> {
	success: true;
	data: T;
	meta?: Record<string, unknown> | null;
}

export interface ApiErrorDetail {
	code: string;
	message: string;
}

export interface ApiErrorResponse {
	success: false;
	error: ApiErrorDetail;
	request_id?: string;
}

export type ApiResponse<T> = ApiSuccessResponse<T> | ApiErrorResponse;

export class ApiClientError extends Error {
	code: string;
	statusCode: number;
	requestId?: string;

	constructor(message: string, code: string, statusCode: number, requestId?: string) {
		super(message);
		this.name = 'ApiClientError';
		this.code = code;
		this.statusCode = statusCode;
		this.requestId = requestId;
	}
}

export interface RequestOptions extends RequestInit {
	params?: Record<string, string | number | boolean | undefined>;
}

export class ApiClient {
	private baseUrl: string;
	private accessToken: string | null = null;
	private refreshHandler?: () => Promise<string | null>;

	constructor(baseUrl?: string) {
		this.baseUrl = baseUrl || env.PUBLIC_API_URL || 'http://localhost:8080/api/v1';
	}

	setAccessToken(token: string | null) {
		this.accessToken = token;
	}

	getAccessToken(): string | null {
		return this.accessToken;
	}

	setRefreshHandler(handler: () => Promise<string | null>) {
		this.refreshHandler = handler;
	}

	private generateRequestId(): string {
		if (typeof crypto !== 'undefined' && crypto.randomUUID) {
			return crypto.randomUUID();
		}
		return 'req-' + Math.random().toString(36).substring(2, 15);
	}

	async request<T>(path: string, options: RequestOptions = {}): Promise<T> {
		const cleanBase = this.baseUrl.replace(/\/+$/, '');
		const cleanPath = path.replace(/^\/+/, '');
		const url = new URL(`${cleanBase}/${cleanPath}`);

		if (options.params) {
			for (const [key, value] of Object.entries(options.params)) {
				if (value !== undefined) {
					url.searchParams.append(key, String(value));
				}
			}
		}

		const headers = new Headers(options.headers);
		if (!headers.has('Content-Type') && !(options.body instanceof FormData)) {
			headers.set('Content-Type', 'application/json');
		}

		if (this.accessToken && !headers.has('Authorization')) {
			headers.set('Authorization', `Bearer ${this.accessToken}`);
		}

		const requestId = headers.get('X-Request-ID') || this.generateRequestId();
		headers.set('X-Request-ID', requestId);

		let response: Response;
		try {
			response = await fetch(url.toString(), {
				...options,
				headers
			});
		} catch (networkErr: unknown) {
			const msg = networkErr instanceof Error ? networkErr.message : 'Network connection failed';
			throw new ApiClientError(msg, 'NETWORK_ERROR', 0, requestId);
		}

		// Handle transparent 401 token expiration refresh if refreshHandler is available
		if (
			response.status === 401 &&
			this.refreshHandler &&
			!path.includes('/auth/refresh') &&
			!path.includes('/auth/otp')
		) {
			const newAccessToken = await this.refreshHandler();
			if (newAccessToken) {
				headers.set('Authorization', `Bearer ${newAccessToken}`);
				response = await fetch(url.toString(), {
					...options,
					headers
				});
			}
		}

		const outboundRequestId = response.headers.get('X-Request-ID') || requestId;

		let json: ApiResponse<T>;
		try {
			json = (await response.json()) as ApiResponse<T>;
		} catch {
			throw new ApiClientError(
				`Server returned unexpected response (${response.status})`,
				'INVALID_JSON',
				response.status,
				outboundRequestId
			);
		}

		if (!response.ok || !json.success) {
			const errorResponse = json as ApiErrorResponse;
			const code = errorResponse.error?.code || 'UNKNOWN_ERROR';
			const message =
				errorResponse.error?.message || `Request failed with status ${response.status}`;
			throw new ApiClientError(message, code, response.status, outboundRequestId);
		}

		return (json as ApiSuccessResponse<T>).data;
	}

	get<T>(path: string, options?: RequestOptions): Promise<T> {
		return this.request<T>(path, { ...options, method: 'GET' });
	}

	post<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
		return this.request<T>(path, {
			...options,
			method: 'POST',
			body: body ? JSON.stringify(body) : undefined
		});
	}

	put<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
		return this.request<T>(path, {
			...options,
			method: 'PUT',
			body: body ? JSON.stringify(body) : undefined
		});
	}

	patch<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
		return this.request<T>(path, {
			...options,
			method: 'PATCH',
			body: body ? JSON.stringify(body) : undefined
		});
	}

	delete<T>(path: string, options?: RequestOptions): Promise<T> {
		return this.request<T>(path, { ...options, method: 'DELETE' });
	}

	async requestBlob(path: string, options: RequestOptions = {}): Promise<Blob> {
		const cleanBase = this.baseUrl.replace(/\/+$/, '');
		const cleanPath = path.replace(/^\/+/, '');
		const url = new URL(`${cleanBase}/${cleanPath}`);

		if (options.params) {
			for (const [key, value] of Object.entries(options.params)) {
				if (value !== undefined) {
					url.searchParams.append(key, String(value));
				}
			}
		}

		const headers = new Headers(options.headers);
		if (this.accessToken && !headers.has('Authorization')) {
			headers.set('Authorization', `Bearer ${this.accessToken}`);
		}
		const requestId = headers.get('X-Request-ID') || this.generateRequestId();
		headers.set('X-Request-ID', requestId);

		const response = await fetch(url.toString(), { ...options, headers });
		if (!response.ok) {
			let errorMsg = `Download failed with status ${response.status}`;
			try {
				const json = await response.json();
				if (json?.error?.message) errorMsg = json.error.message;
			} catch {
				// Non-json error
			}
			throw new ApiClientError(errorMsg, 'DOWNLOAD_FAILED', response.status, requestId);
		}
		return await response.blob();
	}
}

// Global singleton client instance
export const api = new ApiClient();
