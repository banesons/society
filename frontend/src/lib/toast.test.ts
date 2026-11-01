import { describe, it, expect, beforeEach, vi } from 'vitest';
import { toast } from './toast.svelte';

describe('Toast System', () => {
	beforeEach(() => {
		toast.clear();
		vi.useRealTimers();
	});

	it('creates success, info, warning, and error toasts with correct types', () => {
		toast.success('Saved successfully');
		toast.info('Session note');
		toast.warning('Check inputs');
		toast.error('Failed to load');

		expect(toast.toasts).toHaveLength(4);
		expect(toast.toasts[0].type).toBe('success');
		expect(toast.toasts[0].message).toBe('Saved successfully');
		expect(toast.toasts[1].type).toBe('info');
		expect(toast.toasts[2].type).toBe('warning');
		expect(toast.toasts[3].type).toBe('error');
	});

	it('dismisses a toast by id', () => {
		const id = toast.info('To be dismissed');
		expect(toast.toasts).toHaveLength(1);

		toast.dismiss(id);
		expect(toast.toasts).toHaveLength(0);
	});

	it('clears all toasts', () => {
		toast.info('Toast 1');
		toast.info('Toast 2');
		expect(toast.toasts).toHaveLength(2);

		toast.clear();
		expect(toast.toasts).toHaveLength(0);
	});

	it('prevents adding exact duplicate active toasts', () => {
		const id1 = toast.info('Duplicate message');
		const id2 = toast.info('Duplicate message');

		expect(id1).toBe(id2);
		expect(toast.toasts).toHaveLength(1);
	});

	it('caps maximum active toasts to 5 by removing the oldest', () => {
		for (let i = 1; i <= 6; i++) {
			toast.info(`Message ${i}`);
		}

		expect(toast.toasts).toHaveLength(5);
		// Oldest ('Message 1') was evicted, newest ('Message 6') is present
		expect(toast.toasts.map((t) => t.message)).toEqual([
			'Message 2',
			'Message 3',
			'Message 4',
			'Message 5',
			'Message 6'
		]);
	});

	it('auto-dismisses toasts after duration', () => {
		vi.useFakeTimers();

		toast.success('Auto dismiss', { durationMs: 1000 });
		expect(toast.toasts).toHaveLength(1);

		vi.advanceTimersByTime(999);
		expect(toast.toasts).toHaveLength(1);

		vi.advanceTimersByTime(2);
		expect(toast.toasts).toHaveLength(0);
	});
});
