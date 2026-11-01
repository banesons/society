export type ToastType = 'success' | 'info' | 'warning' | 'error';

export interface ToastItem {
	id: string;
	type: ToastType;
	message: string;
	durationMs: number;
	createdAt: number;
}

export interface ToastOptions {
	durationMs?: number;
}

const DEFAULT_DURATIONS: Record<ToastType, number> = {
	success: 4000,
	info: 5000,
	warning: 7000,
	error: 8000
};

const MAX_TOASTS = 5;

class ToastStore {
	toasts = $state<ToastItem[]>([]);
	private timers = new Map<string, ReturnType<typeof setTimeout>>();
	private idCounter = 0;

	private add(type: ToastType, message: string, options?: ToastOptions): string {
		const trimmed = message.trim();
		if (!trimmed) return '';

		// Prevent duplicate active toast with identical type and message
		const existing = this.toasts.find((t) => t.type === type && t.message === trimmed);
		if (existing) {
			return existing.id;
		}

		const id = `toast-${Date.now()}-${++this.idCounter}`;
		const durationMs = options?.durationMs ?? DEFAULT_DURATIONS[type];

		const item: ToastItem = {
			id,
			type,
			message: trimmed,
			durationMs,
			createdAt: Date.now()
		};

		// Limit maximum stacked toasts
		if (this.toasts.length >= MAX_TOASTS) {
			const oldest = this.toasts[0];
			this.dismiss(oldest.id);
		}

		this.toasts = [...this.toasts, item];

		if (durationMs > 0) {
			const timer = setTimeout(() => {
				this.dismiss(id);
			}, durationMs);
			this.timers.set(id, timer);
		}

		return id;
	}

	success(message: string, options?: ToastOptions): string {
		return this.add('success', message, options);
	}

	info(message: string, options?: ToastOptions): string {
		return this.add('info', message, options);
	}

	warning(message: string, options?: ToastOptions): string {
		return this.add('warning', message, options);
	}

	error(message: string, options?: ToastOptions): string {
		return this.add('error', message, options);
	}

	dismiss(id: string): void {
		const timer = this.timers.get(id);
		if (timer) {
			clearTimeout(timer);
			this.timers.delete(id);
		}
		this.toasts = this.toasts.filter((t) => t.id !== id);
	}

	clear(): void {
		for (const timer of this.timers.values()) {
			clearTimeout(timer);
		}
		this.timers.clear();
		this.toasts = [];
	}
}

export const toast = new ToastStore();
