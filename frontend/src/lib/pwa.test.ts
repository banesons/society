import { describe, it, expect, vi, beforeEach } from 'vitest';
import { pwa } from './pwa.svelte';

describe('PWA Store and Lifecycle Manager', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	it('initializes with default properties', () => {
		expect(typeof pwa.isInstalled).toBe('boolean');
		expect(typeof pwa.canInstall).toBe('boolean');
	});

	it('returns false for promptInstall when no deferred prompt is captured', async () => {
		const result = await pwa.promptInstall();
		expect(result).toBe(false);
	});
});
