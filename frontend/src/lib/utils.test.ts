import { describe, it, expect, vi } from 'vitest';
import { cn, formatBytes, triggerFileDownload } from './utils';

describe('Utils', () => {
	it('cn merges class names properly', () => {
		expect(cn('px-2 py-1', 'bg-white', false && 'hidden')).toBe('px-2 py-1 bg-white');
	});

	it('formatBytes formats file sizes correctly', () => {
		expect(formatBytes(0)).toBe('0 B');
		expect(formatBytes(512)).toBe('512 B');
		expect(formatBytes(1024)).toBe('1 KB');
		expect(formatBytes(1536)).toBe('1.5 KB');
		expect(formatBytes(1048576)).toBe('1 MB');
	});

	it('triggerFileDownload creates an anchor and triggers download in browser', () => {
		const mockBlob = new Blob(['sample data'], { type: 'application/pdf' });
		const createObjectURLMock = vi.fn().mockReturnValue('blob:http://localhost/test-uuid');
		const revokeObjectURLMock = vi.fn();
		const clickMock = vi.fn();
		const appendChildMock = vi.fn();
		const removeChildMock = vi.fn();

		const mockAnchor = {
			href: '',
			download: '',
			click: clickMock
		};

		const globalAny = globalThis as unknown as Record<string, unknown>;
		const originalWindow = globalAny.window;
		const originalDocument = globalAny.document;

		globalAny.window = {
			URL: {
				createObjectURL: createObjectURLMock,
				revokeObjectURL: revokeObjectURLMock
			}
		};

		globalAny.document = {
			createElement: vi.fn().mockReturnValue(mockAnchor),
			body: {
				appendChild: appendChildMock,
				removeChild: removeChildMock
			}
		};

		try {
			triggerFileDownload(mockBlob, 'test-bill.pdf');

			expect(createObjectURLMock).toHaveBeenCalledWith(mockBlob);
			expect(mockAnchor.download).toBe('test-bill.pdf');
			expect(clickMock).toHaveBeenCalled();
			expect(revokeObjectURLMock).toHaveBeenCalledWith('blob:http://localhost/test-uuid');
		} finally {
			globalAny.window = originalWindow;
			globalAny.document = originalDocument;
		}
	});

	it('triggerFileDownload safely noops in SSR / node env', () => {
		const mockBlob = new Blob(['sample data'], { type: 'application/pdf' });
		expect(() => triggerFileDownload(mockBlob, 'test.pdf')).not.toThrow();
	});
});
