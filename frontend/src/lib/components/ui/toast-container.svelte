<script lang="ts">
	import { toast, type ToastItem } from '$lib/toast.svelte';
	import { CheckCircle2, Info, AlertTriangle, AlertCircle, X } from '@lucide/svelte';
	import { fly } from 'svelte/transition';

	function getIcon(type: ToastItem['type']) {
		switch (type) {
			case 'success':
				return CheckCircle2;
			case 'info':
				return Info;
			case 'warning':
				return AlertTriangle;
			case 'error':
				return AlertCircle;
		}
	}

	function getIconStyle(type: ToastItem['type']) {
		switch (type) {
			case 'success':
				return 'bg-emerald-50 text-emerald-700 border-emerald-200';
			case 'info':
				return 'bg-neutral-100 text-neutral-800 border-neutral-200';
			case 'warning':
				return 'bg-amber-50 text-amber-800 border-amber-200';
			case 'error':
				return 'bg-rose-50 text-rose-700 border-rose-200';
		}
	}
</script>

<!-- Global Toast Container (Fixed Non-Shifting Viewport Overlay) -->
<div
	class="pointer-events-none fixed top-4 right-4 z-50 flex max-w-sm flex-col gap-2.5 max-sm:inset-x-3 max-sm:top-3 sm:w-96"
	aria-live="polite"
	aria-atomic="false"
>
	{#each toast.toasts as item (item.id)}
		{@const Icon = getIcon(item.type)}
		<div
			in:fly={{ y: -10, duration: 200 }}
			out:fly={{ x: 20, duration: 150 }}
			role={item.type === 'error' || item.type === 'warning' ? 'alert' : 'status'}
			class="pointer-events-auto flex items-start gap-3 rounded-2xl border border-neutral-200/90 bg-white/95 p-3.5 shadow-lg backdrop-blur-md transition-all select-none"
		>
			<div
				class="flex h-7 w-7 shrink-0 items-center justify-center rounded-xl border {getIconStyle(
					item.type
				)}"
			>
				<Icon class="h-4 w-4" />
			</div>

			<div class="min-w-0 flex-1 pt-0.5">
				<p class="text-xs leading-relaxed font-medium text-neutral-900">
					{item.message}
				</p>
			</div>

			<button
				type="button"
				onclick={() => toast.dismiss(item.id)}
				class="rounded-full p-1 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-900 focus:ring-1 focus:ring-neutral-950 focus:outline-hidden"
				aria-label="Dismiss notification"
			>
				<X class="h-3.5 w-3.5" />
			</button>
		</div>
	{/each}
</div>
