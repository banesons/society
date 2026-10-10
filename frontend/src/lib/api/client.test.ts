import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ApiClient, ApiClientError } from './client';

describe('ApiClient', () => {
	let client: ApiClient;

	beforeEach(() => {
		vi.restoreAllMocks();
		client = new ApiClient('http://localhost:8080/api/v1');
	});

	it('unwraps data from standard success envelope', async () => {
		const mockPayload = { status: 'healthy', version: '1.0.0' };
		const mockResponse = {
			ok: true,
			status: 200,
			headers: new Headers({ 'X-Request-ID': 'test-req-id-123' }),
			json: async () => ({
				success: true,
				data: mockPayload,
				meta: null
			})
		};

		globalThis.fetch = vi.fn().mockResolvedValue(mockResponse);

		const result = await client.get('/health');
		expect(result).toEqual(mockPayload);
		expect(globalThis.fetch).toHaveBeenCalledTimes(1);
	});

	it('throws ApiClientError when backend returns error envelope', async () => {
		const mockResponse = {
			ok: false,
			status: 404,
			headers: new Headers({ 'X-Request-ID': 'err-req-id' }),
			json: async () => ({
				success: false,
				error: {
					code: 'NOT_FOUND',
					message: 'Resource not found'
				},
				request_id: 'err-req-id'
			})
		};

		globalThis.fetch = vi.fn().mockResolvedValue(mockResponse);

		await expect(client.get('/not-found')).rejects.toThrow(ApiClientError);

		try {
			await client.get('/not-found');
		} catch (err) {
			const apiErr = err as ApiClientError;
			expect(apiErr.code).toBe('NOT_FOUND');
			expect(apiErr.statusCode).toBe(404);
			expect(apiErr.requestId).toBe('err-req-id');
		}
	});

	it('attaches X-Request-ID header to outbound requests', async () => {
		const mockResponse = {
			ok: true,
			status: 200,
			headers: new Headers({ 'X-Request-ID': 'custom-id-999' }),
			json: async () => ({
				success: true,
				data: { ok: true }
			})
		};

		globalThis.fetch = vi.fn().mockResolvedValue(mockResponse);

		await client.get('/test', {
			headers: { 'X-Request-ID': 'custom-id-999' }
		});

		const fetchMock = globalThis.fetch as unknown as {
			mock: { calls: Array<[string, RequestInit]> };
		};
		const callArgs = fetchMock.mock.calls[0];
		const sentHeaders = callArgs[1].headers as Headers;
		expect(sentHeaders.get('X-Request-ID')).toBe('custom-id-999');
	});

	it('attaches Bearer Authorization header when access token is set', async () => {
		client.setAccessToken('test-jwt-token-123');

		const mockResponse = {
			ok: true,
			status: 200,
			headers: new Headers({ 'X-Request-ID': 'test-req' }),
			json: async () => ({
				success: true,
				data: { id: 'user-1' }
			})
		};

		globalThis.fetch = vi.fn().mockResolvedValue(mockResponse);

		await client.get('/auth/me');

		const fetchMock = globalThis.fetch as unknown as {
			mock: { calls: Array<[string, RequestInit]> };
		};
		const callArgs = fetchMock.mock.calls[0];
		const sentHeaders = callArgs[1].headers as Headers;
		expect(sentHeaders.get('Authorization')).toBe('Bearer test-jwt-token-123');
	});

	it('attempts token refresh on 401 and retries original request', async () => {
		client.setAccessToken('expired-token');

		const refreshHandler = vi.fn().mockResolvedValue('fresh-jwt-token-456');
		client.setRefreshHandler(refreshHandler);

		const expired401Response = {
			ok: false,
			status: 401,
			headers: new Headers({ 'X-Request-ID': 'req-401' }),
			json: async () => ({
				success: false,
				error: { code: 'TOKEN_EXPIRED', message: 'Token has expired' }
			})
		};

		const success200Response = {
			ok: true,
			status: 200,
			headers: new Headers({ 'X-Request-ID': 'req-200' }),
			json: async () => ({
				success: true,
				data: { id: 'user-refreshed' }
			})
		};

		globalThis.fetch = vi
			.fn()
			.mockResolvedValueOnce(expired401Response)
			.mockResolvedValueOnce(success200Response);

		const result = await client.get('/auth/me');

		expect(refreshHandler).toHaveBeenCalledTimes(1);
		expect(globalThis.fetch).toHaveBeenCalledTimes(2);
		expect(result).toEqual({ id: 'user-refreshed' });
	});
});
